# Google Gemini Provider Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement a Google Gemini provider for iris that supports chat, streaming, tool calling, and thinking/reasoning mode for all 5 Gemini models.

**Architecture:** Mirror the existing Anthropic provider structure. The Gemini provider implements `core.Provider` interface with `Chat()` and `StreamChat()` methods. Model-specific thinking configuration (thinkingLevel vs thinkingBudget) is handled internally based on model generation.

**Tech Stack:** Go 1.21+, net/http for API calls, SSE for streaming, httptest for unit tests.

---

## Task 1: Create types_gemini.go

**Files:**
- Create: `providers/gemini/types_gemini.go`

**Step 1: Write the type definitions**

```go
// Package gemini provides a Google Gemini API provider implementation for Iris.
package gemini

import "encoding/json"

// geminiRequest represents a request to the Gemini generateContent API.
type geminiRequest struct {
	Contents          []geminiContent  `json:"contents"`
	SystemInstruction *geminiContent   `json:"system_instruction,omitempty"`
	GenerationConfig  *geminiGenConfig `json:"generationConfig,omitempty"`
	Tools             []geminiTool     `json:"tools,omitempty"`
}

// geminiContent represents a content block (user or model turn).
type geminiContent struct {
	Role  string       `json:"role,omitempty"` // "user" or "model"
	Parts []geminiPart `json:"parts"`
}

// geminiPart represents a part within content (text, function call, etc).
type geminiPart struct {
	Text             string               `json:"text,omitempty"`
	FunctionCall     *geminiFunctionCall  `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResp  `json:"functionResponse,omitempty"`
	Thought          *bool                `json:"thought,omitempty"`
}

// geminiGenConfig holds generation configuration.
type geminiGenConfig struct {
	Temperature     *float32           `json:"temperature,omitempty"`
	MaxOutputTokens *int               `json:"maxOutputTokens,omitempty"`
	ThinkingConfig  *geminiThinkConfig `json:"thinkingConfig,omitempty"`
}

// geminiThinkConfig configures thinking/reasoning mode.
type geminiThinkConfig struct {
	ThinkingLevel   string `json:"thinkingLevel,omitempty"`   // Gemini 3: "minimal"/"low"/"medium"/"high"
	ThinkingBudget  *int   `json:"thinkingBudget,omitempty"`  // Gemini 2.5: token count
	IncludeThoughts bool   `json:"includeThoughts,omitempty"`
}

// geminiTool represents a tool definition.
type geminiTool struct {
	FunctionDeclarations []geminiFunctionDecl `json:"functionDeclarations"`
}

// geminiFunctionDecl declares a function the model can call.
type geminiFunctionDecl struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// geminiFunctionCall represents a function call from the model.
type geminiFunctionCall struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

// geminiFunctionResp provides a function response back to the model.
type geminiFunctionResp struct {
	Name     string          `json:"name"`
	Response json.RawMessage `json:"response"`
}

// geminiResponse represents a response from the Gemini API.
type geminiResponse struct {
	Candidates    []geminiCandidate `json:"candidates"`
	UsageMetadata *geminiUsage      `json:"usageMetadata,omitempty"`
}

// geminiCandidate represents a response candidate.
type geminiCandidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason,omitempty"`
}

// geminiUsage tracks token usage.
type geminiUsage struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	ThoughtsTokenCount   int `json:"thoughtsTokenCount,omitempty"`
}

// geminiErrorResponse represents an error response from the API.
type geminiErrorResponse struct {
	Error geminiError `json:"error"`
}

// geminiError contains error details.
type geminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}
```

**Step 2: Run gofmt to verify syntax**

Run: `gofmt -d providers/gemini/types_gemini.go`
Expected: No output (file is correctly formatted)

**Step 3: Commit**

```bash
git add providers/gemini/types_gemini.go
git commit -m "feat(gemini): add API type definitions"
```

---

## Task 2: Create options.go

**Files:**
- Create: `providers/gemini/options.go`
- Test: `providers/gemini/options_test.go`

**Step 1: Write the failing test**

```go
package gemini

import (
	"net/http"
	"testing"
	"time"
)

func TestWithBaseURL(t *testing.T) {
	cfg := &Config{}
	WithBaseURL("https://custom.api.com")(cfg)

	if cfg.BaseURL != "https://custom.api.com" {
		t.Errorf("BaseURL = %q, want 'https://custom.api.com'", cfg.BaseURL)
	}
}

func TestWithHTTPClient(t *testing.T) {
	customClient := &http.Client{Timeout: 30 * time.Second}
	cfg := &Config{}
	WithHTTPClient(customClient)(cfg)

	if cfg.HTTPClient != customClient {
		t.Error("HTTPClient should be custom client")
	}
}

func TestWithHeader(t *testing.T) {
	cfg := &Config{}
	WithHeader("X-Custom", "value")(cfg)

	if cfg.Headers.Get("X-Custom") != "value" {
		t.Errorf("X-Custom header = %q, want 'value'", cfg.Headers.Get("X-Custom"))
	}
}

func TestWithTimeout(t *testing.T) {
	cfg := &Config{}
	WithTimeout(60 * time.Second)(cfg)

	if cfg.Timeout != 60*time.Second {
		t.Errorf("Timeout = %v, want 60s", cfg.Timeout)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./providers/gemini/... -run TestWith -v`
Expected: FAIL (package doesn't exist yet)

**Step 3: Write minimal implementation**

```go
package gemini

import (
	"net/http"
	"time"
)

// Config holds configuration for the Gemini provider.
type Config struct {
	// APIKey is the Gemini API key (required).
	APIKey string

	// BaseURL is the API base URL. Defaults to https://generativelanguage.googleapis.com
	BaseURL string

	// HTTPClient is the HTTP client to use. Defaults to http.DefaultClient.
	HTTPClient *http.Client

	// Headers contains optional extra headers to include in requests.
	Headers http.Header

	// Timeout is the optional request timeout.
	Timeout time.Duration
}

// DefaultBaseURL is the default Gemini API base URL.
const DefaultBaseURL = "https://generativelanguage.googleapis.com"

// Option configures the Gemini provider.
type Option func(*Config)

// WithBaseURL sets the API base URL.
func WithBaseURL(url string) Option {
	return func(c *Config) {
		c.BaseURL = url
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Config) {
		c.HTTPClient = client
	}
}

// WithHeader adds an extra header to include in requests.
func WithHeader(key, value string) Option {
	return func(c *Config) {
		if c.Headers == nil {
			c.Headers = make(http.Header)
		}
		c.Headers.Set(key, value)
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Config) {
		c.Timeout = d
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./providers/gemini/... -run TestWith -v`
Expected: PASS

**Step 5: Commit**

```bash
git add providers/gemini/options.go providers/gemini/options_test.go
git commit -m "feat(gemini): add configuration options"
```

---

## Task 3: Create models.go

**Files:**
- Create: `providers/gemini/models.go`

**Step 1: Write the model definitions**

```go
// Package gemini provides a Google Gemini API provider implementation for Iris.
package gemini

import "github.com/erikhoward/iris/core"

// Model constants for Google Gemini models.
const (
	// Gemini 3 series (preview)
	ModelGemini3Pro   core.ModelID = "gemini-3-pro-preview"
	ModelGemini3Flash core.ModelID = "gemini-3-flash-preview"

	// Gemini 2.5 series
	ModelGemini25Flash     core.ModelID = "gemini-2.5-flash"
	ModelGemini25FlashLite core.ModelID = "gemini-2.5-flash-lite"
	ModelGemini25Pro       core.ModelID = "gemini-2.5-pro"
)

// models is the static list of supported models.
var models = []core.ModelInfo{
	{
		ID:          ModelGemini3Pro,
		DisplayName: "Gemini 3 Pro Preview",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
			core.FeatureReasoning,
		},
	},
	{
		ID:          ModelGemini3Flash,
		DisplayName: "Gemini 3 Flash Preview",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
			core.FeatureReasoning,
		},
	},
	{
		ID:          ModelGemini25Flash,
		DisplayName: "Gemini 2.5 Flash",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
			core.FeatureReasoning,
		},
	},
	{
		ID:          ModelGemini25FlashLite,
		DisplayName: "Gemini 2.5 Flash Lite",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
			core.FeatureReasoning,
		},
	},
	{
		ID:          ModelGemini25Pro,
		DisplayName: "Gemini 2.5 Pro",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
			core.FeatureReasoning,
		},
	},
}

