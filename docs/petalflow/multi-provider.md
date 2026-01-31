# Multi-Provider Workflows with PetalFlow

This guide covers patterns for using multiple LLM providers in a single workflow for comparison, fallback, and specialized task routing.

## Available Providers

Iris supports these providers through the registry:

```go
import "github.com/erikhoward/iris/providers"

// List all registered providers
for _, name := range providers.List() {
    fmt.Println(name)
}

// Create provider by name
provider, err := providers.Create("anthropic", os.Getenv("ANTHROPIC_API_KEY"))
```

| Provider | ID | Best For |
|----------|-----|----------|
| OpenAI | `openai` | General purpose, embeddings, GPT-5 |
| Anthropic | `anthropic` | Complex reasoning, long context |
| Google Gemini | `gemini` | Multimodal, reasoning |
| xAI Grok | `xai` | Real-time knowledge, reasoning |
| Z.ai GLM | `zai` | Multilingual, thinking |
| Perplexity | `perplexity` | Web search integration |
| Ollama | `ollama` | Local/private deployment |
| HuggingFace | `huggingface` | Open source models |
| VoyageAI | `voyageai` | Embeddings, reranking |

## Competitive Evaluation

Compare responses from multiple providers and select the best.

```go
package main

import (
    "context"
    "os"

    "github.com/erikhoward/iris/agents/petalflow"
    "github.com/erikhoward/iris/agents/petalflow/adapters"
    "github.com/erikhoward/iris/providers"
)

func buildCompetitiveEvalWorkflow() (*petalflow.BasicGraph, error) {
    // Create multiple providers
    anthropic, _ := providers.Create("anthropic", os.Getenv("ANTHROPIC_API_KEY"))
    openai, _ := providers.Create("openai", os.Getenv("OPENAI_API_KEY"))
    gemini, _ := providers.Create("gemini", os.Getenv("GEMINI_API_KEY"))

    // Wrap as LLM clients
    claudeClient := adapters.NewProviderAdapter(anthropic)
    gptClient := adapters.NewProviderAdapter(openai)
    geminiClient := adapters.NewProviderAdapter(gemini)

    // Create LLM nodes for each provider
    claudeNode := petalflow.NewLLMNode("claude", claudeClient, petalflow.LLMNodeConfig{
        Model:        "claude-sonnet-4-20250514",
        SystemPrompt: "Provide a helpful, accurate response.",
        OutputVar:    "claude_response",
    })

    gptNode := petalflow.NewLLMNode("gpt", gptClient, petalflow.LLMNodeConfig{
        Model:        "gpt-4o",
        SystemPrompt: "Provide a helpful, accurate response.",
        OutputVar:    "gpt_response",
    })

    geminiNode := petalflow.NewLLMNode("gemini", geminiClient, petalflow.LLMNodeConfig{
        Model:        "gemini-2.0-flash",
        SystemPrompt: "Provide a helpful, accurate response.",
        OutputVar:    "gemini_response",
    })

    // Merge node with "best score" strategy
    mergeNode := petalflow.NewMergeNode("select_best", petalflow.MergeNodeConfig{
        Strategy: petalflow.NewBestScoreMergeStrategy(petalflow.BestScoreMergeConfig{
            ScoreVar:       "quality_score",
            HigherIsBetter: true,
        }),
    })

    // Evaluator node to score each response
    // In practice, this would use an evaluator LLM or custom scoring
    evaluatorNode := petalflow.NewFuncNode("evaluate", func(ctx context.Context, env *petalflow.Envelope) (*petalflow.Envelope, error) {
        // Score based on response length as a simple heuristic
        // Real implementation would use semantic evaluation
        score := 0.0
        if resp, ok := env.GetVar("response_text"); ok {
            text := resp.(string)
            score = float64(len(text)) / 1000.0 // Normalize
            if score > 1.0 {
                score = 1.0
            }
        }
        env.SetVar("quality_score", score)
        return env, nil
    })

    // Build with parallel execution
    return petalflow.NewGraphBuilder("competitive-eval").
        AddNode(petalflow.NewNoopNode("start")).
        FanOut(claudeNode, gptNode, geminiNode).
        Merge(mergeNode).
        Build()
}
```

## Fallback Chain

Try providers in order until one succeeds.

