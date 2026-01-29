package core

import (
	"context"
	"time"
)

// Provider is the interface that LLM providers must implement.
// Providers SHOULD be safe for concurrent calls.
// If a provider cannot be concurrent-safe, it MUST document this.
type Provider interface {
	// ID returns the provider identifier (e.g., "openai", "anthropic").
	ID() string

	// Models returns the list of models available from this provider.
	Models() []ModelInfo

	// Supports reports whether the provider supports the given feature.
	Supports(feature Feature) bool

	// Chat sends a non-streaming chat request.
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// StreamChat sends a streaming chat request.
	StreamChat(ctx context.Context, req *ChatRequest) (*ChatStream, error)
}

// ImageGenerator is an optional interface for providers that support image generation.
type ImageGenerator interface {
	// GenerateImage generates images from a text prompt.
	GenerateImage(ctx context.Context, req *ImageGenerateRequest) (*ImageResponse, error)

	// EditImage edits existing images using a prompt and optional mask.
	EditImage(ctx context.Context, req *ImageEditRequest) (*ImageResponse, error)

	// StreamImage generates images with streaming partial results.
	// Not all providers support streaming.
	StreamImage(ctx context.Context, req *ImageGenerateRequest) (*ImageStream, error)
}

// Client is the main entry point for interacting with LLM providers.
// Client is safe for concurrent use.
type Client struct {
	provider  Provider
	telemetry TelemetryHook
	retry     RetryPolicy
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// NewClient creates a new Client with the given provider and options.
func NewClient(p Provider, opts ...ClientOption) *Client {
	c := &Client{
		provider:  p,
		telemetry: NoopTelemetryHook{},
		retry:     DefaultRetryPolicy(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithTelemetry sets the telemetry hook for the client.
func WithTelemetry(h TelemetryHook) ClientOption {
	return func(c *Client) {
		if h != nil {
			c.telemetry = h
		}
	}
}

// WithRetryPolicy sets the retry policy for the client.
func WithRetryPolicy(r RetryPolicy) ClientOption {
	return func(c *Client) {
		if r != nil {
			c.retry = r
		}
	}
}

// Provider returns the underlying provider.
func (c *Client) Provider() Provider {
	return c.provider
}

// Chat returns a ChatBuilder for constructing and executing a chat request.
func (c *Client) Chat(model ModelID) *ChatBuilder {
	return &ChatBuilder{
		client: c,
		req: ChatRequest{
			Model: model,
		},
	}
}

// ChatBuilder provides a fluent API for building chat requests.
// ChatBuilder is NOT thread-safe and should not be shared across goroutines.
type ChatBuilder struct {
	client *Client
	req    ChatRequest
}

// System appends a system message.
func (b *ChatBuilder) System(s string) *ChatBuilder {
	b.req.Messages = append(b.req.Messages, Message{Role: RoleSystem, Content: s})
	return b
}

// User appends a user message.
func (b *ChatBuilder) User(s string) *ChatBuilder {
	b.req.Messages = append(b.req.Messages, Message{Role: RoleUser, Content: s})
	return b
}

// Assistant appends an assistant message.
func (b *ChatBuilder) Assistant(s string) *ChatBuilder {
	b.req.Messages = append(b.req.Messages, Message{Role: RoleAssistant, Content: s})
	return b
}

// Temperature sets the temperature parameter.
func (b *ChatBuilder) Temperature(v float32) *ChatBuilder {
	b.req.Temperature = &v
	return b
}

// MaxTokens sets the maximum tokens parameter.
func (b *ChatBuilder) MaxTokens(n int) *ChatBuilder {
	b.req.MaxTokens = &n
	return b
}

// Tools sets the tools available for the request.
func (b *ChatBuilder) Tools(ts ...Tool) *ChatBuilder {
	b.req.Tools = ts
	return b
}

// Instructions sets the system instructions (Responses API style).
// For Chat Completions API, this is equivalent to adding a system message.
func (b *ChatBuilder) Instructions(s string) *ChatBuilder {
	b.req.Instructions = s
	return b
}

// ReasoningEffort sets the reasoning effort level for models that support it.
func (b *ChatBuilder) ReasoningEffort(level ReasoningEffort) *ChatBuilder {
	b.req.ReasoningEffort = level
	return b
}

// BuiltInTool adds a built-in tool to the request.
func (b *ChatBuilder) BuiltInTool(toolType string) *ChatBuilder {
	b.req.BuiltInTools = append(b.req.BuiltInTools, BuiltInTool{Type: toolType})
	return b
}

// WebSearch adds the web_search built-in tool.
func (b *ChatBuilder) WebSearch() *ChatBuilder {
	return b.BuiltInTool("web_search")
}

// FileSearch adds the file_search built-in tool.
func (b *ChatBuilder) FileSearch() *ChatBuilder {
	return b.BuiltInTool("file_search")
}

// CodeInterpreter adds the code_interpreter built-in tool.
func (b *ChatBuilder) CodeInterpreter() *ChatBuilder {
	return b.BuiltInTool("code_interpreter")
}

// ContinueFrom chains this request to a previous response.
func (b *ChatBuilder) ContinueFrom(responseID string) *ChatBuilder {
	b.req.PreviousResponseID = responseID
	return b
}

// Truncation sets the truncation mode for the request.
func (b *ChatBuilder) Truncation(mode string) *ChatBuilder {
	b.req.Truncation = mode
	return b
}

// validate checks that the request is valid.
func (b *ChatBuilder) validate() error {
	if b.req.Model == "" {
		return ErrModelRequired
	}
	if len(b.req.Messages) == 0 {
		return ErrNoMessages
	}
	return nil
}

// GetResponse executes the chat request and returns the response.
// It applies validation, telemetry, and retry logic.
func (b *ChatBuilder) GetResponse(ctx context.Context) (*ChatResponse, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}

	start := time.Now()
	providerID := b.client.provider.ID()

	// Emit telemetry start
	b.client.telemetry.OnRequestStart(RequestStartEvent{
		Provider: providerID,
		Model:    b.req.Model,
		Start:    start,
	})

	var resp *ChatResponse
	var err error

	// Execute with retry logic
retryLoop:
	for attempt := 0; ; attempt++ {
		resp, err = b.client.provider.Chat(ctx, &b.req)
		if err == nil {
			break
		}

		// Check if we should retry
		delay, shouldRetry := b.client.retry.NextDelay(attempt, err)
		if !shouldRetry {
			break
		}

		// Wait before retry, respecting context cancellation
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break retryLoop
		case <-time.After(delay):
			continue
		}
	}

	// Emit telemetry end
	end := time.Now()
	usage := TokenUsage{}
	if resp != nil {
		usage = resp.Usage
	}
	b.client.telemetry.OnRequestEnd(RequestEndEvent{
		Provider: providerID,
		Model:    b.req.Model,
		Start:    start,
		End:      end,
		Usage:    usage,
		Err:      err,
	})

	return resp, err
}

// Stream executes the chat request and returns a streaming response.
// It applies validation and telemetry.
func (b *ChatBuilder) Stream(ctx context.Context) (*ChatStream, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}

	start := time.Now()
	providerID := b.client.provider.ID()

	// Emit telemetry start
	b.client.telemetry.OnRequestStart(RequestStartEvent{
		Provider: providerID,
		Model:    b.req.Model,
		Start:    start,
	})

	stream, err := b.client.provider.StreamChat(ctx, &b.req)
	if err != nil {
		// Emit telemetry end on immediate error
		b.client.telemetry.OnRequestEnd(RequestEndEvent{
			Provider: providerID,
			Model:    b.req.Model,
			Start:    start,
			End:      time.Now(),
			Err:      err,
		})
		return nil, err
	}

	// Wrap the stream to emit telemetry when it completes
	return wrapStreamWithTelemetry(stream, b.client.telemetry, providerID, b.req.Model, start), nil
}

// wrapStreamWithTelemetry wraps a ChatStream to emit telemetry on completion.
func wrapStreamWithTelemetry(
	stream *ChatStream,
	hook TelemetryHook,
	provider string,
	model ModelID,
	start time.Time,
) *ChatStream {
	finalCh := make(chan *ChatResponse, 1)
	errCh := make(chan error, 1)

	go func() {
		defer close(finalCh)
		defer close(errCh)

		var finalResp *ChatResponse
		var finalErr error

		// Wait for either final response or error
		select {
		case resp, ok := <-stream.Final:
			if ok {
				finalResp = resp
				finalCh <- resp
			}
		case err, ok := <-stream.Err:
			if ok {
				finalErr = err
				errCh <- err
			}
		}

		// Emit telemetry end
		usage := TokenUsage{}
		if finalResp != nil {
			usage = finalResp.Usage
		}
		hook.OnRequestEnd(RequestEndEvent{
			Provider: provider,
			Model:    model,
			Start:    start,
			End:      time.Now(),
			Usage:    usage,
			Err:      finalErr,
		})
	}()

	return &ChatStream{
		Ch:    stream.Ch,
		Err:   errCh,
		Final: finalCh,
	}
}
