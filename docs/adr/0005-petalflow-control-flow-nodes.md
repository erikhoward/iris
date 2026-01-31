# ADR 0005: PetalFlow Control Flow Nodes (MapNode & GateNode)

## Status

Accepted

## Context

Workflows need expressive control flow primitives beyond basic linear execution:
- Iterating over collections (map/filter patterns)
- Conditional guards (validation, authorization, feature flags)
- Processing batches of items with optional parallelism

The existing RouterNode handles branching based on input analysis, but doesn't cover iteration or simple pass/fail guards.

## Decision

We implemented two new node types for control flow: **MapNode** and **GateNode**.

### 1. MapNode - Collection Iteration

`MapNode` applies a transformation to each item in a collection.

```go
type MapNodeConfig struct {
    InputVar        string  // Variable containing collection to iterate
    OutputVar       string  // Where to store results (default: "{id}_output")
    ItemVar         string  // Variable name for current item (default: "item")
    IndexVar        string  // Optional variable for item index
    Concurrency     int     // Workers for parallel processing (default: 1)
    Mapper          func(ctx, item, index) (any, error)  // Function mapper
    MapperNode      Node    // Node-based mapper (mutually exclusive)
    ContinueOnError bool    // Skip failed items instead of stopping
    PreserveOrder   bool    // Maintain input order in output (default: true)
}
```

**Supports two mapping strategies:**

| Strategy | Use Case |
|----------|----------|
| Function-based | Simple transformations, calculations |
| Node-based | Complex processing (e.g., LLM call per item) |

**Example usage:**

```go
// Double each number with 4 parallel workers
node := NewMapNode("doubler", MapNodeConfig{
    InputVar:    "numbers",
    Concurrency: 4,
    Mapper: func(ctx context.Context, item any, index int) (any, error) {
        return item.(int) * 2, nil
    },
})
```

### 2. GateNode - Conditional Guards

`GateNode` evaluates a condition and controls execution flow.

```go
type GateNodeConfig struct {
    Condition      func(ctx, env) (bool, error)  // Function condition
    ConditionVar   string        // Variable name to check (alternative)
    OnFail         GateAction    // Action when condition fails
    FailMessage    string        // Error message for GateActionBlock
    RedirectNodeID string        // Target node for GateActionRedirect
    ResultVar      string        // Store pass/fail result
}
```

**Gate Actions:**

| Action | Behavior |
|--------|----------|
| `GateActionBlock` | Stop execution with error (default) |
| `GateActionSkip` | Pass through without modification |
| `GateActionRedirect` | Route to different node |

**Example usage:**

```go
// Block unauthorized requests
gate := NewGateNode("auth-check", GateNodeConfig{
    Condition: func(ctx context.Context, env *Envelope) (bool, error) {
        user, _ := env.GetVar("user")
        return user != nil, nil
    },
    OnFail:      GateActionBlock,
    FailMessage: "authentication required",
})

// Redirect failures to error handler
gate := NewGateNode("validator", GateNodeConfig{
    ConditionVar:   "is_valid",
    OnFail:         GateActionRedirect,
    RedirectNodeID: "error_handler",
})
```

### 3. Runtime Integration

The runtime's `determineSuccessors` function was extended to handle GateNode redirects:

```go
// Check for GateNode redirect
if redirectVal, ok := env.GetVar("__gate_redirect__"); ok {
    redirectNode := redirectVal.(string)
    // Verify target is valid successor, emit event, route
}
```

### 4. Truthiness Evaluation

GateNode uses JavaScript-like truthiness for `ConditionVar`:

| Type | Falsy Values |
|------|--------------|
| `nil` | always false |
| `bool` | `false` |
| `int/float64` | `0` |
| `string` | `""` |
| `[]any` | empty slice |
| `map[string]any` | empty map |

## Consequences

### Positive

- Natural expression of iteration patterns without custom loops
- Simple guard clauses for validation and authorization
- MapNode concurrency enables efficient batch processing
- GateNode redirect integrates cleanly with existing routing
- Both nodes preserve envelope immutability (clone before modify)

### Negative

- MapNode with node-based mapping has higher overhead (envelope cloning per item)
- GateNode redirect relies on magic variable (`__gate_redirect__`)
- `PreserveOrder` defaults to true but can't distinguish "not set" from "set to false"

### Trade-offs

- Function mappers are simpler but can't use other PetalFlow nodes
- Node-based mappers enable complex workflows but add overhead
- Gate's truthiness evaluation matches JavaScript semantics (familiar but implicit)
- Redirect via envelope variable keeps nodes stateless but requires runtime support

## Examples

```go
// Batch processing with error tolerance
graph, _ := NewGraphBuilder("batch-processor").
    AddNode(loadItems).
    Edge(NewMapNode("process", MapNodeConfig{
        InputVar:        "items",
        Concurrency:     8,
        ContinueOnError: true,
        Mapper:          processItem,
    })).
    Edge(saveResults).
    Build()

// Guarded workflow
graph, _ := NewGraphBuilder("guarded-flow").
    AddNode(validateInput).
    Edge(NewGateNode("auth-gate", GateNodeConfig{
        ConditionVar:   "is_authenticated",
        OnFail:         GateActionRedirect,
        RedirectNodeID: "login-redirect",
    })).
    FanOut(
        processAuthorized,
        redirectToLogin,
    ).
    Build()
```

## References

- `agents/petalflow/map_node.go` - MapNode implementation
- `agents/petalflow/gate_node.go` - GateNode implementation
- `agents/petalflow/runtime.go` - determineSuccessors with redirect handling
- `agents/petalflow/map_node_test.go` - MapNode tests
- `agents/petalflow/gate_node_test.go` - GateNode tests
