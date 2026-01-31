# PetalFlow Overview

PetalFlow is a workflow orchestration framework for building LLM-powered agent pipelines. It provides a graph-based execution model with first-class support for Iris providers, tools, and embeddings.

## Core Concepts

### Nodes

Nodes are the building blocks of PetalFlow workflows. Each node performs a specific task and passes an envelope to the next node.

| Node Type | Purpose | Use Case |
|-----------|---------|----------|
| `LLMNode` | LLM completion | Chat, generation, analysis |
| `ToolNode` | Tool/function execution | API calls, calculations |
| `RouterNode` | Conditional branching | Intent classification, routing |
| `MergeNode` | Combine parallel results | Fan-in from parallel branches |
| `MapNode` | Collection iteration | Batch processing |
| `GateNode` | Conditional guards | Validation, authorization |
| `NoopNode` | Pass-through | Placeholders, debugging |

### Envelope

The `Envelope` carries data between nodes:

```go
type Envelope struct {
    Messages  []Message           // Chat history
    Vars      map[string]any      // Variables for data passing
    Artifacts []Artifact          // Documents, images, etc.
    Trace     TraceInfo           // Run ID, span ID, timing
    Errors    []NodeError         // Recorded errors (ContinueOnError mode)
}
```

### Graph

A `Graph` defines the workflow structure:

```go
// Create graph
g := petalflow.NewGraph("my-workflow")

// Add nodes
g.AddNode(node1)
g.AddNode(node2)

// Connect nodes
g.AddEdge("node1", "node2")

// Set entry point
g.SetEntry("node1")
```

### GraphBuilder

Fluent API for complex workflows:

```go
graph, err := petalflow.NewGraphBuilder("workflow").
    AddNode(inputNode).
    Edge(processNode).
    FanOut(branchA, branchB).
    Merge(combineNode).
    Edge(outputNode).
    Build()
```

### Runtime

The runtime executes graphs:

```go
rt := petalflow.NewRuntime()

opts := petalflow.RunOptions{
    MaxHops:         100,        // Prevent infinite loops
    ContinueOnError: false,      // Stop on first error
    Concurrency:     4,          // Parallel execution workers
}

result, err := rt.Run(ctx, graph, envelope, opts)
```

## Iris Integration

### Provider Adapter

Connect Iris providers to PetalFlow:

```go
import (
    "github.com/erikhoward/iris/providers"
    "github.com/erikhoward/iris/agents/petalflow/adapters"
)

// Create Iris provider
provider, _ := providers.Create("anthropic", os.Getenv("ANTHROPIC_API_KEY"))

// Wrap as LLMClient for PetalFlow
client := adapters.NewProviderAdapter(provider)

// Use in LLMNode
llmNode := petalflow.NewLLMNode("chat", client, petalflow.LLMNodeConfig{
    Model:       "claude-sonnet-4-20250514",
    SystemPrompt: "You are a helpful assistant.",
})
```

### Available Providers

| Provider | ID | Features |
|----------|-----|----------|
| OpenAI | `openai` | Chat, Streaming, Tools, Embeddings |
| Anthropic | `anthropic` | Chat, Streaming, Tools |
| Google Gemini | `gemini` | Chat, Streaming, Tools, Reasoning |
| xAI Grok | `xai` | Chat, Streaming, Tools, Reasoning |
| Z.ai GLM | `zai` | Chat, Streaming, Tools, Thinking |
| Perplexity | `perplexity` | Chat, Streaming, Web Search |
| Ollama | `ollama` | Chat, Streaming, Local Models |
| HuggingFace | `huggingface` | Chat, Streaming |
| VoyageAI | `voyageai` | Embeddings, Reranking |

### Tool Adapter

Connect Iris tools to PetalFlow:

```go
import "github.com/erikhoward/iris/agents/petalflow/adapters"

// Wrap existing tool
petalTool := adapters.NewToolAdapter(existingTool)

// Or create inline function tool
searchTool := adapters.NewFuncTool(
    "web_search",
    "Search the web for information",
    func(ctx context.Context, args map[string]any) (map[string]any, error) {
        query := args["query"].(string)
        results := performSearch(query)
        return map[string]any{"results": results}, nil
    },
)

// Use in ToolNode
toolNode := petalflow.NewToolNode("search", petalTool, petalflow.ToolNodeConfig{
    InputVar:  "search_query",
    OutputVar: "search_results",
})
```

## Event System

PetalFlow emits events during execution for observability:

```go
rt := petalflow.NewRuntime()

// Option 1: Event handler callback
opts := petalflow.RunOptions{
    EventHandler: func(e petalflow.Event) {
        fmt.Printf("[%s] %s: %s\n", e.Type, e.NodeID, e.Payload)
    },
}

// Option 2: Event channel
go func() {
    for event := range rt.Events() {
        log.Printf("Event: %+v", event)
    }
}()
```

Event types:
- `EventRunStarted` / `EventRunFinished`
- `EventNodeStarted` / `EventNodeFinished`
- `EventRouteDecision`
- `EventMergeComplete`
- `EventToolInvoked`
- `EventLLMRequest` / `EventLLMResponse`

## Error Handling

### Fail Fast (Default)

```go
opts := petalflow.RunOptions{
    ContinueOnError: false,
}
// Execution stops on first error
```

### Record and Continue

```go
opts := petalflow.RunOptions{
    ContinueOnError: true,
}
// Errors recorded in envelope.Errors, execution continues
```

### Gate-Based Validation

```go
gate := petalflow.NewGateNode("validate", petalflow.GateNodeConfig{
    Condition: func(ctx context.Context, env *petalflow.Envelope) (bool, error) {
        input, _ := env.GetVar("user_input")
        return isValid(input), nil
    },
    OnFail:      petalflow.GateActionBlock,
    FailMessage: "invalid input",
})
```

## Next Steps

See the use case guides for detailed examples:
- [LLM Workflows](./llm-workflows.md) - Basic chat and generation patterns
- [RAG Pipelines](./rag-pipelines.md) - Embeddings, retrieval, and reranking
- [Multi-Provider Evaluation](./multi-provider.md) - Competitive LLM comparison
