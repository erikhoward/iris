# ADR 0002: PetalFlow Adapters for Provider and Tool Integration

## Status

Accepted

## Context

PetalFlow needs to integrate with existing Iris components:
- `core.Provider` for LLM calls (OpenAI, Anthropic, etc.)
- `tools.Tool` for tool execution

These interfaces have different signatures than what PetalFlow nodes need:
- `core.Provider.Chat()` returns `*core.ChatResponse` with `TokenUsage`
- `tools.Tool.Call()` takes `json.RawMessage` and returns `any`

PetalFlow nodes need:
- `LLMClient.Complete()` returning `LLMResponse` with PetalFlow-specific fields
- `PetalTool.Invoke()` taking/returning `map[string]any` for envelope integration

## Decision

We chose the **Adapter Pattern** to bridge these interfaces:

1. **ProviderAdapter**: Wraps `core.Provider` to implement `LLMClient`
   - Converts `LLMRequest` to `core.ChatRequest`
   - Converts `core.ChatResponse` to `LLMResponse`
   - Handles JSON schema parsing for structured output

2. **ToolAdapter**: Wraps `tools.Tool` to implement `PetalTool`
   - Converts `map[string]any` args to `json.RawMessage`
   - Converts result back to `map[string]any`
   - Handles primitive results by wrapping in `{"result": value}`

3. **FuncTool**: Lightweight inline tool for workflows
   - No adapter needed, directly implements `PetalTool`
   - Useful for custom logic without full Tool implementation

4. **ToolRegistry**: Name-based tool lookup
   - Enables dynamic tool selection in ToolNode
   - Supports late binding of tools

## Consequences

### Positive

- Clean separation between PetalFlow and Iris internals
- Existing providers/tools work without modification
- Easy to add new adapter implementations
- FuncTool simplifies workflow-specific tools
- Registry enables declarative tool configuration

### Negative

- Additional abstraction layer
- Some type conversion overhead (JSON round-trips)
- Need to maintain adapter code as underlying interfaces evolve

### Trade-offs

- Chose separate `LLMClient` interface over using `core.Provider` directly to:
  - Provide PetalFlow-specific request/response types
  - Enable non-Provider implementations (mock clients, caching layers)
  - Decouple PetalFlow from core package changes

- Chose `map[string]any` for tool args/results over typed structs to:
  - Support dynamic tool parameters from envelope variables
  - Simplify JSON schema validation
  - Match envelope variable storage format

## References

- `agents/petalflow/adapters/provider.go` - ProviderAdapter implementation
- `agents/petalflow/adapters/tool.go` - ToolAdapter, FuncTool, ToolRegistry