```go
func buildFallbackChainWorkflow() *petalflow.BasicGraph {
    // Primary: Claude (most capable)
    anthropic, _ := providers.Create("anthropic", os.Getenv("ANTHROPIC_API_KEY"))
    primaryClient := adapters.NewProviderAdapter(anthropic)

    // Secondary: GPT-4 (backup)
    openai, _ := providers.Create("openai", os.Getenv("OPENAI_API_KEY"))
    secondaryClient := adapters.NewProviderAdapter(openai)

    // Tertiary: Local Ollama (always available)
    ollama, _ := providers.Create("ollama", "http://localhost:11434")
    localClient := adapters.NewProviderAdapter(ollama)

    // Primary attempt
    primaryNode := petalflow.NewLLMNode("primary", primaryClient, petalflow.LLMNodeConfig{
        Model:     "claude-sonnet-4-20250514",
        OutputVar: "response",
        Retry: petalflow.RetryPolicy{
            MaxAttempts: 2,
        },
    })

    // Check if primary succeeded
    checkPrimary := petalflow.NewGateNode("check_primary", petalflow.GateNodeConfig{
        Condition: func(ctx context.Context, env *petalflow.Envelope) (bool, error) {
            _, hasResponse := env.GetVar("response")
            return hasResponse, nil
        },
        OnFail:         petalflow.GateActionRedirect,
        RedirectNodeID: "secondary",
    })

    // Secondary attempt
    secondaryNode := petalflow.NewLLMNode("secondary", secondaryClient, petalflow.LLMNodeConfig{
        Model:     "gpt-4o",
        OutputVar: "response",
        Retry: petalflow.RetryPolicy{
            MaxAttempts: 2,
        },
    })

    // Check if secondary succeeded
    checkSecondary := petalflow.NewGateNode("check_secondary", petalflow.GateNodeConfig{
        Condition: func(ctx context.Context, env *petalflow.Envelope) (bool, error) {
            _, hasResponse := env.GetVar("response")
            return hasResponse, nil
        },
        OnFail:         petalflow.GateActionRedirect,
        RedirectNodeID: "local",
    })

    // Local fallback (always available)
    localNode := petalflow.NewLLMNode("local", localClient, petalflow.LLMNodeConfig{
        Model:     "llama3.2",
        OutputVar: "response",
    })

    // Final output node
    outputNode := petalflow.NewNoopNode("output")

    graph := petalflow.NewGraph("fallback-chain")
    graph.AddNode(primaryNode)
    graph.AddNode(checkPrimary)
    graph.AddNode(secondaryNode)
    graph.AddNode(checkSecondary)
    graph.AddNode(localNode)
    graph.AddNode(outputNode)

    graph.AddEdge("primary", "check_primary")
    graph.AddEdge("check_primary", "output")     // Success path
    graph.AddEdge("check_primary", "secondary")   // Redirect on fail
    graph.AddEdge("secondary", "check_secondary")
    graph.AddEdge("check_secondary", "output")    // Success path
    graph.AddEdge("check_secondary", "local")     // Redirect on fail
    graph.AddEdge("local", "output")

    graph.SetEntry("primary")
    return graph
}
```

## Task-Specialized Routing

Route different tasks to the most appropriate provider.

```go
func buildSpecializedRoutingWorkflow() (*petalflow.BasicGraph, error) {
    // Claude: Complex reasoning
    anthropic, _ := providers.Create("anthropic", os.Getenv("ANTHROPIC_API_KEY"))
    reasoningClient := adapters.NewProviderAdapter(anthropic)

    // GPT-4: Code generation
    openai, _ := providers.Create("openai", os.Getenv("OPENAI_API_KEY"))
    codeClient := adapters.NewProviderAdapter(openai)

    // Perplexity: Real-time information
    perplexity, _ := providers.Create("perplexity", os.Getenv("PERPLEXITY_API_KEY"))
    searchClient := adapters.NewProviderAdapter(perplexity)

    // Ollama: Privacy-sensitive tasks
    ollama, _ := providers.Create("ollama", "http://localhost:11434")
    privateClient := adapters.NewProviderAdapter(ollama)

    // Classify the task type
    classifyNode := petalflow.NewLLMNode("classify", reasoningClient, petalflow.LLMNodeConfig{
        Model: "claude-3-5-haiku-20241022", // Fast, cheap for classification
        SystemPrompt: `Classify the user's request into one category:
