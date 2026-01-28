# Google Gemini Provider Implementation Design

## Overview

Implement the Google Gemini provider for iris, following the established OpenAI and Anthropic provider patterns.

## Models

| Model ID | Generation | Default Thinking |
|----------|------------|------------------|
| `gemini-3-pro-preview` | 3 | Cannot disable |
| `gemini-3-flash-preview` | 3 | Can use "minimal" |
| `gemini-2.5-flash` | 2.5 | Budget-based |
| `gemini-2.5-flash-lite` | 2.5 | Budget-based |
| `gemini-2.5-pro` | 2.5 | Budget-based |

**Default model for CLI:** `gemini-2.5-flash`

## API Details

- **Base URL:** `https://generativelanguage.googleapis.com`
- **Non-streaming:** `POST /v1beta/models/{model}:generateContent`
- **Streaming:** `POST /v1beta/models/{model}:streamGenerateContent?alt=sse`
- **Auth header:** `x-goog-api-key: {API_KEY}`

## File Structure

```
providers/gemini/
  provider.go        # Provider struct, ID(), Models(), Supports(), Chat(), StreamChat()
  options.go         # Config struct and functional options
  models.go          # Model definitions with thinking configurations
  client.go          # Non-streaming chat implementation
  streaming.go       # SSE streaming implementation
  mapping.go         # Request/response mapping (iris ↔ Gemini)
  errors.go          # Error normalization
  types_gemini.go    # Gemini API request/response types
```

## Type Definitions

### Request Types

```go
type geminiRequest struct {
    Contents          []geminiContent      `json:"contents"`
    SystemInstruction *geminiContent       `json:"system_instruction,omitempty"`
    GenerationConfig  *geminiGenConfig     `json:"generationConfig,omitempty"`
    Tools             []geminiTool         `json:"tools,omitempty"`
}

type geminiContent struct {
    Role  string       `json:"role,omitempty"` // "user" or "model"
    Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
    Text             string                `json:"text,omitempty"`
    FunctionCall     *geminiFunctionCall   `json:"functionCall,omitempty"`
    FunctionResponse *geminiFunctionResp   `json:"functionResponse,omitempty"`
    Thought          *bool                 `json:"thought,omitempty"`
}

type geminiGenConfig struct {
    Temperature     *float32           `json:"temperature,omitempty"`
    MaxOutputTokens *int               `json:"maxOutputTokens,omitempty"`
    ThinkingConfig  *geminiThinkConfig `json:"thinkingConfig,omitempty"`
}

type geminiThinkConfig struct {
    ThinkingLevel   string `json:"thinkingLevel,omitempty"`   // Gemini 3: "low"/"medium"/"high"
    ThinkingBudget  *int   `json:"thinkingBudget,omitempty"`  // Gemini 2.5: token count
    IncludeThoughts bool   `json:"includeThoughts,omitempty"`
}

type geminiTool struct {
    FunctionDeclarations []geminiFunctionDecl `json:"functionDeclarations"`
}

type geminiFunctionDecl struct {
    Name        string          `json:"name"`
    Description string          `json:"description"`
    Parameters  json.RawMessage `json:"parameters"`
}

type geminiFunctionCall struct {
    Name string          `json:"name"`
    Args json.RawMessage `json:"args"`
}

type geminiFunctionResp struct {
    Name     string          `json:"name"`
    Response json.RawMessage `json:"response"`
}
```

### Response Types

```go
type geminiResponse struct {
    Candidates    []geminiCandidate `json:"candidates"`
    UsageMetadata *geminiUsage      `json:"usageMetadata,omitempty"`
}

type geminiCandidate struct {
    Content      geminiContent `json:"content"`
    FinishReason string        `json:"finishReason,omitempty"`
}

type geminiUsage struct {
    PromptTokenCount     int `json:"promptTokenCount"`
    CandidatesTokenCount int `json:"candidatesTokenCount"`
    ThoughtsTokenCount   int `json:"thoughtsTokenCount,omitempty"`
}
```

### Error Types

```go
type geminiErrorResponse struct {
    Error geminiError `json:"error"`
}

type geminiError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Status  string `json:"status"`
}
```

## Thinking/Reasoning Configuration

### ReasoningEffort Mapping

| iris ReasoningEffort | Gemini 3 (thinkingLevel) | Gemini 2.5 (thinkingBudget) |
|---------------------|--------------------------|----------------------------|
| None | "minimal" (Flash) / "low" (Pro) | 0 |
| Low | "low" | 1024 |
| Medium | "medium" | 8192 |
| High | "high" | 24576 |
| XHigh | "high" | 32768 |

**Special cases:**
- Gemini 3 Pro cannot disable thinking; use "low" for `ReasoningEffort=None`
- Gemini 3 Flash supports "minimal" level
- All thinking responses set `includeThoughts: true`

## Request/Response Mapping

### Request Mapping (iris → Gemini)

1. **System messages:** Extract from `Messages`, concatenate with `\n\n`, put in `system_instruction.parts[0].text`

2. **Role mapping:**
   - `core.RoleUser` → `"user"`
   - `core.RoleAssistant` → `"model"`

3. **Tools:** Map `core.Tool` to `geminiTool.FunctionDeclarations`

4. **Generation config:** Map `Temperature`, `MaxTokens`, build `ThinkingConfig` based on model

### Response Mapping (Gemini → iris)

1. **Output:** Extract text from `candidates[0].content.parts` (non-thought parts)
2. **Tool calls:** Extract `functionCall` parts, generate IDs as `call_{index}`
3. **Usage:** Map token counts
4. **Reasoning:** Extract thought parts into `ChatResponse.Reasoning`

## Streaming

**SSE Format:** Each event contains partial `geminiResponse`:
```
data: {"candidates":[{"content":{"parts":[{"text":"partial"}]}}]}
```

**Processing:**
1. Parse SSE `data:` lines as `geminiResponse`
2. Text parts → emit `ChatChunk{Delta: text}`
3. Function calls → accumulate for final response
4. Thought parts → accumulate for reasoning output
5. Final event → send `ChatResponse` with all accumulated data

## Error Handling

| HTTP Status | Gemini Status | iris Error |
|-------------|---------------|------------|
| 400 | INVALID_ARGUMENT | `ErrBadRequest` |
| 401 | UNAUTHENTICATED | `ErrUnauthorized` |
| 403 | PERMISSION_DENIED | `ErrUnauthorized` |
| 404 | NOT_FOUND | `ErrBadRequest` |
| 429 | RESOURCE_EXHAUSTED | `ErrRateLimited` |
| 500+ | INTERNAL, UNAVAILABLE | `ErrServer` |

## CLI Integration

**chat.go:** Add `gemini` case to `createProvider()`

**init.go:** Add default model `gemini-2.5-flash`

**Environment variable:** `GEMINI_API_KEY`

## Testing

- Unit tests with `httptest.NewServer` mocks
- Test thinking config differs between Gemini 3 and 2.5
- Test tool call ID generation
- Test SSE parsing with multi-part responses
- Manual integration tests with real API key