// modelRegistry is a map for quick model lookup by ID.
var modelRegistry = buildModelRegistry()

// buildModelRegistry creates a map from model ID to ModelInfo.
func buildModelRegistry() map[core.ModelID]*core.ModelInfo {
	registry := make(map[core.ModelID]*core.ModelInfo, len(models))
	for i := range models {
		registry[models[i].ID] = &models[i]
	}
	return registry
}

// GetModelInfo returns the ModelInfo for a given model ID, or nil if not found.
func GetModelInfo(id core.ModelID) *core.ModelInfo {
	return modelRegistry[id]
}

// isGemini3Model returns true if the model is a Gemini 3 series model.
func isGemini3Model(model string) bool {
	return model == string(ModelGemini3Pro) || model == string(ModelGemini3Flash)
}
```

**Step 2: Run go build to verify**

Run: `go build ./providers/gemini/...`
Expected: No errors

**Step 3: Commit**

```bash
git add providers/gemini/models.go
git commit -m "feat(gemini): add model definitions"
```

---

## Task 4: Create errors.go

**Files:**
- Create: `providers/gemini/errors.go`
- Test: `providers/gemini/errors_test.go`

**Step 1: Write the failing test**

```go
package gemini

import (
	"errors"
	"net/http"
	"testing"

	"github.com/erikhoward/iris/core"
)

func TestNormalizeError(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantSentinel error
	}{
		{
			name:       "bad request",
			status:     400,
			body:       `{"error":{"code":400,"message":"Invalid argument","status":"INVALID_ARGUMENT"}}`,
			wantSentinel: core.ErrBadRequest,
		},
		{
			name:       "unauthorized",
			status:     401,
			body:       `{"error":{"code":401,"message":"Invalid API key","status":"UNAUTHENTICATED"}}`,
			wantSentinel: core.ErrUnauthorized,
		},
		{
			name:       "forbidden",
			status:     403,
			body:       `{"error":{"code":403,"message":"Permission denied","status":"PERMISSION_DENIED"}}`,
			wantSentinel: core.ErrUnauthorized,
		},
		{
			name:       "not found",
			status:     404,
			body:       `{"error":{"code":404,"message":"Not found","status":"NOT_FOUND"}}`,
			wantSentinel: core.ErrBadRequest,
		},
		{
			name:       "rate limited",
			status:     429,
			body:       `{"error":{"code":429,"message":"Resource exhausted","status":"RESOURCE_EXHAUSTED"}}`,
			wantSentinel: core.ErrRateLimited,
		},
		{
			name:       "server error",
			status:     500,
			body:       `{"error":{"code":500,"message":"Internal error","status":"INTERNAL"}}`,
			wantSentinel: core.ErrServer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := normalizeError(tt.status, []byte(tt.body))

			var provErr *core.ProviderError
			if !errors.As(err, &provErr) {
				t.Fatalf("error is not ProviderError: %v", err)
			}

			if !errors.Is(provErr.Err, tt.wantSentinel) {
				t.Errorf("sentinel = %v, want %v", provErr.Err, tt.wantSentinel)
			}

			if provErr.Provider != "gemini" {
				t.Errorf("Provider = %q, want 'gemini'", provErr.Provider)
			}
		})
	}
}

func TestNewNetworkError(t *testing.T) {
	origErr := errors.New("connection refused")
	err := newNetworkError(origErr)

	var provErr *core.ProviderError
	if !errors.As(err, &provErr) {
		t.Fatalf("error is not ProviderError: %v", err)
	}

	if !errors.Is(provErr.Err, core.ErrNetwork) {
		t.Errorf("sentinel = %v, want ErrNetwork", provErr.Err)
	}
}