- "reasoning": Complex analysis, math, logic
- "code": Programming, code generation
- "search": Current events, real-time info
- "private": Contains sensitive/personal data

Return JSON: {"category": "...", "confidence": 0.0-1.0}`,
        OutputVar: "classification",
    })

    // Router based on classification
    routerNode := petalflow.NewRouterNode("router", petalflow.RouterNodeConfig{
        Routes: []petalflow.Route{
            {Name: "reasoning", Target: "reasoning_llm"},
            {Name: "code", Target: "code_llm"},
            {Name: "search", Target: "search_llm"},
            {Name: "private", Target: "private_llm"},
        },
        InputVar: "classification",
        Strategy: petalflow.RouteStrategyFunction,
        Selector: func(env *petalflow.Envelope) ([]string, error) {
            class, _ := env.GetVar("classification")
            data := class.(map[string]any)
            return []string{data["category"].(string)}, nil
        },
    })

    // Specialized LLM nodes
    reasoningNode := petalflow.NewLLMNode("reasoning_llm", reasoningClient, petalflow.LLMNodeConfig{
        Model: "claude-sonnet-4-20250514",
        SystemPrompt: `You excel at complex reasoning and analysis.
Think step by step and show your work.`,
    })

    codeNode := petalflow.NewLLMNode("code_llm", codeClient, petalflow.LLMNodeConfig{
        Model: "gpt-4o",
        SystemPrompt: `You are an expert programmer.
Write clean, well-documented code with explanations.`,
    })

    searchNode := petalflow.NewLLMNode("search_llm", searchClient, petalflow.LLMNodeConfig{
        Model: "llama-3.1-sonar-large-128k-online",
        SystemPrompt: `You have access to real-time information.
Provide current, accurate information with sources.`,
    })

    privateNode := petalflow.NewLLMNode("private_llm", privateClient, petalflow.LLMNodeConfig{
        Model: "llama3.2",
        SystemPrompt: `Process this request privately.
Do not store or log any information.`,
    })

    // Merge all paths
    mergeNode := petalflow.NewMergeNode("merge", petalflow.MergeNodeConfig{
        Strategy: petalflow.NewJSONMergeStrategy(petalflow.JSONMergeConfig{}),
    })

    return petalflow.NewGraphBuilder("specialized-routing").
        AddNode(classifyNode).
        Edge(routerNode).
        FanOut(reasoningNode, codeNode, searchNode, privateNode).
        Merge(mergeNode).
        Build()
}
```

## Consensus Voting

Get multiple opinions and reach consensus.

```go
func buildConsensusWorkflow() (*petalflow.BasicGraph, error) {
    // Create multiple providers
    anthropic, _ := providers.Create("anthropic", os.Getenv("ANTHROPIC_API_KEY"))
    openai, _ := providers.Create("openai", os.Getenv("OPENAI_API_KEY"))
    gemini, _ := providers.Create("gemini", os.Getenv("GEMINI_API_KEY"))

    clients := []adapters.LLMClient{
        adapters.NewProviderAdapter(anthropic),
        adapters.NewProviderAdapter(openai),
        adapters.NewProviderAdapter(gemini),
    }

    // Create voting nodes for each provider
    claudeVote := petalflow.NewLLMNode("claude_vote", clients[0], petalflow.LLMNodeConfig{
        Model: "claude-sonnet-4-20250514",
        SystemPrompt: `Answer the question and rate your confidence.
Return JSON: {"answer": "yes" or "no", "reasoning": "...", "confidence": 0.0-1.0}`,
        OutputVar: "claude_vote",
    })

    gptVote := petalflow.NewLLMNode("gpt_vote", clients[1], petalflow.LLMNodeConfig{
        Model: "gpt-4o",
        SystemPrompt: `Answer the question and rate your confidence.
Return JSON: {"answer": "yes" or "no", "reasoning": "...", "confidence": 0.0-1.0}`,
        OutputVar: "gpt_vote",
    })

    geminiVote := petalflow.NewLLMNode("gemini_vote", clients[2], petalflow.LLMNodeConfig{
        Model: "gemini-2.0-flash",
        SystemPrompt: `Answer the question and rate your confidence.
