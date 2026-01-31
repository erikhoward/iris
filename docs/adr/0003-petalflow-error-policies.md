# ADR 0003: PetalFlow Error Handling Policies

## Status

Accepted

## Context

Workflow nodes (LLMNode, ToolNode) can fail due to:
- Transient errors (network timeouts, rate limits)
- Permanent errors (invalid input, missing resources)
- Budget overruns (token limits exceeded)

Different workflows need different error handling strategies:
- Critical paths should fail fast
- Notification workflows should continue despite failures
- Debugging workflows need error details preserved

## Decision

We implemented a **three-tier error handling strategy**:

### 1. Retry Policy (Per-Node)

Each node has a `RetryPolicy` with:
- `MaxAttempts`: Number of retry attempts (default: 3)
- `Backoff`: Base delay between retries (default: 1s)

Retries use linear backoff: `Backoff * attempt_number`

### 2. Error Policy (Per-Node)

`OnError` determines behavior after all retries are exhausted:
- `ErrorPolicyFail`: Return error immediately (default)
- `ErrorPolicyContinue`: Record error and continue with nil output
- `ErrorPolicyRecord`: Same as continue (alias for clarity)

### 3. ContinueOnError (Runtime-Level)

`RunOptions.ContinueOnError` applies to all nodes:
- `false`: First node error stops the workflow (default)
- `true`: Record error in envelope, continue to next node

### Error Recording

When errors are recorded (not failed), they're stored as:
- `NodeError` in `Envelope.Errors` list
- `{outputKey}_error` variable in envelope
- Output set to `nil` to indicate failure

## Consequences

### Positive

- Flexible error handling at multiple levels
- Automatic retry handles transient failures
- Error recording enables post-workflow analysis
- Continue policy supports "best effort" workflows

### Negative

- Multiple error handling levels add complexity
- Need to check for nil outputs after continue policy
- Error details scattered between Errors list and variables

### Trade-offs

- Chose linear backoff over exponential for simplicity
- Kept retry logic in node rather than runtime for node-specific policies
- Used enum for error policy rather than callback for simplicity

## Examples

```go
// Fail fast on error (default)
node := NewToolNode("critical-tool", tool, ToolNodeConfig{
    OnError: ErrorPolicyFail,
})

// Continue on error, check later
node := NewToolNode("optional-tool", tool, ToolNodeConfig{
    OnError: ErrorPolicyContinue,
})
// After execution, check: if output == nil { handle error }

// Workflow-level continue
opts := RunOptions{
    ContinueOnError: true,
}
// After execution, check: if env.HasErrors() { handle errors }
```

## References

- `agents/petalflow/types.go` - ErrorPolicy, RetryPolicy, NodeError
- `agents/petalflow/tool_node.go` - handleError implementation
- `agents/petalflow/runtime.go` - ContinueOnError handling