func TestNewDecodeError(t *testing.T) {
	origErr := errors.New("invalid json")
	err := newDecodeError(origErr)

	var provErr *core.ProviderError
	if !errors.As(err, &provErr) {
		t.Fatalf("error is not ProviderError: %v", err)
	}

	if !errors.Is(provErr.Err, core.ErrDecode) {
		t.Errorf("sentinel = %v, want ErrDecode", provErr.Err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./providers/gemini/... -run TestNormalizeError -v`
Expected: FAIL (function not defined)

**Step 3: Write minimal implementation**

```go
package gemini

import (
	"encoding/json"
	"net/http"

	"github.com/erikhoward/iris/core"
)

// normalizeError converts an HTTP error response to a ProviderError with the appropriate sentinel.
func normalizeError(status int, body []byte) error {
	// Parse error response if possible
	var errResp geminiErrorResponse
	_ = json.Unmarshal(body, &errResp)

	message := errResp.Error.Message
	if message == "" {
		message = http.StatusText(status)
	}

	code := errResp.Error.Status
	if code == "" {
		code = "unknown_error"
	}

	// Determine sentinel error based on status
	var sentinel error
	switch {
	case status == http.StatusBadRequest || status == http.StatusNotFound:
		sentinel = core.ErrBadRequest
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		sentinel = core.ErrUnauthorized
	case status == http.StatusTooManyRequests:
		sentinel = core.ErrRateLimited
	case status >= 500:
		sentinel = core.ErrServer
	default:
		sentinel = core.ErrServer
	}

	return &core.ProviderError{
		Provider: "gemini",
		Status:   status,
		Code:     code,
		Message:  message,
		Err:      sentinel,
	}
}

// newNetworkError creates a ProviderError for network-related failures.
func newNetworkError(err error) error {
	return &core.ProviderError{
		Provider: "gemini",
		Message:  err.Error(),
		Err:      core.ErrNetwork,
	}
}

// newDecodeError creates a ProviderError for JSON decode failures.
func newDecodeError(err error) error {
	return &core.ProviderError{
		Provider: "gemini",
		Message:  err.Error(),
		Err:      core.ErrDecode,
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./providers/gemini/... -run "TestNormalizeError|TestNewNetworkError|TestNewDecodeError" -v`
Expected: PASS

**Step 5: Commit**

```bash
git add providers/gemini/errors.go providers/gemini/errors_test.go
git commit -m "feat(gemini): add error normalization"
```

---

## Task 5: Create mapping.go

**Files:**
- Create: `providers/gemini/mapping.go`
- Test: `providers/gemini/mapping_test.go`

**Step 1: Write the failing tests**

```go
package gemini

import (
	"encoding/json"
	"testing"

	"github.com/erikhoward/iris/core"
)

func TestMapMessages(t *testing.T) {
	msgs := []core.Message{
		{Role: core.RoleSystem, Content: "You are helpful."},
		{Role: core.RoleSystem, Content: "Be concise."},
		{Role: core.RoleUser, Content: "Hello"},
		{Role: core.RoleAssistant, Content: "Hi there"},
		{Role: core.RoleUser, Content: "How are you?"},
	}

	system, contents := mapMessages(msgs)

	// Check system concatenation
	if system != "You are helpful.\n\nBe concise." {
		t.Errorf("system = %q, want 'You are helpful.\\n\\nBe concise.'", system)
	}

	// Check message count (excluding system)
	if len(contents) != 3 {
		t.Fatalf("contents count = %d, want 3", len(contents))
	}

	// Check roles
	if contents[0].Role != "user" {
		t.Errorf("contents[0].Role = %q, want 'user'", contents[0].Role)
	}
	if contents[1].Role != "model" {
		t.Errorf("contents[1].Role = %q, want 'model'", contents[1].Role)
	}
	if contents[2].Role != "user" {
		t.Errorf("contents[2].Role = %q, want 'user'", contents[2].Role)
	}

	// Check content
	if contents[0].Parts[0].Text != "Hello" {
		t.Errorf("contents[0].Parts[0].Text = %q, want 'Hello'", contents[0].Parts[0].Text)
	}
}

func TestMapMessagesNoSystem(t *testing.T) {
	msgs := []core.Message{
		{Role: core.RoleUser, Content: "Hello"},
	}

	system, contents := mapMessages(msgs)

	if system != "" {
		t.Errorf("system = %q, want empty", system)
	}

	if len(contents) != 1 {
		t.Fatalf("contents count = %d, want 1", len(contents))
	}
}

func TestBuildThinkingConfigGemini3(t *testing.T) {
	tests := []struct {
		model  string
		effort core.ReasoningEffort
		wantLevel string
	}{
		{string(ModelGemini3Flash), core.ReasoningEffortNone, "minimal"},
		{string(ModelGemini3Flash), core.ReasoningEffortLow, "low"},
		{string(ModelGemini3Flash), core.ReasoningEffortMedium, "medium"},
		{string(ModelGemini3Flash), core.ReasoningEffortHigh, "high"},
		{string(ModelGemini3Pro), core.ReasoningEffortNone, "low"}, // Pro can't disable
		{string(ModelGemini3Pro), core.ReasoningEffortHigh, "high"},
	}

	for _, tt := range tests {
		t.Run(tt.model+"_"+string(tt.effort), func(t *testing.T) {
			cfg := buildThinkingConfig(tt.model, tt.effort)
			if cfg == nil {
				t.Fatal("cfg is nil")
			}
			if cfg.ThinkingLevel != tt.wantLevel {
				t.Errorf("ThinkingLevel = %q, want %q", cfg.ThinkingLevel, tt.wantLevel)
			}
			if cfg.ThinkingBudget != nil {
				t.Error("ThinkingBudget should be nil for Gemini 3")
			}
		})
	}
}

func TestBuildThinkingConfigGemini25(t *testing.T) {
	tests := []struct {
		effort     core.ReasoningEffort
		wantBudget int
	}{
		{core.ReasoningEffortNone, 0},
		{core.ReasoningEffortLow, 1024},
		{core.ReasoningEffortMedium, 8192},
		{core.ReasoningEffortHigh, 24576},
		{core.ReasoningEffortXHigh, 32768},
	}

	for _, tt := range tests {
		t.Run(string(tt.effort), func(t *testing.T) {
			cfg := buildThinkingConfig(string(ModelGemini25Flash), tt.effort)
			if cfg == nil {
				t.Fatal("cfg is nil")
			}
			if cfg.ThinkingLevel != "" {
				t.Errorf("ThinkingLevel = %q, want empty for Gemini 2.5", cfg.ThinkingLevel)
			}
			if cfg.ThinkingBudget == nil {
				t.Fatal("ThinkingBudget is nil")
			}
			if *cfg.ThinkingBudget != tt.wantBudget {
				t.Errorf("ThinkingBudget = %d, want %d", *cfg.ThinkingBudget, tt.wantBudget)
			}
		})
	}
}

func TestBuildThinkingConfigNoReasoning(t *testing.T) {
	cfg := buildThinkingConfig(string(ModelGemini25Flash), "")
	if cfg != nil {
		t.Error("cfg should be nil when no reasoning effort specified")
	}
}

func TestMapResponse(t *testing.T) {
	resp := &geminiResponse{
		Candidates: []geminiCandidate{
			{
				Content: geminiContent{
					Role: "model",
					Parts: []geminiPart{
						{Text: "Hello "},
						{Text: "world!"},
					},
				},
				FinishReason: "STOP",
			},
		},
		UsageMetadata: &geminiUsage{
			PromptTokenCount:     10,
			CandidatesTokenCount: 5,
		},
	}

	result, err := mapResponse(resp, "gemini-2.5-flash")
	if err != nil {
		t.Fatalf("mapResponse error = %v", err)
	}

	if result.Output != "Hello world!" {
		t.Errorf("Output = %q, want 'Hello world!'", result.Output)
	}

	if result.Model != "gemini-2.5-flash" {
		t.Errorf("Model = %q, want 'gemini-2.5-flash'", result.Model)
	}

	if result.Usage.PromptTokens != 10 {
		t.Errorf("PromptTokens = %d, want 10", result.Usage.PromptTokens)
	}

	if result.Usage.CompletionTokens != 5 {
		t.Errorf("CompletionTokens = %d, want 5", result.Usage.CompletionTokens)
	}
}

func TestMapResponseWithToolCalls(t *testing.T) {
	resp := &geminiResponse{
		Candidates: []geminiCandidate{
			{
				Content: geminiContent{
					Role: "model",
					Parts: []geminiPart{
						{
							FunctionCall: &geminiFunctionCall{
								Name: "get_weather",
								Args: json.RawMessage(`{"location":"NYC"}`),
							},
						},
					},
				},
			},
		},
		UsageMetadata: &geminiUsage{
			PromptTokenCount:     10,
			CandidatesTokenCount: 5,
		},
	}

	result, err := mapResponse(resp, "gemini-2.5-flash")
	if err != nil {
		t.Fatalf("mapResponse error = %v", err)
	}

	if len(result.ToolCalls) != 1 {
		t.Fatalf("ToolCalls count = %d, want 1", len(result.ToolCalls))
	}

	tc := result.ToolCalls[0]
	if tc.ID != "call_0" {
		t.Errorf("ToolCall ID = %q, want 'call_0'", tc.ID)
	}
	if tc.Name != "get_weather" {
		t.Errorf("ToolCall Name = %q, want 'get_weather'", tc.Name)
	}
	if string(tc.Arguments) != `{"location":"NYC"}` {
		t.Errorf("ToolCall Arguments = %s, want '{\"location\":\"NYC\"}'", tc.Arguments)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./providers/gemini/... -run "TestMapMessages|TestBuildThinkingConfig|TestMapResponse" -v`
Expected: FAIL (functions not defined)

**Step 3: Write minimal implementation**

```go
package gemini

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/erikhoward/iris/core"
	"github.com/erikhoward/iris/tools"
)

// schemaProvider is an interface for tools that provide a JSON schema.
type schemaProvider interface {
	Schema() tools.ToolSchema
}

// buildRequest creates a Gemini API request from an Iris ChatRequest.
func buildRequest(req *core.ChatRequest) *geminiRequest {
	system, contents := mapMessages(req.Messages)

	gemReq := &geminiRequest{
		Contents: contents,
	}

	// Set system instruction if present
	if system != "" {
		gemReq.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: system}},
		}
	}

	// Build generation config
	genConfig := &geminiGenConfig{}
	hasGenConfig := false

	if req.Temperature != nil {
		genConfig.Temperature = req.Temperature
		hasGenConfig = true
	}

	if req.MaxTokens != nil {
		genConfig.MaxOutputTokens = req.MaxTokens
		hasGenConfig = true
	}

	// Build thinking config if reasoning effort specified
	if req.ReasoningEffort != "" {
		genConfig.ThinkingConfig = buildThinkingConfig(string(req.Model), req.ReasoningEffort)
		if genConfig.ThinkingConfig != nil {
			hasGenConfig = true
		}
	}

	if hasGenConfig {
		gemReq.GenerationConfig = genConfig
	}

	// Map tools if present
	if len(req.Tools) > 0 {
		gemReq.Tools = mapTools(req.Tools)
	}

	return gemReq
}

// mapMessages converts Iris messages to Gemini format.
// It extracts system messages into a single string and converts
// user/assistant messages to the Gemini content format.
func mapMessages(msgs []core.Message) (system string, contents []geminiContent) {
	var systemParts []string

	for _, msg := range msgs {
		switch msg.Role {
		case core.RoleSystem:
			systemParts = append(systemParts, msg.Content)
		case core.RoleUser:
			contents = append(contents, geminiContent{
				Role:  "user",
				Parts: []geminiPart{{Text: msg.Content}},
			})
		case core.RoleAssistant:
			contents = append(contents, geminiContent{
				Role:  "model",
				Parts: []geminiPart{{Text: msg.Content}},
			})
		}
	}

	// Concatenate system messages with double newlines
	if len(systemParts) > 0 {
		system = strings.Join(systemParts, "\n\n")
	}

	return system, contents
}

// buildThinkingConfig creates thinking configuration based on model and effort.
func buildThinkingConfig(model string, effort core.ReasoningEffort) *geminiThinkConfig {
	if effort == "" {
		return nil
	}

	cfg := &geminiThinkConfig{
		IncludeThoughts: true,
	}

	if isGemini3Model(model) {
		// Gemini 3 uses thinkingLevel
		cfg.ThinkingLevel = mapThinkingLevel(model, effort)
	} else {
		// Gemini 2.5 uses thinkingBudget
		budget := mapThinkingBudget(effort)
		cfg.ThinkingBudget = &budget
	}

	return cfg
}

// mapThinkingLevel maps ReasoningEffort to Gemini 3 thinkingLevel.
func mapThinkingLevel(model string, effort core.ReasoningEffort) string {
	// Gemini 3 Pro cannot disable thinking
	if model == string(ModelGemini3Pro) && effort == core.ReasoningEffortNone {
		return "low"
	}

	switch effort {
	case core.ReasoningEffortNone:
		return "minimal"
	case core.ReasoningEffortLow:
		return "low"
	case core.ReasoningEffortMedium:
		return "medium"
	case core.ReasoningEffortHigh, core.ReasoningEffortXHigh:
		return "high"
	default:
		return "medium"
	}
}

// mapThinkingBudget maps ReasoningEffort to Gemini 2.5 thinkingBudget.
func mapThinkingBudget(effort core.ReasoningEffort) int {
	switch effort {
	case core.ReasoningEffortNone:
		return 0
	case core.ReasoningEffortLow:
		return 1024
	case core.ReasoningEffortMedium:
		return 8192
	case core.ReasoningEffortHigh:
		return 24576
	case core.ReasoningEffortXHigh:
		return 32768
	default:
		return 8192
	}
}

// mapTools converts Iris tools to Gemini tool format.
func mapTools(irisTools []core.Tool) []geminiTool {
	if len(irisTools) == 0 {
		return nil
	}

	decls := make([]geminiFunctionDecl, len(irisTools))
	for i, t := range irisTools {
		var params json.RawMessage

		// Check if the tool provides a schema
		if sp, ok := t.(schemaProvider); ok {
			params = sp.Schema().JSONSchema
		}

		// Default to empty object if no schema
		if params == nil {
			params = json.RawMessage(`{}`)
		}

		decls[i] = geminiFunctionDecl{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  params,
		}
	}

	return []geminiTool{{FunctionDeclarations: decls}}
}

// mapResponse converts a Gemini response to an Iris ChatResponse.
func mapResponse(resp *geminiResponse, model string) (*core.ChatResponse, error) {
	result := &core.ChatResponse{
		Model: core.ModelID(model),
	}

	// Map usage
	if resp.UsageMetadata != nil {
		result.Usage = core.TokenUsage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.PromptTokenCount + resp.UsageMetadata.CandidatesTokenCount,
		}
	}

	// Extract content from first candidate
	if len(resp.Candidates) == 0 {
		return result, nil
	}

	candidate := resp.Candidates[0]

	var textParts []string
	var toolCalls []core.ToolCall
	var thoughtParts []string
	toolCallIndex := 0

	for _, part := range candidate.Content.Parts {
		// Check if this is a thought part
		if part.Thought != nil && *part.Thought {
			if part.Text != "" {
				thoughtParts = append(thoughtParts, part.Text)
			}
			continue
		}

		if part.Text != "" {
			textParts = append(textParts, part.Text)
		}

		if part.FunctionCall != nil {
			toolCalls = append(toolCalls, core.ToolCall{
				ID:        fmt.Sprintf("call_%d", toolCallIndex),
				Name:      part.FunctionCall.Name,
				Arguments: part.FunctionCall.Args,
			})
			toolCallIndex++
		}
	}

	result.Output = strings.Join(textParts, "")
	result.ToolCalls = toolCalls

	// Add reasoning output if thoughts were present
	if len(thoughtParts) > 0 {
		result.Reasoning = &core.ReasoningOutput{
			Summary: thoughtParts,
		}
	}

	return result, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./providers/gemini/... -run "TestMapMessages|TestBuildThinkingConfig|TestMapResponse" -v`
Expected: PASS

**Step 5: Commit**

```bash
git add providers/gemini/mapping.go providers/gemini/mapping_test.go
git commit -m "feat(gemini): add request/response mapping"
```

---

## Task 6: Create provider.go

**Files:**
- Create: `providers/gemini/provider.go`
- Test: `providers/gemini/provider_test.go`

**Step 1: Write the failing tests**

```go
package gemini

import (
	"net/http"
	"testing"
	"time"

	"github.com/erikhoward/iris/core"
)

func TestNew(t *testing.T) {
	p := New("test-key")

	if p.config.APIKey != "test-key" {
		t.Errorf("APIKey = %q, want 'test-key'", p.config.APIKey)
	}

	if p.config.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL = %q, want %q", p.config.BaseURL, DefaultBaseURL)
	}

	if p.config.HTTPClient != http.DefaultClient {
		t.Error("HTTPClient should be http.DefaultClient")
	}
}

func TestNewWithOptions(t *testing.T) {
	customClient := &http.Client{Timeout: 30 * time.Second}

	p := New("test-key",
		WithBaseURL("https://custom.api.com"),
		WithHTTPClient(customClient),
		WithHeader("X-Custom", "value"),
		WithTimeout(60*time.Second),
	)

	if p.config.BaseURL != "https://custom.api.com" {
		t.Errorf("BaseURL = %q, want 'https://custom.api.com'", p.config.BaseURL)
	}

	if p.config.HTTPClient != customClient {
		t.Error("HTTPClient should be custom client")
	}

	if p.config.Headers.Get("X-Custom") != "value" {
		t.Errorf("X-Custom header = %q, want 'value'", p.config.Headers.Get("X-Custom"))
	}

	if p.config.Timeout != 60*time.Second {
		t.Errorf("Timeout = %v, want 60s", p.config.Timeout)
	}
}

func TestID(t *testing.T) {
	p := New("test-key")

	if p.ID() != "gemini" {
		t.Errorf("ID() = %q, want 'gemini'", p.ID())
	}
}

func TestModels(t *testing.T) {
	p := New("test-key")
	models := p.Models()

	if len(models) != 5 {
		t.Errorf("Models() count = %d, want 5", len(models))
	}

	// Verify model IDs
	modelIDs := make(map[core.ModelID]bool)
	for _, m := range models {
		modelIDs[m.ID] = true
	}

	expected := []core.ModelID{
		ModelGemini3Pro,
		ModelGemini3Flash,
		ModelGemini25Flash,
		ModelGemini25FlashLite,
		ModelGemini25Pro,
	}

	for _, id := range expected {
		if !modelIDs[id] {
			t.Errorf("Missing model: %s", id)
		}
	}
}

func TestModelsReturnsCopy(t *testing.T) {
	p := New("test-key")

	models1 := p.Models()
	models2 := p.Models()

	// Modify first slice
	models1[0].DisplayName = "Modified"

	// Second slice should be unchanged
	if models2[0].DisplayName == "Modified" {
		t.Error("Models() should return a copy")
	}
}

func TestSupports(t *testing.T) {
	p := New("test-key")

	tests := []struct {
		feature core.Feature
		want    bool
	}{
		{core.FeatureChat, true},
		{core.FeatureChatStreaming, true},
		{core.FeatureToolCalling, true},
		{core.FeatureReasoning, true},
		{core.FeatureBuiltInTools, false},
		{core.FeatureResponseChain, false},
		{core.Feature("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.feature), func(t *testing.T) {
			if got := p.Supports(tt.feature); got != tt.want {
				t.Errorf("Supports(%q) = %v, want %v", tt.feature, got, tt.want)
			}
		})
	}
}

func TestBuildHeaders(t *testing.T) {
	p := New("test-key", WithHeader("X-Custom", "value"))
	headers := p.buildHeaders()

	if headers.Get("x-goog-api-key") != "test-key" {
		t.Errorf("x-goog-api-key = %q, want 'test-key'", headers.Get("x-goog-api-key"))
	}

	if headers.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want 'application/json'", headers.Get("Content-Type"))
	}

	if headers.Get("X-Custom") != "value" {
		t.Errorf("X-Custom = %q, want 'value'", headers.Get("X-Custom"))
	}
}

func TestGetModelInfo(t *testing.T) {
	tests := []struct {
		id      core.ModelID
		wantNil bool
		wantID  core.ModelID
	}{
		{ModelGemini3Pro, false, ModelGemini3Pro},
		{ModelGemini25Flash, false, ModelGemini25Flash},
		{"unknown-model", true, ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.id), func(t *testing.T) {
			info := GetModelInfo(tt.id)

			if tt.wantNil {
				if info != nil {
					t.Errorf("GetModelInfo(%q) should be nil", tt.id)
				}
				return
			}

			if info == nil {
				t.Fatalf("GetModelInfo(%q) = nil, want non-nil", tt.id)
			}

			if info.ID != tt.wantID {
				t.Errorf("ModelInfo.ID = %q, want %q", info.ID, tt.wantID)
			}
		})
	}
}