Return JSON: {"answer": "yes" or "no", "reasoning": "...", "confidence": 0.0-1.0}`,
        OutputVar: "gemini_vote",
    })

    // Merge with custom consensus strategy
    consensusMerge := petalflow.NewMergeNode("consensus", petalflow.MergeNodeConfig{
        Strategy: petalflow.NewFuncMergeStrategy(func(ctx context.Context, inputs []*petalflow.Envelope) (*petalflow.Envelope, error) {
            result := petalflow.NewEnvelope()

            yesVotes := 0
            noVotes := 0
            var allVotes []map[string]any

            voteKeys := []string{"claude_vote", "gpt_vote", "gemini_vote"}
            for _, input := range inputs {
                for _, key := range voteKeys {
                    if vote, ok := input.GetVar(key); ok {
                        voteData := vote.(map[string]any)
                        allVotes = append(allVotes, voteData)

                        if voteData["answer"] == "yes" {
                            yesVotes++
                        } else {
                            noVotes++
                        }
                    }
                }
            }

            // Determine consensus
            consensus := "yes"
            if noVotes > yesVotes {
                consensus = "no"
            }

            result.SetVar("consensus", consensus)
            result.SetVar("yes_votes", yesVotes)
            result.SetVar("no_votes", noVotes)
            result.SetVar("all_votes", allVotes)
            result.SetVar("unanimous", yesVotes == 0 || noVotes == 0)

            return result, nil
        }),
    })

    // Synthesize final answer
    synthesizeNode := petalflow.NewLLMNode("synthesize", clients[0], petalflow.LLMNodeConfig{
        Model: "claude-sonnet-4-20250514",
        SystemPrompt: `Review the voting results and provide a final synthesis.
Explain the consensus and any dissenting opinions.`,
    })

    return petalflow.NewGraphBuilder("consensus-voting").
        AddNode(petalflow.NewNoopNode("start")).
        FanOut(claudeVote, gptVote, geminiVote).
        Merge(consensusMerge).
        Edge(synthesizeNode).
        Build()
}
```

## Cost Optimization

Route based on task complexity to optimize costs.

```go
func buildCostOptimizedWorkflow() (*petalflow.BasicGraph, error) {
    // Cheap tier: Fast, small models
    anthropic, _ := providers.Create("anthropic", os.Getenv("ANTHROPIC_API_KEY"))
    cheapClient := adapters.NewProviderAdapter(anthropic)

    // Expensive tier: Capable models
    expensiveClient := adapters.NewProviderAdapter(anthropic)

    // Classify complexity (use cheap model)
    classifyNode := petalflow.NewLLMNode("classify", cheapClient, petalflow.LLMNodeConfig{
        Model: "claude-3-5-haiku-20241022",
        SystemPrompt: `Rate the complexity of this request:
- "simple": Factual questions, simple tasks
- "moderate": Requires some reasoning
- "complex": Deep analysis, creative work

Return JSON: {"complexity": "simple"|"moderate"|"complex"}`,
        OutputVar: "complexity",
    })

    // Router based on complexity
    routerNode := petalflow.NewRouterNode("router", petalflow.RouterNodeConfig{
        Routes: []petalflow.Route{
            {Name: "simple", Target: "cheap_llm"},
            {Name: "moderate", Target: "cheap_llm"},
            {Name: "complex", Target: "expensive_llm"},
        },
        InputVar: "complexity",
        Strategy: petalflow.RouteStrategyFunction,
        Selector: func(env *petalflow.Envelope) ([]string, error) {
            data, _ := env.GetVar("complexity")
            complexity := data.(map[string]any)["complexity"].(string)
            if complexity == "complex" {
                return []string{"complex"}, nil
            }
            return []string{"simple"}, nil
        },
    })

    // Cheap LLM for simple/moderate tasks
    cheapNode := petalflow.NewLLMNode("cheap_llm", cheapClient, petalflow.LLMNodeConfig{
        Model: "claude-3-5-haiku-20241022",
    })

    // Expensive LLM for complex tasks
    expensiveNode := petalflow.NewLLMNode("expensive_llm", expensiveClient, petalflow.LLMNodeConfig{
        Model: "claude-sonnet-4-20250514",
    })

    // Merge paths
    mergeNode := petalflow.NewMergeNode("merge", petalflow.MergeNodeConfig{
        Strategy: petalflow.NewJSONMergeStrategy(petalflow.JSONMergeConfig{}),
    })

    return petalflow.NewGraphBuilder("cost-optimized").
        AddNode(classifyNode).
        Edge(routerNode).
        FanOut(cheapNode, expensiveNode).
        Merge(mergeNode).
        Build()
}
```

