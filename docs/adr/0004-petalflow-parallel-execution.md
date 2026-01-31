# ADR 0004: PetalFlow Parallel Execution & Merge

## Status

Accepted

## Context

Workflows often need to execute multiple branches concurrently:
- Fan-out patterns (one node spawns multiple parallel branches)
- Competitive execution (run multiple models, pick best result)
- Independent tasks that don't need serialization

After parallel branches complete, results must be merged back into a single envelope for downstream processing.

## Decision

We implemented **concurrent execution with configurable merge strategies**.

### 1. Concurrency Control (Runtime Level)

`RunOptions.Concurrency` controls worker pool size:
- `1` (default): Sequential execution (original behavior)
- `>1`: Parallel execution with worker pool

```go
opts := RunOptions{
    Concurrency: 4, // Use 4 workers
}
```

### 2. Envelope Cloning

When executing parallel branches, envelopes are cloned to prevent race conditions:
- Each branch receives a `Clone()` of the parent envelope
- Modifications in one branch don't affect others
- Merge node collects all branch envelopes

### 3. MergeNode & Strategies

`MergeNode` combines multiple envelopes using pluggable strategies:

```go
type MergeStrategy interface {
    Name() string
    Merge(ctx context.Context, inputs []*Envelope) (*Envelope, error)
}
```

**Built-in strategies:**

| Strategy | Purpose | Use Case |
|----------|---------|----------|
| `JSONMergeStrategy` | Merge Vars maps | Combine independent data |
| `ConcatMergeStrategy` | Concatenate strings | Collect text outputs |
| `BestScoreMergeStrategy` | Pick highest/lowest score | Competitive evaluation |
| `FuncMergeStrategy` | Custom function | Domain-specific logic |
| `AllMergeStrategy` | Collect all + apply inner | Preserve all inputs |

### 4. GraphBuilder FanOut

Fluent API for constructing parallel workflows:

```go
graph, _ := NewGraphBuilder("workflow").
    AddNode(start).
    FanOut(branchA, branchB, branchC).
    Merge(NewMergeNode("combiner", MergeNodeConfig{
        Strategy: NewBestScoreMergeStrategy(BestScoreMergeConfig{
            ScoreVar:       "confidence",
            HigherIsBetter: true,
        }),
    })).
    Edge(finalNode).
    Build()
```

### 5. Merge Coordination

The runtime tracks pending inputs for merge nodes:
- Counts predecessors that must complete
- Collects envelope from each completed branch
- Executes merge only when all inputs arrive
- Propagates merged envelope to successors

## Consequences

### Positive

- Natural expression of parallel patterns
- Worker pool limits resource usage
- Pluggable merge strategies for flexibility
- Envelope cloning prevents race conditions
- Builder API makes parallel graphs readable

### Negative

- Parallel execution adds complexity to debugging
- Envelope cloning has memory overhead
- Merge coordination requires careful synchronization
- Non-deterministic execution order in parallel branches

### Trade-offs

- Chose worker pool over unbounded goroutines for predictability
- Shallow clone by default (deep values still shared)
- Merge waits for ALL inputs (no early exit on error by default)
- Sequential mode preserved for simpler use cases

## Examples

```go
// Diamond pattern with merge
graph, _ := NewGraphBuilder("diamond").
    AddNode(NewNoopNode("start")).
    FanOut(
        NewNoopNode("left"),
        NewNoopNode("right"),
    ).
    Merge(NewMergeNode("join", MergeNodeConfig{})).
    Build()

// Competitive LLM evaluation
graph, _ := NewGraphBuilder("eval").
    AddNode(inputNode).
    FanOutBranches(
        NewBranch(NewLLMNode("gpt4", gpt4Client, LLMNodeConfig{})),
        NewBranch(NewLLMNode("claude", claudeClient, LLMNodeConfig{})),
    ).
    Merge(NewMergeNode("best", MergeNodeConfig{
        Strategy: NewBestScoreMergeStrategy(BestScoreMergeConfig{
            ScoreVar:       "score",
            HigherIsBetter: true,
        }),
    })).
    Build()
```

## References

- `agents/petalflow/merge_node.go` - MergeNode and strategies
- `agents/petalflow/builder.go` - GraphBuilder with FanOut
- `agents/petalflow/runtime.go` - executeGraphParallel
- `agents/petalflow/envelope.go` - Clone() method