func TestProviderImplementsInterface(t *testing.T) {
	var _ core.Provider = (*Gemini)(nil)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./providers/gemini/... -run "TestNew|TestID|TestModels|TestSupports|TestBuildHeaders|TestGetModelInfo|TestProviderImplementsInterface" -v`
Expected: FAIL (Gemini struct not defined)

**Step 3: Write minimal implementation**

```go
package gemini

import (
	"context"
	"net/http"

	"github.com/erikhoward/iris/core"
)

// Gemini is an LLM provider implementation for the Google Gemini API.
// Gemini is safe for concurrent use.
type Gemini struct {
	config Config
}

// New creates a new Gemini provider with the given API key and options.
func New(apiKey string, opts ...Option) *Gemini {
	cfg := Config{
		APIKey:     apiKey,
		BaseURL:    DefaultBaseURL,
		HTTPClient: http.DefaultClient,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return &Gemini{config: cfg}
}

// ID returns the provider identifier.
func (p *Gemini) ID() string {
	return "gemini"
}

// Models returns the list of available models.
func (p *Gemini) Models() []core.ModelInfo {
	// Return a copy to prevent mutation
	result := make([]core.ModelInfo, len(models))
	copy(result, models)
	return result
}

// Supports reports whether the provider supports the given feature.
func (p *Gemini) Supports(feature core.Feature) bool {
	switch feature {
	case core.FeatureChat, core.FeatureChatStreaming, core.FeatureToolCalling, core.FeatureReasoning:
		return true
	default:
		return false
	}
}

// buildHeaders constructs the HTTP headers for an API request.
func (p *Gemini) buildHeaders() http.Header {
	headers := make(http.Header)

	// Required headers for Gemini API
	headers.Set("x-goog-api-key", p.config.APIKey)
	headers.Set("Content-Type", "application/json")

	// Copy any extra headers
	for key, values := range p.config.Headers {
		for _, v := range values {
			headers.Add(key, v)
		}
	}

	return headers
}

// Chat sends a non-streaming chat request.
func (p *Gemini) Chat(ctx context.Context, req *core.ChatRequest) (*core.ChatResponse, error) {
	return p.doChat(ctx, req)
}

// StreamChat sends a streaming chat request.
func (p *Gemini) StreamChat(ctx context.Context, req *core.ChatRequest) (*core.ChatStream, error) {
	return p.doStreamChat(ctx, req)
}

// Compile-time check that Gemini implements Provider.
var _ core.Provider = (*Gemini)(nil)
```

**Step 4: Run test to verify it passes**

Run: `go test ./providers/gemini/... -run "TestNew|TestID|TestModels|TestSupports|TestBuildHeaders|TestGetModelInfo|TestProviderImplementsInterface" -v`
Expected: FAIL (doChat and doStreamChat not defined yet - that's expected)

Note: Tests will fail until client.go and streaming.go are created. Continue to next task.

**Step 5: Commit (partial - provider structure only)**

```bash
git add providers/gemini/provider.go providers/gemini/provider_test.go
git commit -m "feat(gemini): add provider structure"
```

---

## Task 7: Create client.go

**Files:**
- Create: `providers/gemini/client.go`
- Test: `providers/gemini/client_test.go`

**Step 1: Write the failing tests**

```go
package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erikhoward/iris/core"
)

func TestDoChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Header.Get("x-goog-api-key") != "test-key" {
			t.Errorf("x-goog-api-key = %q, want 'test-key'", r.Header.Get("x-goog-api-key"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want 'application/json'", r.Header.Get("Content-Type"))
		}

		// Verify URL contains model
		if r.URL.Path != "/v1beta/models/gemini-2.5-flash:generateContent" {
			t.Errorf("Path = %q, want '/v1beta/models/gemini-2.5-flash:generateContent'", r.URL.Path)
		}

		// Verify request body
		body, _ := io.ReadAll(r.Body)
		var req geminiRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("unmarshaling request: %v", err)
		}

		if len(req.Contents) != 1 {
			t.Errorf("Contents count = %d, want 1", len(req.Contents))
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		resp := geminiResponse{
			Candidates: []geminiCandidate{
				{
					Content: geminiContent{
						Role:  "model",
						Parts: []geminiPart{{Text: "Hello!"}},
					},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: &geminiUsage{
				PromptTokenCount:     10,
				CandidatesTokenCount: 5,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))

	req := &core.ChatRequest{
		Model: "gemini-2.5-flash",
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "Hello"},
		},
	}

	resp, err := p.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if resp.Output != "Hello!" {
		t.Errorf("Output = %q, want 'Hello!'", resp.Output)
	}

	if resp.Usage.PromptTokens != 10 {
		t.Errorf("PromptTokens = %d, want 10", resp.Usage.PromptTokens)
	}

	if resp.Usage.CompletionTokens != 5 {
		t.Errorf("CompletionTokens = %d, want 5", resp.Usage.CompletionTokens)
	}
}

func TestDoChatWithSystemMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req geminiRequest
		json.Unmarshal(body, &req)

		// Verify system instruction
		if req.SystemInstruction == nil {
			t.Error("SystemInstruction is nil")
		} else if req.SystemInstruction.Parts[0].Text != "Be helpful." {
			t.Errorf("SystemInstruction = %q, want 'Be helpful.'", req.SystemInstruction.Parts[0].Text)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := geminiResponse{
			Candidates: []geminiCandidate{
				{Content: geminiContent{Parts: []geminiPart{{Text: "Hi!"}}}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))

	req := &core.ChatRequest{
		Model: "gemini-2.5-flash",
		Messages: []core.Message{
			{Role: core.RoleSystem, Content: "Be helpful."},
			{Role: core.RoleUser, Content: "Hello"},
		},
	}

	_, err := p.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
}

func TestDoChatWithToolCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := geminiResponse{
			Candidates: []geminiCandidate{
				{
					Content: geminiContent{
						Role: "model",
						Parts: []geminiPart{
							{
								FunctionCall: &geminiFunctionCall{
									Name: "get_weather",
									Args: json.RawMessage(`{"location":"NYC"}`),
								},
							},
						},
					},
				},
			},
			UsageMetadata: &geminiUsage{
				PromptTokenCount:     20,
				CandidatesTokenCount: 10,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))

	req := &core.ChatRequest{
		Model: "gemini-2.5-flash",
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "What's the weather?"},
		},
	}

	resp, err := p.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls count = %d, want 1", len(resp.ToolCalls))
	}

	tc := resp.ToolCalls[0]
	if tc.Name != "get_weather" {
		t.Errorf("ToolCall Name = %q, want 'get_weather'", tc.Name)
	}
	if string(tc.Arguments) != `{"location":"NYC"}` {
		t.Errorf("ToolCall Arguments = %s, want '{\"location\":\"NYC\"}'", tc.Arguments)
	}
}

