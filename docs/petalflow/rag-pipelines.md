# RAG Pipelines with PetalFlow

This guide covers Retrieval-Augmented Generation (RAG) patterns using PetalFlow with Iris embedding and reranking providers.

## Iris Embedding Providers

Iris supports embeddings through providers that implement the `EmbeddingProvider` interface:

| Provider | Models | Features |
|----------|--------|----------|
| OpenAI | `text-embedding-3-small`, `text-embedding-3-large` | High quality, configurable dimensions |
| VoyageAI | `voyage-3`, `voyage-3-lite`, `voyage-code-3` | Specialized models, contextualized embeddings |

### Embedding Request Types

```go
// Standard embedding
req := &core.EmbeddingRequest{
    Model: "text-embedding-3-small",
    Input: []core.EmbeddingInput{
        {Text: "Document text here"},
    },
    InputType: core.InputTypeDocument, // or InputTypeQuery
}

// Contextualized embedding (VoyageAI)
req := &core.ContextualizedEmbeddingRequest{
    Model: "voyage-3",
    Input: [][]string{
        {"document context", "chunk 1", "chunk 2"},
    },
}
```

## Basic RAG Pipeline

A simple retrieve-and-generate workflow.

```go
package main

import (
    "context"
    "fmt"
    "math"
    "os"

    "github.com/erikhoward/iris/agents/petalflow"
    "github.com/erikhoward/iris/agents/petalflow/adapters"
    "github.com/erikhoward/iris/core"
    "github.com/erikhoward/iris/providers"
)

// SimpleVectorStore is an in-memory store for demo purposes
type SimpleVectorStore struct {
    embeddings []struct {
        ID     string
        Text   string
        Vector []float32
    }
    embeddingProvider core.EmbeddingProvider
    model             string
}

func NewSimpleVectorStore(provider core.EmbeddingProvider, model string) *SimpleVectorStore {
    return &SimpleVectorStore{
        embeddingProvider: provider,
        model:             model,
    }
}

func (s *SimpleVectorStore) Add(ctx context.Context, id, text string) error {
    resp, err := s.embeddingProvider.CreateEmbeddings(ctx, &core.EmbeddingRequest{
        Model:     core.ModelID(s.model),
        Input:     []core.EmbeddingInput{{Text: text, ID: id}},
        InputType: core.InputTypeDocument,
    })
    if err != nil {
        return err
    }

    s.embeddings = append(s.embeddings, struct {
        ID     string
        Text   string
        Vector []float32
    }{
        ID:     id,
        Text:   text,
        Vector: resp.Vectors[0].Vector,
    })
    return nil
}

func (s *SimpleVectorStore) Search(ctx context.Context, query string, topK int) ([]string, error) {
    // Embed query
    resp, err := s.embeddingProvider.CreateEmbeddings(ctx, &core.EmbeddingRequest{
        Model:     core.ModelID(s.model),
        Input:     []core.EmbeddingInput{{Text: query}},
        InputType: core.InputTypeQuery,
    })
    if err != nil {
        return nil, err
    }
    queryVec := resp.Vectors[0].Vector

    // Calculate cosine similarity for each document
    type scored struct {
        text  string
        score float64
    }
    var results []scored

    for _, doc := range s.embeddings {
        score := cosineSimilarity(queryVec, doc.Vector)
        results = append(results, scored{text: doc.Text, score: score})
    }

    // Sort by score descending
    for i := 0; i < len(results)-1; i++ {
        for j := i + 1; j < len(results); j++ {
            if results[j].score > results[i].score {
                results[i], results[j] = results[j], results[i]
            }
        }
    }

    // Return top K
    var texts []string
    for i := 0; i < topK && i < len(results); i++ {
        texts = append(texts, results[i].text)
    }
    return texts, nil
}

func cosineSimilarity(a, b []float32) float64 {
    var dot, normA, normB float64
    for i := range a {
        dot += float64(a[i]) * float64(b[i])
        normA += float64(a[i]) * float64(a[i])
        normB += float64(b[i]) * float64(b[i])
    }
    if normA == 0 || normB == 0 {
        return 0
    }
    return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func buildBasicRAGWorkflow(
    llmClient adapters.LLMClient,
    vectorStore *SimpleVectorStore,
) *petalflow.BasicGraph {
    // Retrieve relevant documents
    retrieveNode := petalflow.NewFuncNode("retrieve", func(ctx context.Context, env *petalflow.Envelope) (*petalflow.Envelope, error) {
        query, _ := env.GetVar("query")
        docs, err := vectorStore.Search(ctx, query.(string), 3)
        if err != nil {
            return nil, err
        }
        env.SetVar("retrieved_docs", docs)
        return env, nil
    })

    // Format context for LLM
    formatNode := petalflow.NewFuncNode("format", func(ctx context.Context, env *petalflow.Envelope) (*petalflow.Envelope, error) {
        docs, _ := env.GetVar("retrieved_docs")
        query, _ := env.GetVar("query")

        context := "Relevant documents:\n"
        for i, doc := range docs.([]string) {
            context += fmt.Sprintf("\n[%d] %s\n", i+1, doc)
        }

        env.Messages = append(env.Messages, petalflow.Message{
            Role: "user",
            Content: fmt.Sprintf(`%s

