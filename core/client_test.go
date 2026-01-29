package core

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// mockProvider is a test implementation of Provider.
type mockProvider struct {
	id          string
	chatFunc    func(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	streamFunc  func(ctx context.Context, req *ChatRequest) (*ChatStream, error)
	callCount   int
	lastRequest *ChatRequest
	mu          sync.Mutex
}

func (m *mockProvider) ID() string {
	return m.id
}

func (m *mockProvider) Models() []ModelInfo {
	return []ModelInfo{
		{ID: "mock-model", DisplayName: "Mock Model", Capabilities: []Feature{FeatureChat, FeatureChatStreaming}},
	}
}

func (m *mockProvider) Supports(feature Feature) bool {
	return feature == FeatureChat || feature == FeatureChatStreaming
}

func (m *mockProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	m.mu.Lock()
	m.callCount++
	m.lastRequest = req
	m.mu.Unlock()

	if m.chatFunc != nil {
		return m.chatFunc(ctx, req)
	}
	return &ChatResponse{
		ID:     "resp-1",
		Model:  req.Model,
		Output: "Hello!",
		Usage:  TokenUsage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
	}, nil
}

func (m *mockProvider) StreamChat(ctx context.Context, req *ChatRequest) (*ChatStream, error) {
	m.mu.Lock()
	m.callCount++
	m.lastRequest = req
	m.mu.Unlock()

	if m.streamFunc != nil {
		return m.streamFunc(ctx, req)
	}

	ch := make(chan ChatChunk, 1)
	errCh := make(chan error, 1)
	finalCh := make(chan *ChatResponse, 1)

	go func() {
		ch <- ChatChunk{Delta: "Hello"}
		close(ch)
		finalCh <- &ChatResponse{
			ID:     "resp-1",
			Model:  req.Model,
			Output: "Hello!",
			Usage:  TokenUsage{TotalTokens: 15},
		}
		close(finalCh)
		close(errCh)
	}()

	return &ChatStream{Ch: ch, Err: errCh, Final: finalCh}, nil
}

// mockTelemetryHook records telemetry events for testing.
type mockTelemetryHook struct {
	startEvents []RequestStartEvent
	endEvents   []RequestEndEvent
	mu          sync.Mutex
}

func (h *mockTelemetryHook) OnRequestStart(e RequestStartEvent) {
	h.mu.Lock()
	h.startEvents = append(h.startEvents, e)
	h.mu.Unlock()
}

func (h *mockTelemetryHook) OnRequestEnd(e RequestEndEvent) {
	h.mu.Lock()
	h.endEvents = append(h.endEvents, e)
	h.mu.Unlock()
}

func TestNewClient(t *testing.T) {
	p := &mockProvider{id: "test"}
	c := NewClient(p)

	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	if c.provider != p {
		t.Error("provider not set correctly")
	}
}

func TestNewClientWithOptions(t *testing.T) {
	p := &mockProvider{id: "test"}
	hook := &mockTelemetryHook{}
	retry := NewRetryPolicy(RetryConfig{MaxRetries: 5})

	c := NewClient(p,
		WithTelemetry(hook),
		WithRetryPolicy(retry),
	)

	if c.telemetry != hook {
		t.Error("telemetry hook not set")
	}
	if c.retry != retry {
		t.Error("retry policy not set")
	}
}

func TestChatBuilderFluentAPI(t *testing.T) {
	p := &mockProvider{id: "test"}
	c := NewClient(p)

	builder := c.Chat("gpt-4").
		System("You are helpful").
		User("Hello").
		Assistant("Hi there").
		User("How are you?").
		Temperature(0.7).
		MaxTokens(100)

	if builder.req.Model != "gpt-4" {
		t.Errorf("Model = %v, want gpt-4", builder.req.Model)
	}
	if len(builder.req.Messages) != 4 {
		t.Errorf("len(Messages) = %d, want 4", len(builder.req.Messages))
	}
	if *builder.req.Temperature != 0.7 {
		t.Errorf("Temperature = %v, want 0.7", *builder.req.Temperature)
	}
	if *builder.req.MaxTokens != 100 {
		t.Errorf("MaxTokens = %v, want 100", *builder.req.MaxTokens)
	}
}

func TestChatBuilderMessageOrder(t *testing.T) {
	p := &mockProvider{id: "test"}
	c := NewClient(p)

	builder := c.Chat("gpt-4").
		System("System").
		User("User1").
		Assistant("Assistant1").
		User("User2")

	expected := []struct {
		role    Role
		content string
	}{
		{RoleSystem, "System"},
		{RoleUser, "User1"},
		{RoleAssistant, "Assistant1"},
		{RoleUser, "User2"},
	}

	if len(builder.req.Messages) != len(expected) {
		t.Fatalf("len(Messages) = %d, want %d", len(builder.req.Messages), len(expected))
	}

	for i, exp := range expected {
		msg := builder.req.Messages[i]
		if msg.Role != exp.role {
			t.Errorf("Messages[%d].Role = %v, want %v", i, msg.Role, exp.role)
		}
		if msg.Content != exp.content {
			t.Errorf("Messages[%d].Content = %v, want %v", i, msg.Content, exp.content)
		}
	}
}

func TestGetResponseValidationModelRequired(t *testing.T) {
	p := &mockProvider{id: "test"}
	c := NewClient(p)

	_, err := c.Chat(""). // empty model
				User("Hello").
				GetResponse(context.Background())

	if !errors.Is(err, ErrModelRequired) {
		t.Errorf("err = %v, want ErrModelRequired", err)
	}
}

func TestGetResponseValidationNoMessages(t *testing.T) {
	p := &mockProvider{id: "test"}
	c := NewClient(p)

	_, err := c.Chat("gpt-4"). // no messages
					GetResponse(context.Background())

	if !errors.Is(err, ErrNoMessages) {
		t.Errorf("err = %v, want ErrNoMessages", err)
	}
}

func TestGetResponseSuccess(t *testing.T) {
	p := &mockProvider{id: "test"}
	c := NewClient(p)

	resp, err := c.Chat("gpt-4").
		User("Hello").
		GetResponse(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("response is nil")
	}
	if resp.Output != "Hello!" {
		t.Errorf("Output = %v, want Hello!", resp.Output)
	}
}

func TestGetResponseTelemetry(t *testing.T) {
	p := &mockProvider{id: "test-provider"}
	hook := &mockTelemetryHook{}
	c := NewClient(p, WithTelemetry(hook))

	_, err := c.Chat("gpt-4").
		User("Hello").
		GetResponse(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(hook.startEvents) != 1 {
		t.Errorf("expected 1 start event, got %d", len(hook.startEvents))
	}
	if len(hook.endEvents) != 1 {
		t.Errorf("expected 1 end event, got %d", len(hook.endEvents))
	}

	if hook.startEvents[0].Provider != "test-provider" {
		t.Error("start event should have correct provider")
	}
	if hook.endEvents[0].Provider != "test-provider" {
		t.Error("end event should have correct provider")
	}
	if hook.endEvents[0].Err != nil {
		t.Error("end event should have nil error on success")
	}
}

func TestGetResponseRetryOnRetryableError(t *testing.T) {
	callCount := 0
	p := &mockProvider{
		id: "test",
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			callCount++
			if callCount < 3 {
				return nil, ErrNetwork // retryable
			}
			return &ChatResponse{Output: "Success"}, nil
		},
	}

	// Use fast retry for testing
	retry := NewRetryPolicy(RetryConfig{
		MaxRetries: 5,
		BaseDelay:  time.Millisecond,
		MaxDelay:   10 * time.Millisecond,
		Jitter:     0,
	})
	c := NewClient(p, WithRetryPolicy(retry))

	resp, err := c.Chat("gpt-4").
		User("Hello").
		GetResponse(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 3 {
		t.Errorf("callCount = %d, want 3 (initial + 2 retries)", callCount)
	}
	if resp.Output != "Success" {
		t.Errorf("Output = %v, want Success", resp.Output)
	}
}

func TestGetResponseNoRetryOnNonRetryableError(t *testing.T) {
	callCount := 0
	p := &mockProvider{
		id: "test",
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			callCount++
			return nil, ErrUnauthorized // not retryable
		},
	}

	c := NewClient(p)

	_, err := c.Chat("gpt-4").
		User("Hello").
		GetResponse(context.Background())

	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("err = %v, want ErrUnauthorized", err)
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (no retries)", callCount)
	}
}