func TestDoChatError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"code":401,"message":"Invalid API key","status":"UNAUTHENTICATED"}}`))
	}))
	defer server.Close()

	p := New("bad-key", WithBaseURL(server.URL))

	req := &core.ChatRequest{
		Model: "gemini-2.5-flash",
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "Hello"},
		},
	}

	_, err := p.Chat(context.Background(), req)
	if err == nil {
		t.Fatal("Chat() should return error")
	}

	var provErr *core.ProviderError
	if errors.As(err, &provErr) {
		if provErr.Message != "Invalid API key" {
			t.Errorf("error message = %q, want 'Invalid API key'", provErr.Message)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./providers/gemini/... -run "TestDoChat" -v`
Expected: FAIL (doChat not defined)

**Step 3: Write minimal implementation**

```go
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/erikhoward/iris/core"
)

// doChat performs a non-streaming chat request.
func (p *Gemini) doChat(ctx context.Context, req *core.ChatRequest) (*core.ChatResponse, error) {
	// Build Gemini request
	gemReq := buildRequest(req)

	// Marshal request body
	body, err := json.Marshal(gemReq)
	if err != nil {
		return nil, newDecodeError(err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", p.config.BaseURL, req.Model)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, newNetworkError(err)
	}

	// Set headers
	for key, values := range p.buildHeaders() {
		for _, v := range values {
			httpReq.Header.Add(key, v)
		}
	}

	// Execute request
	resp, err := p.config.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, newNetworkError(err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, newNetworkError(err)
	}

	// Check for error status
	if resp.StatusCode >= 400 {
		return nil, normalizeError(resp.StatusCode, respBody)
	}

	// Parse response
	var gemResp geminiResponse
	if err := json.Unmarshal(respBody, &gemResp); err != nil {
		return nil, newDecodeError(err)
	}

	// Map to Iris response
	return mapResponse(&gemResp, string(req.Model))
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./providers/gemini/... -run "TestDoChat" -v`
Expected: PASS (except tests requiring doStreamChat)