Based on the above context, answer this question: %s

If the context doesn't contain relevant information, say so.`, context, query),
        })
        return env, nil
    })

    // Generate answer with LLM
    generateNode := petalflow.NewLLMNode("generate", llmClient, petalflow.LLMNodeConfig{
        Model: "claude-sonnet-4-20250514",
        SystemPrompt: `You are a helpful assistant that answers questions based on provided context.
Always cite which document(s) you used in your answer.`,
    })

    graph := petalflow.NewGraph("basic-rag")
    graph.AddNode(retrieveNode)
    graph.AddNode(formatNode)
    graph.AddNode(generateNode)
    graph.AddEdge("retrieve", "format")
    graph.AddEdge("format", "generate")
    graph.SetEntry("retrieve")

    return graph
}

func main() {
    ctx := context.Background()

    // Setup embedding provider (OpenAI)
    embProvider, _ := providers.Create("openai", os.Getenv("OPENAI_API_KEY"))
    embeddingProvider := embProvider.(core.EmbeddingProvider)

    // Setup LLM provider
    llmProvider, _ := providers.Create("anthropic", os.Getenv("ANTHROPIC_API_KEY"))
    llmClient := adapters.NewProviderAdapter(llmProvider)

    // Create vector store
    store := NewSimpleVectorStore(embeddingProvider, "text-embedding-3-small")

    // Index some documents
    store.Add(ctx, "1", "Paris is the capital of France. It is known for the Eiffel Tower.")
    store.Add(ctx, "2", "Tokyo is the capital of Japan. It has a population of over 13 million.")
    store.Add(ctx, "3", "The Great Wall of China is over 13,000 miles long.")

    // Build and run RAG pipeline
    graph := buildBasicRAGWorkflow(llmClient, store)

    env := petalflow.NewEnvelope()
    env.SetVar("query", "What is the capital of France and what is it known for?")

    rt := petalflow.NewRuntime()
    result, err := rt.Run(ctx, graph, env, petalflow.DefaultRunOptions())
    if err != nil {
        panic(err)
    }

    // Print answer
    lastMsg := result.Messages[len(result.Messages)-1]
    fmt.Println("Answer:", lastMsg.Content)
}
```

## RAG with Reranking

Improve retrieval quality with a reranking step.

```go
func buildRAGWithReranking(
    llmClient adapters.LLMClient,
    embeddingProvider core.EmbeddingProvider,
    rerankerProvider core.RerankerProvider,
    vectorStore *SimpleVectorStore,
) (*petalflow.BasicGraph, error) {
    // Initial retrieval (get more candidates)
    retrieveNode := petalflow.NewFuncNode("retrieve", func(ctx context.Context, env *petalflow.Envelope) (*petalflow.Envelope, error) {
        query, _ := env.GetVar("query")
        // Retrieve more documents than needed for reranking
        docs, err := vectorStore.Search(ctx, query.(string), 10)
        if err != nil {
            return nil, err
        }
        env.SetVar("candidate_docs", docs)
        return env, nil
    })

    // Rerank candidates
    rerankNode := petalflow.NewFuncNode("rerank", func(ctx context.Context, env *petalflow.Envelope) (*petalflow.Envelope, error) {
        query, _ := env.GetVar("query")
        candidates, _ := env.GetVar("candidate_docs")

        topK := 3
        resp, err := rerankerProvider.Rerank(ctx, &core.RerankRequest{
            Model:           "rerank-2",
            Query:           query.(string),
            Documents:       candidates.([]string),
            TopK:            &topK,
            ReturnDocuments: true,
        })
        if err != nil {
            return nil, err
        }

        // Extract reranked documents
        var rerankedDocs []string
        for _, result := range resp.Results {
            rerankedDocs = append(rerankedDocs, result.Document)
        }

        env.SetVar("retrieved_docs", rerankedDocs)
        env.SetVar("rerank_scores", resp.Results) // For debugging
        return env, nil
    })

    // Format and generate (same as basic RAG)
    formatNode := petalflow.NewFuncNode("format", func(ctx context.Context, env *petalflow.Envelope) (*petalflow.Envelope, error) {
        docs, _ := env.GetVar("retrieved_docs")
        query, _ := env.GetVar("query")

        context := "Relevant documents (ranked by relevance):\n"
        for i, doc := range docs.([]string) {
            context += fmt.Sprintf("\n[%d] %s\n", i+1, doc)
        }

        env.Messages = append(env.Messages, petalflow.Message{
            Role:    "user",
            Content: fmt.Sprintf("%s\n\nQuestion: %s", context, query),
        })
        return env, nil
    })

    generateNode := petalflow.NewLLMNode("generate", llmClient, petalflow.LLMNodeConfig{
        Model:        "claude-sonnet-4-20250514",
        SystemPrompt: "Answer based on the provided context. Cite your sources.",
    })

    return petalflow.NewGraphBuilder("rag-with-rerank").
        AddNode(retrieveNode).
        Edge(rerankNode).
        Edge(formatNode).
        Edge(generateNode).
        Build()
}
```

## Hybrid Search Pipeline

Combine semantic and keyword search.

```go
func buildHybridSearchPipeline(
    llmClient adapters.LLMClient,
    vectorStore *SimpleVectorStore,
    keywordSearchFn func(query string, topK int) []string,
    rerankerProvider core.RerankerProvider,
) (*petalflow.BasicGraph, error) {
    // Parallel retrieval: semantic + keyword
    semanticNode := petalflow.NewFuncNode("semantic_search", func(ctx context.Context, env *petalflow.Envelope) (*petalflow.Envelope, error) {
        query, _ := env.GetVar("query")
        docs, err := vectorStore.Search(ctx, query.(string), 5)
        if err != nil {
            return nil, err
        }
        env.SetVar("semantic_results", docs)
        return env, nil
    })

    keywordNode := petalflow.NewFuncNode("keyword_search", func(ctx context.Context, env *petalflow.Envelope) (*petalflow.Envelope, error) {
        query, _ := env.GetVar("query")
        docs := keywordSearchFn(query.(string), 5)
        env.SetVar("keyword_results", docs)
        return env, nil
    })

    // Merge results
    mergeNode := petalflow.NewMergeNode("merge_results", petalflow.MergeNodeConfig{
        Strategy: petalflow.NewFuncMergeStrategy(func(ctx context.Context, inputs []*petalflow.Envelope) (*petalflow.Envelope, error) {
            result := petalflow.NewEnvelope()

            // Collect all documents, deduplicate
            seen := make(map[string]bool)
            var allDocs []string

            for _, input := range inputs {
                if semantic, ok := input.GetVar("semantic_results"); ok {
                    for _, doc := range semantic.([]string) {
                        if !seen[doc] {
                            seen[doc] = true
                            allDocs = append(allDocs, doc)
                        }
                    }
                }
                if keyword, ok := input.GetVar("keyword_results"); ok {
                    for _, doc := range keyword.([]string) {
                        if !seen[doc] {
                            seen[doc] = true
                            allDocs = append(allDocs, doc)
                        }
                    }
                }
                // Preserve query
                if q, ok := input.GetVar("query"); ok {
                    result.SetVar("query", q)
                }
            }

            result.SetVar("merged_candidates", allDocs)
            return result, nil
        }),
    })

    // Rerank merged results
    rerankNode := petalflow.NewFuncNode("rerank", func(ctx context.Context, env *petalflow.Envelope) (*petalflow.Envelope, error) {
        query, _ := env.GetVar("query")
        candidates, _ := env.GetVar("merged_candidates")

        topK := 5
        resp, err := rerankerProvider.Rerank(ctx, &core.RerankRequest{
            Model:           "rerank-2",
            Query:           query.(string),
            Documents:       candidates.([]string),
            TopK:            &topK,
            ReturnDocuments: true,
        })
        if err != nil {
            return nil, err
        }

        var docs []string
        for _, r := range resp.Results {
            docs = append(docs, r.Document)
        }
        env.SetVar("final_docs", docs)
        return env, nil
    })

    // Generate answer
    generateNode := petalflow.NewLLMNode("generate", llmClient, petalflow.LLMNodeConfig{
        Model:        "claude-sonnet-4-20250514",
        SystemPrompt: "Answer based on the provided context.",
    })

    // Build graph with fan-out for parallel search
    graph := petalflow.NewGraph("hybrid-search")

    // Entry point that clones query to both paths
    entryNode := petalflow.NewNoopNode("entry")
    graph.AddNode(entryNode)
    graph.AddNode(semanticNode)
    graph.AddNode(keywordNode)
    graph.AddNode(mergeNode)
    graph.AddNode(rerankNode)
    graph.AddNode(generateNode)

    graph.AddEdge("entry", "semantic_search")
    graph.AddEdge("entry", "keyword_search")
    graph.AddEdge("semantic_search", "merge_results")
    graph.AddEdge("keyword_search", "merge_results")
    graph.AddEdge("merge_results", "rerank")
    graph.AddEdge("rerank", "generate")
    graph.SetEntry("entry")

    return graph, nil
}
```

## Multi-Query RAG

Generate multiple queries to improve retrieval coverage.

```go
func buildMultiQueryRAG(
    llmClient adapters.LLMClient,
    vectorStore *SimpleVectorStore,
) (*petalflow.BasicGraph, error) {
    // Generate query variations
    queryGenNode := petalflow.NewLLMNode("generate_queries", llmClient, petalflow.LLMNodeConfig{
        Model: "claude-sonnet-4-20250514",
        SystemPrompt: `Generate 3 different versions of the user's query to improve search coverage.
Return as JSON: {"queries": ["query1", "query2", "query3"]}`,
        OutputVar: "query_variations",
    })

    // Map over queries to retrieve for each
    mapRetrieveNode := petalflow.NewMapNode("retrieve_each", petalflow.MapNodeConfig{
        InputVar:    "queries",
        OutputVar:   "all_results",
        ItemVar:     "search_query",
        Concurrency: 3,
        Mapper: func(ctx context.Context, item any, index int) (any, error) {
            query := item.(string)
            docs, err := vectorStore.Search(ctx, query, 3)
            return docs, err
        },
    })

    // Parse queries from LLM output
    parseNode := petalflow.NewFuncNode("parse_queries", func(ctx context.Context, env *petalflow.Envelope) (*petalflow.Envelope, error) {
        variations, _ := env.GetVar("query_variations")
        data := variations.(map[string]any)
        queries := data["queries"].([]any)

        // Convert to []string
        var queryStrings []string
        for _, q := range queries {
            queryStrings = append(queryStrings, q.(string))
        }
        env.SetVar("queries", queryStrings)
        return env, nil
    })

    // Deduplicate and combine results
    combineNode := petalflow.NewFuncNode("combine", func(ctx context.Context, env *petalflow.Envelope) (*petalflow.Envelope, error) {
        allResults, _ := env.GetVar("all_results")
        seen := make(map[string]bool)
        var uniqueDocs []string

        for _, result := range allResults.([]any) {
            for _, doc := range result.([]string) {
                if !seen[doc] {
                    seen[doc] = true
                    uniqueDocs = append(uniqueDocs, doc)
                }
            }
        }

        env.SetVar("retrieved_docs", uniqueDocs)
        return env, nil
    })

    // Generate final answer
    generateNode := petalflow.NewLLMNode("generate", llmClient, petalflow.LLMNodeConfig{
        Model: "claude-sonnet-4-20250514",
        SystemPrompt: `Answer the user's question using the provided documents.
The documents have been retrieved using multiple search queries to ensure comprehensive coverage.`,
    })

    return petalflow.NewGraphBuilder("multi-query-rag").
        AddNode(queryGenNode).
        Edge(parseNode).
        Edge(mapRetrieveNode).
        Edge(combineNode).
        Edge(generateNode).
        Build()
}
```

## Agentic RAG

Let the LLM decide when to retrieve more information.

```go
func buildAgenticRAG(
    llmClient adapters.LLMClient,
    vectorStore *SimpleVectorStore,
) *petalflow.BasicGraph {
    // Create retrieval tool
    retrieveTool := adapters.NewFuncTool(
        "search_knowledge_base",
        "Search the knowledge base for relevant information",
        func(ctx context.Context, args map[string]any) (map[string]any, error) {
            query := args["query"].(string)
            docs, err := vectorStore.Search(ctx, query, 3)
            if err != nil {
                return nil, err
            }
            return map[string]any{
                "documents": docs,
                "count":     len(docs),
            }, nil
        },
    )

    // Tool node for retrieval
    toolNode := petalflow.NewToolNode("retrieve", retrieveTool, petalflow.ToolNodeConfig{
        InputVar:  "tool_call",
        OutputVar: "retrieved_info",
    })

    // Agent node that can call tools
    agentNode := petalflow.NewLLMNode("agent", llmClient, petalflow.LLMNodeConfig{
        Model: "claude-sonnet-4-20250514",
        SystemPrompt: `You are a helpful assistant with access to a knowledge base.
Use the search_knowledge_base tool when you need information to answer the user's question.
Think step by step about what information you need.`,
        // Tools would be registered here in a full implementation
    })

    // Router to check if agent wants to use tool
    routerNode := petalflow.NewRouterNode("router", petalflow.RouterNodeConfig{
        Strategy: petalflow.RouteStrategyFunction,
        Routes: []petalflow.Route{
            {Name: "retrieve", Target: "retrieve"},
            {Name: "respond", Target: "final"},
        },
        Selector: func(env *petalflow.Envelope) ([]string, error) {
            // Check if agent requested a tool call
            if _, hasToolCall := env.GetVar("tool_call"); hasToolCall {
                return []string{"retrieve"}, nil
            }
            return []string{"respond"}, nil
        },
    })

    // Final response node
    finalNode := petalflow.NewLLMNode("final", llmClient, petalflow.LLMNodeConfig{
        Model:        "claude-sonnet-4-20250514",
        SystemPrompt: "Provide a final, complete answer based on all gathered information.",
    })

    graph := petalflow.NewGraph("agentic-rag")
    graph.AddNode(agentNode)
    graph.AddNode(routerNode)
    graph.AddNode(toolNode)
    graph.AddNode(finalNode)

    graph.AddEdge("agent", "router")
    graph.AddEdge("router", "retrieve")
    graph.AddEdge("router", "final")
    graph.AddEdge("retrieve", "agent") // Loop back for more reasoning
    graph.SetEntry("agent")

    return graph
}
```

## Batch Document Processing

Process many documents through a RAG-style pipeline.

```go
func buildBatchRAGProcessor(
    llmClient adapters.LLMClient,
    embeddingProvider core.EmbeddingProvider,
) *petalflow.BasicGraph {
    // Batch embed documents
    embedNode := petalflow.NewFuncNode("embed_batch", func(ctx context.Context, env *petalflow.Envelope) (*petalflow.Envelope, error) {
        docs, _ := env.GetVar("documents")
        docList := docs.([]string)

        // Create embedding inputs
        var inputs []core.EmbeddingInput
        for i, doc := range docList {
            inputs = append(inputs, core.EmbeddingInput{
                Text: doc,
                ID:   fmt.Sprintf("doc_%d", i),
            })
        }

        resp, err := embeddingProvider.CreateEmbeddings(ctx, &core.EmbeddingRequest{
            Model:     "text-embedding-3-small",
            Input:     inputs,
            InputType: core.InputTypeDocument,
        })
        if err != nil {
            return nil, err
        }

        env.SetVar("embeddings", resp.Vectors)
        return env, nil
    })

    // Process each document with LLM
    processNode := petalflow.NewMapNode("process_docs", petalflow.MapNodeConfig{
        InputVar:    "documents",
        OutputVar:   "processed",
        Concurrency: 4,
        Mapper: func(ctx context.Context, item any, index int) (any, error) {
            doc := item.(string)

            resp, err := llmClient.Complete(ctx, adapters.LLMRequest{
                Model:  "claude-sonnet-4-20250514",
                System: "Extract key facts from this document as bullet points.",
                Messages: []adapters.LLMMessage{
                    {Role: "user", Content: doc},
                },
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

    graph := petalflow.NewGraph("batch-processor")
    graph.AddNode(embedNode)
    graph.AddNode(processNode)
    graph.AddEdge("embed_batch", "process_docs")
    graph.SetEntry("embed_batch")

    return graph
}
```

## Next Steps

- [Multi-Provider Evaluation](./multi-provider.md) - Compare provider outputs for quality assessment
- [Overview](./overview.md) - Return to PetalFlow fundamentals