func TestGetResponseContextCancellation(t *testing.T) {
	p := &mockProvider{
		id: "test",
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			return nil, ErrNetwork // would retry, but context cancelled
		},
	}

	retry := NewRetryPolicy(RetryConfig{
		MaxRetries: 5,
		BaseDelay:  time.Second, // long delay
	})
	c := NewClient(p, WithRetryPolicy(retry))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := c.Chat("gpt-4").
		User("Hello").
		GetResponse(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestStreamValidation(t *testing.T) {
	p := &mockProvider{id: "test"}
	c := NewClient(p)

	// No model
	_, err := c.Chat("").User("Hello").Stream(context.Background())
	if !errors.Is(err, ErrModelRequired) {
		t.Errorf("err = %v, want ErrModelRequired", err)
	}

	// No messages
	_, err = c.Chat("gpt-4").Stream(context.Background())
	if !errors.Is(err, ErrNoMessages) {
		t.Errorf("err = %v, want ErrNoMessages", err)
	}
}

func TestStreamSuccess(t *testing.T) {
	p := &mockProvider{id: "test"}
	c := NewClient(p)

	stream, err := c.Chat("gpt-4").
		User("Hello").
		Stream(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stream == nil {
		t.Fatal("stream is nil")
	}

	// Read chunks
	for chunk := range stream.Ch {
		if chunk.Delta != "Hello" {
			t.Errorf("Delta = %v, want Hello", chunk.Delta)
		}
	}
}

func TestStreamTelemetry(t *testing.T) {
	p := &mockProvider{id: "test-provider"}
	hook := &mockTelemetryHook{}
	c := NewClient(p, WithTelemetry(hook))

	stream, err := c.Chat("gpt-4").
		User("Hello").
		Stream(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Start event should be immediate
	if len(hook.startEvents) != 1 {
		t.Errorf("expected 1 start event, got %d", len(hook.startEvents))
	}

	// Drain the stream to trigger end event
	for range stream.Ch {
	}
	<-stream.Final

	// Give goroutine time to emit end event
	time.Sleep(10 * time.Millisecond)

	hook.mu.Lock()
	endCount := len(hook.endEvents)
	hook.mu.Unlock()

	if endCount != 1 {
		t.Errorf("expected 1 end event, got %d", endCount)
	}
}

func TestClientConcurrentUse(t *testing.T) {
	p := &mockProvider{id: "test"}
	c := NewClient(p)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.Chat("gpt-4").
				User("Hello").
				GetResponse(context.Background())
			if err != nil {
				t.Errorf("concurrent call failed: %v", err)
			}
		}()
	}
	wg.Wait()

	p.mu.Lock()
	count := p.callCount
	p.mu.Unlock()

	if count != 10 {
		t.Errorf("callCount = %d, want 10", count)
	}
}

func TestChatBuilderTools(t *testing.T) {
	p := &mockProvider{id: "test"}
	c := NewClient(p)

	// Create mock tools
	tool1 := &mockTool{name: "tool1"}
	tool2 := &mockTool{name: "tool2"}

	builder := c.Chat("gpt-4").
		User("Hello").
		Tools(tool1, tool2)

	if len(builder.req.Tools) != 2 {
		t.Errorf("len(Tools) = %d, want 2", len(builder.req.Tools))
	}
}

// mockTool is a test implementation of Tool.
type mockTool struct {
	name string
}

func (t *mockTool) Name() string        { return t.name }
func (t *mockTool) Description() string { return "mock tool" }

func TestImageGeneratorInterface(t *testing.T) {
	// Verify the interface is defined
	var _ ImageGenerator = (*mockImageGenerator)(nil)
}

type mockImageGenerator struct{}

func (m *mockImageGenerator) GenerateImage(ctx context.Context, req *ImageGenerateRequest) (*ImageResponse, error) {
	return nil, nil
}

func (m *mockImageGenerator) EditImage(ctx context.Context, req *ImageEditRequest) (*ImageResponse, error) {
	return nil, nil
}

func (m *mockImageGenerator) StreamImage(ctx context.Context, req *ImageGenerateRequest) (*ImageStream, error) {
	return nil, nil
}

func TestFileSearchWithVectorStoreIDs(t *testing.T) {
	p := &mockProvider{id: "test"}
	client := NewClient(p)

	builder := client.Chat("gpt-4.1-mini").
		User("Search my docs").
		FileSearch("vs_abc123", "vs_def456")

	// Access the internal request to verify
	if builder.req.ToolResources == nil {
		t.Fatal("expected ToolResources to be set")
	}
	if builder.req.ToolResources.FileSearch == nil {
		t.Fatal("expected FileSearch resources to be set")
	}
	if len(builder.req.ToolResources.FileSearch.VectorStoreIDs) != 2 {
		t.Errorf("expected 2 vector store IDs, got %d", len(builder.req.ToolResources.FileSearch.VectorStoreIDs))
	}
	if builder.req.ToolResources.FileSearch.VectorStoreIDs[0] != "vs_abc123" {
		t.Errorf("expected vs_abc123, got %s", builder.req.ToolResources.FileSearch.VectorStoreIDs[0])
	}

	// Verify file_search tool was added
	found := false
	for _, tool := range builder.req.BuiltInTools {
		if tool.Type == "file_search" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected file_search tool to be added")
	}
}

func TestFileSearchWithoutVectorStoreIDs(t *testing.T) {
	p := &mockProvider{id: "test"}
	client := NewClient(p)

	builder := client.Chat("gpt-4.1-mini").
		User("Search").
		FileSearch()

	// Should add the tool but no resources
	found := false
	for _, tool := range builder.req.BuiltInTools {
		if tool.Type == "file_search" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected file_search tool to be added")
	}
	if builder.req.ToolResources != nil {
		t.Error("expected no ToolResources when no vector store IDs provided")
	}
}