**Step 5: Commit**

```bash
git add providers/gemini/client.go providers/gemini/client_test.go
git commit -m "feat(gemini): add non-streaming chat"
```

---

## Task 8: Create streaming.go

**Files:**
- Create: `providers/gemini/streaming.go`
- Test: `providers/gemini/streaming_test.go`

**Step 1: Write the failing tests**

```go
package gemini

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/erikhoward/iris/core"
)

func TestDoStreamChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify URL contains streaming endpoint
		if !strings.Contains(r.URL.Path, ":streamGenerateContent") {
			t.Errorf("Path should contain ':streamGenerateContent', got %q", r.URL.Path)
		}
		if r.URL.Query().Get("alt") != "sse" {
			t.Errorf("alt query param = %q, want 'sse'", r.URL.Query().Get("alt"))
		}

		// Write SSE response
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		events := []string{
			`data: {"candidates":[{"content":{"parts":[{"text":"Hello"}]}}]}`,
			``,
			`data: {"candidates":[{"content":{"parts":[{"text":" world!"}]}}]}`,
			``,
			`data: {"candidates":[{"content":{"parts":[{"text":""}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5}}`,
			``,
		}

		for _, line := range events {
			w.Write([]byte(line + "\n"))
		}
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))

	req := &core.ChatRequest{
		Model: "gemini-2.5-flash",
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "Hello"},
		},
	}

	stream, err := p.StreamChat(context.Background(), req)
	if err != nil {
		t.Fatalf("StreamChat() error = %v", err)
	}

	// Collect chunks
	var chunks []string
	for chunk := range stream.Ch {
		chunks = append(chunks, chunk.Delta)
	}

	// Check for errors
	var streamErr error
	select {
	case e := <-stream.Err:
		streamErr = e
	default:
	}
	if streamErr != nil {
		t.Errorf("stream error = %v", streamErr)
	}

	// Get final response
	var finalResp *core.ChatResponse
	select {
	case r := <-stream.Final:
		finalResp = r
	default:
	}

	// Verify chunks
	if len(chunks) != 2 {
		t.Errorf("chunks count = %d, want 2", len(chunks))
	}

	accumulated := strings.Join(chunks, "")
	if accumulated != "Hello world!" {
		t.Errorf("accumulated = %q, want 'Hello world!'", accumulated)
	}

	// Verify final response
	if finalResp == nil {
		t.Fatal("finalResp is nil")
	}

	if finalResp.Usage.PromptTokens != 10 {
		t.Errorf("PromptTokens = %d, want 10", finalResp.Usage.PromptTokens)
	}

	if finalResp.Usage.CompletionTokens != 5 {
		t.Errorf("CompletionTokens = %d, want 5", finalResp.Usage.CompletionTokens)
	}
}

func TestDoStreamChatWithToolCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		events := []string{
			`data: {"candidates":[{"content":{"parts":[{"functionCall":{"name":"get_weather","args":{"location":"NYC"}}}]}}]}`,
			``,
			`data: {"candidates":[{"content":{"parts":[]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":20,"candidatesTokenCount":10}}`,
			``,
		}

		for _, line := range events {
			w.Write([]byte(line + "\n"))
		}
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))

	req := &core.ChatRequest{
		Model: "gemini-2.5-flash",
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "What's the weather?"},
		},
	}

	stream, err := p.StreamChat(context.Background(), req)
	if err != nil {
		t.Fatalf("StreamChat() error = %v", err)
	}

	// Drain chunks (should be none for tool-only response)
	var chunks []string
	for chunk := range stream.Ch {
		chunks = append(chunks, chunk.Delta)
	}

	if len(chunks) != 0 {
		t.Errorf("chunks count = %d, want 0 (tool use only)", len(chunks))
	}

	// Get final response
	var finalResp *core.ChatResponse
	select {
	case r := <-stream.Final:
		finalResp = r
	default:
	}

	if finalResp == nil {
		t.Fatal("finalResp is nil")
	}

	if len(finalResp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls count = %d, want 1", len(finalResp.ToolCalls))
	}

	tc := finalResp.ToolCalls[0]
	if tc.Name != "get_weather" {
		t.Errorf("ToolCall Name = %q, want 'get_weather'", tc.Name)
	}
}

func TestDoStreamChatError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"code":401,"message":"Invalid API key","status":"UNAUTHENTICATED"}}`))
	}))
	defer server.Close()

	p := New("bad-key", WithBaseURL(server.URL))

	req := &core.ChatRequest{
		Model: "gemini-2.5-flash",
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "Hello"},
		},
	}

	_, err := p.StreamChat(context.Background(), req)
	if err == nil {
		t.Fatal("StreamChat() should return error")
	}

	var provErr *core.ProviderError
	if errors.As(err, &provErr) {
		if provErr.Message != "Invalid API key" {
			t.Errorf("error message = %q, want 'Invalid API key'", provErr.Message)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./providers/gemini/... -run "TestDoStreamChat" -v`
