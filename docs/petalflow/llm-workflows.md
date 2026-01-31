# LLM Workflows with PetalFlow

This guide covers common patterns for building LLM-powered workflows using PetalFlow with Iris providers.

## Basic Chat Completion

The simplest workflow: send a prompt, get a response.

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/erikhoward/iris/agents/petalflow"
    "github.com/erikhoward/iris/agents/petalflow/adapters"
    "github.com/erikhoward/iris/providers"
)

func main() {
    ctx := context.Background()

    // Create Anthropic provider
    provider, err := providers.Create("anthropic", os.Getenv("ANTHROPIC_API_KEY"))
    if err != nil {
        panic(err)
    }

    // Wrap as PetalFlow LLMClient
    client := adapters.NewProviderAdapter(provider)

    // Create LLM node
    chatNode := petalflow.NewLLMNode("chat", client, petalflow.LLMNodeConfig{
        Model:        "claude-sonnet-4-20250514",
        SystemPrompt: "You are a helpful assistant. Be concise.",
    })

    // Build graph
    graph := petalflow.NewGraph("basic-chat")
    graph.AddNode(chatNode)
    graph.SetEntry("chat")

    // Create envelope with user message
    env := petalflow.NewEnvelope()
    env.Messages = append(env.Messages, petalflow.Message{
        Role:    "user",
        Content: "What is the capital of France?",
    })

    // Execute
    rt := petalflow.NewRuntime()
    result, err := rt.Run(ctx, graph, env, petalflow.DefaultRunOptions())
    if err != nil {
        panic(err)
    }

    // Get response
    if len(result.Messages) > 0 {
        lastMsg := result.Messages[len(result.Messages)-1]
        fmt.Println("Response:", lastMsg.Content)
    }
}
```

## Multi-Turn Conversation

Chain multiple LLM calls for conversation flow.

```go
func buildConversationWorkflow(client adapters.LLMClient) *petalflow.BasicGraph {
    // Initial response node
    initialNode := petalflow.NewLLMNode("initial", client, petalflow.LLMNodeConfig{
        Model:        "claude-sonnet-4-20250514",
        SystemPrompt: "You are a friendly assistant.",
    })

    // Follow-up clarification node
    clarifyNode := petalflow.NewLLMNode("clarify", client, petalflow.LLMNodeConfig{
        Model:        "claude-sonnet-4-20250514",
        SystemPrompt: "Ask one clarifying question about the user's request.",
    })

    // Final response node
    finalNode := petalflow.NewLLMNode("final", client, petalflow.LLMNodeConfig{
        Model:        "claude-sonnet-4-20250514",
        SystemPrompt: "Provide a complete, helpful response.",
    })

    // Build linear graph: initial -> clarify -> final
    graph := petalflow.NewGraph("conversation")
    graph.AddNode(initialNode)
    graph.AddNode(clarifyNode)
    graph.AddNode(finalNode)
    graph.AddEdge("initial", "clarify")
    graph.AddEdge("clarify", "final")
    graph.SetEntry("initial")

    return graph
}
```

## Chain of Thought with Router

Use RouterNode to decide next steps based on LLM analysis.

```go
func buildChainOfThoughtWorkflow(client adapters.LLMClient) (*petalflow.BasicGraph, error) {
    // Analyze the question complexity
    analyzeNode := petalflow.NewLLMNode("analyze", client, petalflow.LLMNodeConfig{
        Model: "claude-sonnet-4-20250514",
        SystemPrompt: `Analyze the user's question.
Respond with JSON: {"complexity": "simple" | "complex", "reasoning": "..."}`,
        OutputVar: "analysis",
    })

    // Route based on complexity
    routerNode := petalflow.NewRouterNode("router", petalflow.RouterNodeConfig{
        Routes: []petalflow.Route{
            {
                Name:        "simple_path",
                Description: "For simple, factual questions",
                Target:      "simple_answer",
            },
            {
                Name:        "complex_path",
                Description: "For complex questions requiring reasoning",
                Target:      "deep_think",
            },
        },
        InputVar: "analysis",
        Strategy: petalflow.RouteStrategyLLM,
        LLM:      client,
    })

    // Simple answer path
    simpleNode := petalflow.NewLLMNode("simple_answer", client, petalflow.LLMNodeConfig{
        Model:        "claude-sonnet-4-20250514",
        SystemPrompt: "Give a brief, direct answer.",
    })

    // Complex reasoning path
    deepThinkNode := petalflow.NewLLMNode("deep_think", client, petalflow.LLMNodeConfig{
        Model: "claude-sonnet-4-20250514",
        SystemPrompt: `Think step by step:
1. Break down the problem
2. Consider multiple angles
3. Synthesize a comprehensive answer`,
    })

    // Build graph with branching
    return petalflow.NewGraphBuilder("chain-of-thought").
        AddNode(analyzeNode).
        Edge(routerNode).
        FanOut(simpleNode, deepThinkNode).
        Build()
}
```

## Tool-Augmented LLM

Combine LLM with tool execution for agentic workflows.

```go
func buildToolAugmentedWorkflow(client adapters.LLMClient) (*petalflow.BasicGraph, error) {
    // Create tools
    weatherTool := adapters.NewFuncTool(
        "get_weather",
        "Get current weather for a location",
        func(ctx context.Context, args map[string]any) (map[string]any, error) {
            location := args["location"].(string)
            // Simulated weather API call
            return map[string]any{
                "location":    location,
                "temperature": 72,
                "conditions":  "sunny",
            }, nil
        },
    )

    calculatorTool := adapters.NewFuncTool(
        "calculate",
        "Perform mathematical calculations",
        func(ctx context.Context, args map[string]any) (map[string]any, error) {
            expression := args["expression"].(string)
            // Simulated calculation
            return map[string]any{
                "expression": expression,
                "result":     42,
            }, nil
        },
    )

    // Tool registry
    registry := adapters.NewToolRegistry()
    registry.Register(weatherTool)
    registry.Register(calculatorTool)

    // Intent classifier
    classifyNode := petalflow.NewLLMNode("classify", client, petalflow.LLMNodeConfig{
        Model: "claude-sonnet-4-20250514",
        SystemPrompt: `Classify the user's intent. Return JSON:
{"intent": "weather" | "calculation" | "general", "params": {...}}`,
        OutputVar: "intent",
    })

    // Router based on intent
    routerNode := petalflow.NewRouterNode("router", petalflow.RouterNodeConfig{
        Routes: []petalflow.Route{
            {Name: "weather", Target: "weather_tool"},
            {Name: "calculation", Target: "calc_tool"},
            {Name: "general", Target: "general_response"},
        },
        InputVar: "intent",
        Strategy: petalflow.RouteStrategyFunction,
        Selector: func(env *petalflow.Envelope) ([]string, error) {
            intent, _ := env.GetVar("intent")
            data := intent.(map[string]any)
            return []string{data["intent"].(string)}, nil
        },
    })

    // Tool nodes
    weatherNode := petalflow.NewToolNode("weather_tool", weatherTool, petalflow.ToolNodeConfig{
        InputVar:  "intent",
        OutputVar: "tool_result",
    })

    calcNode := petalflow.NewToolNode("calc_tool", calculatorTool, petalflow.ToolNodeConfig{
        InputVar:  "intent",
        OutputVar: "tool_result",
    })

    // General LLM response
    generalNode := petalflow.NewLLMNode("general_response", client, petalflow.LLMNodeConfig{
        Model:        "claude-sonnet-4-20250514",
        SystemPrompt: "Respond helpfully to the user.",
    })

    // Final synthesis node
    synthesizeNode := petalflow.NewLLMNode("synthesize", client, petalflow.LLMNodeConfig{
        Model: "claude-sonnet-4-20250514",
        SystemPrompt: `Use the tool result to form a natural response.
Tool result is in the 'tool_result' variable.`,
    })

    // Merge tool outputs
    mergeNode := petalflow.NewMergeNode("merge", petalflow.MergeNodeConfig{
        Strategy: petalflow.NewJSONMergeStrategy(petalflow.JSONMergeConfig{}),
    })

    // Build complex graph
    graph := petalflow.NewGraph("tool-augmented")
    graph.AddNode(classifyNode)
    graph.AddNode(routerNode)
    graph.AddNode(weatherNode)
    graph.AddNode(calcNode)
    graph.AddNode(generalNode)
    graph.AddNode(mergeNode)
    graph.AddNode(synthesizeNode)

    graph.AddEdge("classify", "router")
    graph.AddEdge("router", "weather_tool")
    graph.AddEdge("router", "calc_tool")
    graph.AddEdge("router", "general_response")
    graph.AddEdge("weather_tool", "merge")
    graph.AddEdge("calc_tool", "merge")
    graph.AddEdge("general_response", "merge")
    graph.AddEdge("merge", "synthesize")
    graph.SetEntry("classify")

    return graph, nil
}
```

## Batch Processing with MapNode

Process multiple items through an LLM.

```go
func buildBatchSummaryWorkflow(client adapters.LLMClient) *petalflow.BasicGraph {
    // Setup node: prepare documents
    setupNode := petalflow.NewFuncNode("setup", func(ctx context.Context, env *petalflow.Envelope) (*petalflow.Envelope, error) {
        documents := []string{
            "Document 1: The quick brown fox jumps over the lazy dog.",
            "Document 2: Machine learning is transforming industries.",
            "Document 3: Climate change requires urgent action.",
        }
        env.SetVar("documents", documents)
        return env, nil
    })

    // Map node: summarize each document
    mapNode := petalflow.NewMapNode("summarize_each", petalflow.MapNodeConfig{
        InputVar:    "documents",
        OutputVar:   "summaries",
        ItemVar:     "document",
        Concurrency: 3, // Process 3 documents in parallel
        Mapper: func(ctx context.Context, item any, index int) (any, error) {
            doc := item.(string)

            // Create mini-workflow for each document
            env := petalflow.NewEnvelope()
            env.Messages = append(env.Messages, petalflow.Message{
                Role:    "user",
                Content: fmt.Sprintf("Summarize in one sentence: %s", doc),
            })

            // Direct LLM call (simplified)
            resp, err := client.Complete(ctx, adapters.LLMRequest{
                Model:    "claude-sonnet-4-20250514",
                Messages: []adapters.LLMMessage{{Role: "user", Content: doc}},
            })
            if err != nil {
                return nil, err
            }

            return map[string]any{
                "original": doc,
                "summary":  resp.Text,
            }, nil
        },
    })

    // Combine summaries
    combineNode := petalflow.NewLLMNode("combine", client, petalflow.LLMNodeConfig{
        Model: "claude-sonnet-4-20250514",
        SystemPrompt: `You have individual summaries in the 'summaries' variable.
Create a cohesive overview combining all summaries.`,
    })

    graph := petalflow.NewGraph("batch-summary")
    graph.AddNode(setupNode)
    graph.AddNode(mapNode)
    graph.AddNode(combineNode)
    graph.AddEdge("setup", "summarize_each")
    graph.AddEdge("summarize_each", "combine")
    graph.SetEntry("setup")

    return graph
}
```

## Provider Switching

Use different providers for different tasks.

```go
func buildMultiProviderWorkflow() (*petalflow.BasicGraph, error) {
    // Fast model for classification
    fastProvider, _ := providers.Create("anthropic", os.Getenv("ANTHROPIC_API_KEY"))
    fastClient := adapters.NewProviderAdapter(fastProvider)

    // Capable model for generation
    capableProvider, _ := providers.Create("openai", os.Getenv("OPENAI_API_KEY"))
    capableClient := adapters.NewProviderAdapter(capableProvider)

    // Local model for privacy-sensitive tasks
    localProvider, _ := providers.Create("ollama", "http://localhost:11434")
    localClient := adapters.NewProviderAdapter(localProvider)

    // Classification (fast, cheap)
    classifyNode := petalflow.NewLLMNode("classify", fastClient, petalflow.LLMNodeConfig{
        Model:     "claude-3-5-haiku-20241022",
        OutputVar: "classification",
    })

    // Generation (high quality)
    generateNode := petalflow.NewLLMNode("generate", capableClient, petalflow.LLMNodeConfig{
        Model: "gpt-4o",
    })

    // PII handling (local, private)
    sanitizeNode := petalflow.NewLLMNode("sanitize", localClient, petalflow.LLMNodeConfig{
        Model:        "llama3.2",
        SystemPrompt: "Remove any PII from the text. Keep the meaning intact.",
    })

    return petalflow.NewGraphBuilder("multi-provider").
        AddNode(sanitizeNode).
        Edge(classifyNode).
        Edge(generateNode).
        Build()
}
```

## Error Handling Patterns

### Retry with Fallback

```go
func buildRetryWorkflow(primaryClient, fallbackClient adapters.LLMClient) *petalflow.BasicGraph {
    // Primary LLM
    primaryNode := petalflow.NewLLMNode("primary", primaryClient, petalflow.LLMNodeConfig{
        Model: "gpt-4o",
        Retry: petalflow.RetryPolicy{
            MaxAttempts: 3,
            Backoff:     time.Second,
        },
    })

    // Gate to check if primary succeeded
    successGate := petalflow.NewGateNode("check_success", petalflow.GateNodeConfig{
        Condition: func(ctx context.Context, env *petalflow.Envelope) (bool, error) {
            _, hasResponse := env.GetVar("primary_output")
            return hasResponse, nil
        },
        OnFail:         petalflow.GateActionRedirect,
        RedirectNodeID: "fallback",
    })

    // Fallback LLM
    fallbackNode := petalflow.NewLLMNode("fallback", fallbackClient, petalflow.LLMNodeConfig{
        Model: "claude-sonnet-4-20250514",
    })

    graph := petalflow.NewGraph("retry-fallback")
    graph.AddNode(primaryNode)
    graph.AddNode(successGate)
    graph.AddNode(fallbackNode)
    graph.AddEdge("primary", "check_success")
    graph.AddEdge("check_success", "fallback") // Redirect path
    graph.SetEntry("primary")

    return graph
}
```

### Validation Gate

```go
func buildValidatedWorkflow(client adapters.LLMClient) *petalflow.BasicGraph {
    // Generate response
    generateNode := petalflow.NewLLMNode("generate", client, petalflow.LLMNodeConfig{
        Model: "claude-sonnet-4-20250514",
        SystemPrompt: `Generate a JSON response with fields:
{"answer": "...", "confidence": 0.0-1.0}`,
        OutputVar: "generated",
    })

    // Validate the response
    validateGate := petalflow.NewGateNode("validate", petalflow.GateNodeConfig{
        Condition: func(ctx context.Context, env *petalflow.Envelope) (bool, error) {
            gen, ok := env.GetVar("generated")
            if !ok {
                return false, nil
            }
            data, ok := gen.(map[string]any)
            if !ok {
                return false, nil
            }
            confidence, ok := data["confidence"].(float64)
            return ok && confidence >= 0.7, nil
        },
        OnFail:      petalflow.GateActionBlock,
        FailMessage: "low confidence response",
    })

    graph := petalflow.NewGraph("validated")
    graph.AddNode(generateNode)
    graph.AddNode(validateGate)
    graph.AddEdge("generate", "validate")
    graph.SetEntry("generate")

    return graph
}
```

## Next Steps

- [RAG Pipelines](./rag-pipelines.md) - Add retrieval and embeddings
- [Multi-Provider Evaluation](./multi-provider.md) - Compare provider outputs
