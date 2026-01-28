package openai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erikhoward/iris/core"
)

func TestResponsesAPIChatSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/responses" {
			t.Errorf("Path = %q, want /responses", r.URL.Path)
		}

		w.Header().Set("x-request-id", "req-resp-123")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(responsesResponse{
			ID:         "resp-123",
			Model:      "gpt-5.2",
			Status:     "completed",
			OutputText: "Hello! How can I help you?",
			Output: []responsesOutput{
				{
					Type: "message",
					Role: "assistant",
				},
			},
			Usage: &responsesUsage{
				InputTokens:  10,
				OutputTokens: 8,
				TotalTokens:  18,
			},
		})
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	resp, err := p.Chat(context.Background(), &core.ChatRequest{
		Model: ModelGPT52, // Uses Responses API
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "Hello"},
		},
	})

	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if resp.ID != "resp-123" {
		t.Errorf("ID = %q, want %q", resp.ID, "resp-123")
	}

	if resp.Model != "gpt-5.2" {
		t.Errorf("Model = %q, want %q", resp.Model, "gpt-5.2")
	}

	if resp.Output != "Hello! How can I help you?" {
		t.Errorf("Output = %q, want %q", resp.Output, "Hello! How can I help you?")
	}

	if resp.Status != "completed" {
		t.Errorf("Status = %q, want %q", resp.Status, "completed")
	}

	if resp.Usage.PromptTokens != 10 {
		t.Errorf("Usage.PromptTokens = %d, want 10", resp.Usage.PromptTokens)
	}
}

func TestResponsesAPIChatWithReasoning(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify reasoning is in the request using raw JSON
		var reqBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		reasoning, ok := reqBody["reasoning"].(map[string]any)
		if !ok {
			t.Error("Expected reasoning parameter in request")
		} else if reasoning["effort"] != "high" {
			t.Errorf("Reasoning.Effort = %v, want %q", reasoning["effort"], "high")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(responsesResponse{
			ID:         "resp-reason-123",
			Model:      "gpt-5.2",
			Status:     "completed",
			OutputText: "The answer is 42.",
			Output: []responsesOutput{
				{
					Type: "reasoning",
					ID:   "rs_123",
					Summary: []responsesReasoningSummary{
						{Type: "text", Text: "Calculated the answer"},
					},
				},
				{
					Type: "message",
					Role: "assistant",
				},
			},
			Usage: &responsesUsage{
				InputTokens:     10,
				OutputTokens:    20,
				TotalTokens:     30,
				ReasoningTokens: 15,
			},
		})
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	resp, err := p.Chat(context.Background(), &core.ChatRequest{
		Model:           ModelGPT52,
		ReasoningEffort: core.ReasoningEffortHigh,
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "What is the meaning of life?"},
		},
	})

	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if resp.Reasoning == nil {
		t.Fatal("Expected reasoning output")
	}

	if len(resp.Reasoning.Summary) != 1 {
		t.Fatalf("len(Reasoning.Summary) = %d, want 1", len(resp.Reasoning.Summary))
	}

	if resp.Reasoning.Summary[0] != "Calculated the answer" {
		t.Errorf("Reasoning.Summary[0] = %q, want %q", resp.Reasoning.Summary[0], "Calculated the answer")
	}
}

func TestResponsesAPIChatWithToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(responsesResponse{
			ID:     "resp-tool-123",
			Model:  "gpt-5.2",
			Status: "completed",
			Output: []responsesOutput{
				{
					Type:      "function_call",
					CallID:    "call_abc123",
					Name:      "get_weather",
					Arguments: `{"location":"San Francisco","unit":"celsius"}`,
				},
			},
			Usage: &responsesUsage{
				InputTokens:  15,
				OutputTokens: 20,
				TotalTokens:  35,
			},
		})
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	resp, err := p.Chat(context.Background(), &core.ChatRequest{
		Model: ModelGPT52,
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "What's the weather?"},
		},
	})

	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(resp.ToolCalls))
	}

	tc := resp.ToolCalls[0]
	if tc.ID != "call_abc123" {
		t.Errorf("ToolCalls[0].ID = %q, want %q", tc.ID, "call_abc123")
	}

	if tc.Name != "get_weather" {
		t.Errorf("ToolCalls[0].Name = %q, want %q", tc.Name, "get_weather")
	}
}

func TestResponsesAPIChatWithBuiltInTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify built-in tools are in the request using raw JSON
		var reqBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		toolsRaw, ok := reqBody["tools"].([]any)
		if !ok {
			t.Fatal("Expected tools in request")
		}

		foundWebSearch := false
		for _, toolRaw := range toolsRaw {
			tool, ok := toolRaw.(map[string]any)
			if ok && tool["type"] == "web_search" {
				foundWebSearch = true
				break
			}
		}

		if !foundWebSearch {
			t.Error("Expected web_search tool in request")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(responsesResponse{
			ID:         "resp-web-123",
			Model:      "gpt-5.2",
			Status:     "completed",
			OutputText: "Based on my web search...",
			Usage: &responsesUsage{
				InputTokens:  10,
				OutputTokens: 50,
				TotalTokens:  60,
			},
		})
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	resp, err := p.Chat(context.Background(), &core.ChatRequest{
		Model: ModelGPT52,
		BuiltInTools: []core.BuiltInTool{
			{Type: "web_search"},
		},
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "What's the latest news?"},
		},
	})

	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if resp.Output != "Based on my web search..." {
		t.Errorf("Output = %q, want %q", resp.Output, "Based on my web search...")
	}
}

func TestResponsesAPIChatWithInstructions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		instructions, _ := reqBody["instructions"].(string)
		if instructions != "You are a helpful assistant." {
			t.Errorf("Instructions = %q, want %q", instructions, "You are a helpful assistant.")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(responsesResponse{
			ID:         "resp-inst-123",
			Model:      "gpt-5.2",
			Status:     "completed",
			OutputText: "Hello!",
			Usage: &responsesUsage{
				InputTokens:  10,
				OutputTokens: 5,
				TotalTokens:  15,
			},
		})
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	resp, err := p.Chat(context.Background(), &core.ChatRequest{
		Model:        ModelGPT52,
		Instructions: "You are a helpful assistant.",
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "Hi"},
		},
	})

	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if resp.Output != "Hello!" {
		t.Errorf("Output = %q, want %q", resp.Output, "Hello!")
	}
}

func TestResponsesAPIChatWithPreviousResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		prevID, _ := reqBody["previous_response_id"].(string)
		if prevID != "resp-prev-123" {
			t.Errorf("PreviousResponseID = %q, want %q", prevID, "resp-prev-123")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(responsesResponse{
			ID:         "resp-chain-123",
			Model:      "gpt-5.2",
			Status:     "completed",
			OutputText: "Continuing from before...",
			Usage: &responsesUsage{
				InputTokens:  10,
				OutputTokens: 10,
				TotalTokens:  20,
			},
		})
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	resp, err := p.Chat(context.Background(), &core.ChatRequest{
		Model:              ModelGPT52,
		PreviousResponseID: "resp-prev-123",
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "Continue"},
		},
	})

	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if resp.ID != "resp-chain-123" {
		t.Errorf("ID = %q, want %q", resp.ID, "resp-chain-123")
	}
}

func TestResponsesAPIChatError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-request-id", "req-err-resp")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"message":"Invalid request","type":"invalid_request_error"}}`))
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	_, err := p.Chat(context.Background(), &core.ChatRequest{
		Model: ModelGPT52,
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "Test"},
		},
	})

	if !errors.Is(err, core.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestOlderModelUsesCompletionsAPI(t *testing.T) {
	// Track which endpoint was called
	var calledPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledPath = r.URL.Path

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(openAIResponse{
			ID:    "chatcmpl-legacy",
			Model: "gpt-4o",
			Choices: []openAIChoice{
				{
					Message: openAIRespMsg{
						Role:    "assistant",
						Content: "Hello from Chat Completions!",
					},
				},
			},
			Usage: openAIUsage{
				PromptTokens:     5,
				CompletionTokens: 5,
				TotalTokens:      10,
			},
		})
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	_, err := p.Chat(context.Background(), &core.ChatRequest{
		Model: ModelGPT4o, // Uses Chat Completions API
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "Hello"},
		},
	})

	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if calledPath != "/chat/completions" {
		t.Errorf("Called path = %q, want /chat/completions", calledPath)
	}
}

func TestUnknownModelUsesCompletionsAPI(t *testing.T) {
	var calledPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledPath = r.URL.Path

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(openAIResponse{
			ID:    "chatcmpl-unknown",
			Model: "some-future-model",
			Choices: []openAIChoice{
				{
					Message: openAIRespMsg{
						Role:    "assistant",
						Content: "Hello!",
					},
				},
			},
		})
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	_, err := p.Chat(context.Background(), &core.ChatRequest{
		Model: "some-future-model", // Unknown model defaults to completions
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "Hello"},
		},
	})

	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if calledPath != "/chat/completions" {
		t.Errorf("Called path = %q, want /chat/completions", calledPath)
	}
}
