# ADR 0001: PetalFlow Dynamic Routing

## Status

Accepted

## Context

PetalFlow needs to support both static graph execution (where all paths are known at build time) and dynamic routing (where path decisions are made at runtime based on LLM classification or rule evaluation).

The initial runtime implementation used static pre-computation of execution order via DFS traversal. This approach is simple but cannot support RouterNode decisions that determine which successor nodes to visit at runtime.

## Decision

We chose to implement **dynamic successor selection** in the runtime:

1. **Queue-based execution** instead of pre-computed order
2. After each node executes, check if it implements `RouterNode` interface
3. For RouterNodes, use `RouteDecision.Targets` to filter which successors to visit
4. For regular nodes, visit all graph successors
5. Emit `EventRouteDecision` events when routing decisions are made

Key design points:
- RouterNodes store their decision in the envelope via a well-known key pattern (`{nodeID}_decision`)
- The runtime reads this decision after node execution
- This keeps routing logic in the node while the runtime handles traversal

## Consequences

### Positive

- Enables LLM-based classification for intelligent routing
- Supports rule-based branching without code changes
- Maintains separation between routing logic (nodes) and execution (runtime)
- Events provide observability into routing decisions

### Negative

- Slightly more complex runtime code
- Execution order is no longer deterministic for graphs with routers
- Cannot pre-validate that all paths lead to completion

### Trade-offs

- Chose envelope storage over return values for routing decisions to maintain consistent `(*Envelope, error)` signature for all nodes
- Used interface detection (`node.(RouterNode)`) rather than a `Kind()` check to support custom router implementations

## References

- `agents/petalflow/runtime.go` - Dynamic execution implementation
- `agents/petalflow/router_node.go` - RouterNode interface and implementations