Expected: FAIL (doStreamChat not defined)

**Step 3: Write minimal implementation**

```go
package gemini

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/erikhoward/iris/core"
)

// doStreamChat performs a streaming chat request.
func (p *Gemini) doStreamChat(ctx context.Context, req *core.ChatRequest) (*core.ChatStream, error) {
	// Build Gemini request
	gemReq := buildRequest(req)

	// Marshal request body
	body, err := json.Marshal(gemReq)
	if err != nil {
		return nil, newDecodeError(err)
	}

	// Create HTTP request with streaming endpoint
	url := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse", p.config.BaseURL, req.Model)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, newNetworkError(err)
	}

	// Set headers
	for key, values := range p.buildHeaders() {
		for _, v := range values {
			httpReq.Header.Add(key, v)
		}
	}

	// Execute request
	resp, err := p.config.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, newNetworkError(err)
	}

	// Check for error status
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		return nil, normalizeError(resp.StatusCode, respBody)
	}

	// Create channels
	chunkCh := make(chan core.ChatChunk, 100)
	errCh := make(chan error, 1)
	finalCh := make(chan *core.ChatResponse, 1)

	// Start goroutine to process SSE stream
	go p.processSSEStream(ctx, resp.Body, chunkCh, errCh, finalCh, string(req.Model))

	return &core.ChatStream{
		Ch:    chunkCh,
		Err:   errCh,
		Final: finalCh,
	}, nil
}

// processSSEStream reads the SSE stream and emits chunks.
func (p *Gemini) processSSEStream(
	ctx context.Context,
	body io.ReadCloser,
	chunkCh chan<- core.ChatChunk,
	errCh chan<- error,
	finalCh chan<- *core.ChatResponse,
	model string,
) {
	defer body.Close()
	defer close(chunkCh)
	defer close(errCh)
	defer close(finalCh)

	reader := bufio.NewReader(body)

	var accumulatedText strings.Builder
	var toolCalls []core.ToolCall
	var thoughtParts []string
	var usage *geminiUsage
	toolCallIndex := 0

	for {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			errCh <- ctx.Err()
			return
		default:
		}

		// Read line
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			errCh <- newNetworkError(err)
			return
		}

		// Trim whitespace
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Process data lines
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

		// Skip empty data
		if payload == "" {
			continue
		}

		// Parse event
		var event geminiResponse
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			errCh <- newDecodeError(err)
			return
		}

		// Update usage if present
		if event.UsageMetadata != nil {
			usage = event.UsageMetadata
		}

		// Process candidates
		if len(event.Candidates) == 0 {
			continue
		}

		candidate := event.Candidates[0]

		for _, part := range candidate.Content.Parts {
			// Check if this is a thought part
			if part.Thought != nil && *part.Thought {
				if part.Text != "" {
					thoughtParts = append(thoughtParts, part.Text)
				}
				continue
			}

			// Emit text delta
			if part.Text != "" {
				accumulatedText.WriteString(part.Text)
				select {
				case chunkCh <- core.ChatChunk{Delta: part.Text}:
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				}
			}

			// Accumulate tool calls
			if part.FunctionCall != nil {
				toolCalls = append(toolCalls, core.ToolCall{
					ID:        fmt.Sprintf("call_%d", toolCallIndex),
					Name:      part.FunctionCall.Name,
					Arguments: part.FunctionCall.Args,
				})
				toolCallIndex++
			}
		}
	}

	// Build final response
	finalResp := &core.ChatResponse{
		Model:     core.ModelID(model),
		Output:    accumulatedText.String(),
		ToolCalls: toolCalls,
	}

	if usage != nil {
		finalResp.Usage = core.TokenUsage{
			PromptTokens:     usage.PromptTokenCount,
			CompletionTokens: usage.CandidatesTokenCount,
			TotalTokens:      usage.PromptTokenCount + usage.CandidatesTokenCount,
		}
	}

	// Add reasoning output if thoughts were present
	if len(thoughtParts) > 0 {
		finalResp.Reasoning = &core.ReasoningOutput{
			Summary: thoughtParts,
		}
	}

	finalCh <- finalResp
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./providers/gemini/... -run "TestDoStreamChat" -v`
Expected: PASS

