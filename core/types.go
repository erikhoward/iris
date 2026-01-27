// Package core provides the Iris SDK client and types.
package core

import "encoding/json"

// Feature represents a capability that a provider may support.
type Feature string

const (
	FeatureChat          Feature = "chat"
	FeatureChatStreaming Feature = "chat_streaming"
	FeatureToolCalling   Feature = "tool_calling"
)

// ModelID is a string identifier for a model.
// Using string avoids coupling to provider-specific enums.
type ModelID string

// Role represents a message participant role.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message represents a single message in a conversation.
// Content is plain text for v0.1.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// TokenUsage tracks token consumption for a request.
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ToolCall represents a tool invocation requested by the model.
// Arguments MUST be valid JSON bytes and MUST preserve raw JSON (no reformatting).
type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// Tool is a placeholder interface for tool definitions.
// Full implementation is in the tools package (Task 07).
type Tool interface {
	Name() string
	Description() string
}

// ChatRequest represents a request to a chat model.
type ChatRequest struct {
	Model       ModelID  `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature *float32 `json:"temperature,omitempty"`
	MaxTokens   *int     `json:"max_tokens,omitempty"`
	Tools       []Tool   `json:"-"` // Tools are handled separately by providers
}

// ChatResponse represents a response from a chat model.
// For providers returning multiple choices, v0.1 uses only the first choice.
type ChatResponse struct {
	ID        string     `json:"id"`
	Model     ModelID    `json:"model"`
	Output    string     `json:"output"`
	Usage     TokenUsage `json:"usage"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ChatChunk represents an incremental streaming response.
// Delta contains incremental assistant text.
type ChatChunk struct {
	Delta string `json:"delta"`
}