## A/B Testing Framework

Test different configurations and track metrics.

```go
func buildABTestWorkflow() *petalflow.BasicGraph {
    anthropic, _ := providers.Create("anthropic", os.Getenv("ANTHROPIC_API_KEY"))
    client := adapters.NewProviderAdapter(anthropic)

    // Variant A: Standard prompt
    variantA := petalflow.NewLLMNode("variant_a", client, petalflow.LLMNodeConfig{
        Model:        "claude-sonnet-4-20250514",
        SystemPrompt: "You are a helpful assistant.",
        OutputVar:    "response_a",
    })

    // Variant B: Chain of thought prompt
    variantB := petalflow.NewLLMNode("variant_b", client, petalflow.LLMNodeConfig{
        Model:        "claude-sonnet-4-20250514",
        SystemPrompt: "You are a helpful assistant. Think step by step before answering.",
        OutputVar:    "response_b",
    })

    // Random assignment (50/50 split)
    assignNode := petalflow.NewFuncNode("assign", func(ctx context.Context, env *petalflow.Envelope) (*petalflow.Envelope, error) {
        // In production, use proper randomization and tracking
        variant := "a"
        if time.Now().UnixNano()%2 == 0 {
            variant = "b"
        }
        env.SetVar("assigned_variant", variant)
        return env, nil
    })

    // Router to selected variant
    routerNode := petalflow.NewRouterNode("router", petalflow.RouterNodeConfig{
        Routes: []petalflow.Route{
            {Name: "a", Target: "variant_a"},
            {Name: "b", Target: "variant_b"},
        },
        InputVar: "assigned_variant",
        Strategy: petalflow.RouteStrategyFunction,
        Selector: func(env *petalflow.Envelope) ([]string, error) {
            variant, _ := env.GetVar("assigned_variant")
            return []string{variant.(string)}, nil
        },
    })

    // Log results for analysis
    logNode := petalflow.NewFuncNode("log_results", func(ctx context.Context, env *petalflow.Envelope) (*petalflow.Envelope, error) {
        variant, _ := env.GetVar("assigned_variant")
        // Log to your analytics system
        fmt.Printf("A/B Test - Variant: %s\n", variant)
        return env, nil
    })

    // Merge node
    mergeNode := petalflow.NewMergeNode("merge", petalflow.MergeNodeConfig{
        Strategy: petalflow.NewJSONMergeStrategy(petalflow.JSONMergeConfig{}),
    })

    graph := petalflow.NewGraph("ab-test")
    graph.AddNode(assignNode)
    graph.AddNode(routerNode)
    graph.AddNode(variantA)
    graph.AddNode(variantB)
    graph.AddNode(mergeNode)
    graph.AddNode(logNode)

    graph.AddEdge("assign", "router")
    graph.AddEdge("router", "variant_a")
    graph.AddEdge("router", "variant_b")
    graph.AddEdge("variant_a", "merge")
    graph.AddEdge("variant_b", "merge")
    graph.AddEdge("merge", "log_results")
    graph.SetEntry("assign")

    return graph
}
```

## Summary

Multi-provider workflows enable:

| Pattern | Use Case |
|---------|----------|
| Competitive Evaluation | Quality comparison, best-of-N |
| Fallback Chain | Reliability, availability |
| Specialized Routing | Cost optimization, task matching |
| Consensus Voting | Accuracy, reduce hallucination |
| Cost Optimization | Budget management |
| A/B Testing | Prompt engineering, model selection |

## Next Steps

- [Overview](./overview.md) - Return to PetalFlow fundamentals
- [LLM Workflows](./llm-workflows.md) - Single-provider patterns
- [RAG Pipelines](./rag-pipelines.md) - Retrieval-augmented generation