**Step 5: Commit**

```bash
git add providers/gemini/streaming.go providers/gemini/streaming_test.go
git commit -m "feat(gemini): add streaming chat support"
```

---

## Task 9: Update CLI - chat.go

**Files:**
- Modify: `cli/commands/chat.go:1-15` (imports)
- Modify: `cli/commands/chat.go:191-214` (createProvider function)

**Step 1: Add import**

Add to imports:
```go
"github.com/erikhoward/iris/providers/gemini"
```

**Step 2: Add gemini case to createProvider**

Add after the anthropic case (around line 210):
```go
case "gemini":
	// Check for custom base URL in config
	var opts []gemini.Option
	if cfg := GetConfig(); cfg != nil {
		if pc := cfg.GetProvider(providerID); pc != nil && pc.BaseURL != "" {
			opts = append(opts, gemini.WithBaseURL(pc.BaseURL))
		}
	}
	return gemini.New(apiKey, opts...), nil
```

**Step 3: Run tests**

Run: `go test ./cli/... -v`
Expected: PASS

**Step 4: Commit**

```bash
git add cli/commands/chat.go
git commit -m "feat(cli): add gemini provider support to chat command"
```

---

## Task 10: Update CLI - init.go

**Files:**
- Modify: `cli/commands/init.go:156-165` (defaultModel function)

**Step 1: Add gemini case to defaultModel**

```go
func defaultModel(provider string) string {
	switch provider {
	case "openai":
		return "gpt-4o"
	case "anthropic":
		return "claude-sonnet-4-5"
	case "gemini":
		return "gemini-2.5-flash"
	default:
		return "default"
	}
}
```

**Step 2: Run tests**

Run: `go test ./cli/... -run TestDefaultModel -v`
Expected: PASS (or update test if needed)

**Step 3: Update test if needed**

Check `cli/commands/init_test.go` for `TestDefaultModel` and add gemini case if the test exists.

**Step 4: Commit**

```bash
git add cli/commands/init.go cli/commands/init_test.go
git commit -m "feat(cli): add gemini default model to init command"
```

---

## Task 11: Run full test suite and verify

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All tests PASS

**Step 2: Run go vet**

Run: `go vet ./...`
Expected: No issues

**Step 3: Run gofmt**

Run: `gofmt -d .`
Expected: No output (all files formatted)

**Step 4: Fix any issues found**

If any tests fail or linting issues are found, fix them before proceeding.

---

## Task 12: Update README.md

**Files:**
- Modify: `README.md`

**Step 1: Add Gemini to supported providers table**

Add row:
```markdown
| Google Gemini | `gemini` | `gemini-2.5-flash`, `gemini-2.5-flash-lite`, `gemini-2.5-pro`, `gemini-3-flash-preview`, `gemini-3-pro-preview` |
```

**Step 2: Add SDK example**

```go
// Google Gemini
geminiProvider := gemini.New(os.Getenv("GEMINI_API_KEY"))
client := core.NewClient(geminiProvider)

resp, err := client.Chat("gemini-2.5-flash").
    System("You are a helpful assistant.").
    User("What is the capital of France?").
    GetResponse(ctx)
```

**Step 3: Add CLI examples**

```bash
# Google Gemini
iris keys set gemini
iris chat --provider gemini --model gemini-2.5-flash --prompt "Hello"
iris chat --provider gemini --model gemini-2.5-flash --prompt "Hello" --stream
```

**Step 4: Commit**

```bash
git add README.md
git commit -m "docs: add Gemini provider documentation"
```

---

## Task 13: Final commit and PR preparation

**Step 1: Verify all files are committed**

Run: `git status`
Expected: Working tree clean

**Step 2: Run final test suite**

Run: `go test ./... -v && go vet ./... && gofmt -d .`
Expected: All pass, no output from gofmt

**Step 3: Create PR**

```bash
gh pr create --title "feat: add Google Gemini provider support" --body "$(cat <<'EOF'
## Summary
- Add Google Gemini provider with support for 5 models
- Support for chat, streaming, tool calling, and thinking/reasoning mode
- CLI integration with `iris chat --provider gemini`

## Models Supported
- gemini-3-pro-preview
- gemini-3-flash-preview
- gemini-2.5-flash (default)
- gemini-2.5-flash-lite
- gemini-2.5-pro

## Test plan
- [ ] Unit tests pass: `go test ./providers/gemini/...`
- [ ] Full test suite passes: `go test ./...`
- [ ] Manual CLI test: `iris chat --provider gemini --model gemini-2.5-flash --prompt "Hello"`
- [ ] Manual streaming test: `iris chat --provider gemini --model gemini-2.5-flash --prompt "Hello" --stream

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

---

Plan complete and saved to `docs/plans/2026-01-27-gemini-provider-implementation.md`. Two execution options:

**1. Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

Which approach?