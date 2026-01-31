// Package petalflow provides a workflow graph runtime for agents.
//
// PetalFlow is intentionally small: it provides a workflow graph runtime for agents
// that can route work, run tools, call LLMs, and merge results.
//
// This package defines the core types and interfaces for building agent workflows.
package petalflow

import (
	"time"
)

// NodeKind identifies the type of a node.
// The set of kinds is intentionally small to avoid growing a "node zoo".
type NodeKind string

const (
	NodeKindLLM    NodeKind = "llm"
	NodeKindTool   NodeKind = "tool"
	NodeKindRouter NodeKind = "router"
	NodeKindMerge  NodeKind = "merge"
	NodeKindMap    NodeKind = "map"
	NodeKindGate   NodeKind = "gate"
	NodeKindNoop   NodeKind = "noop"
)

// String returns the string representation of the NodeKind.
func (k NodeKind) String() string {
	return string(k)
}

// Message is a chat-style message used for LLM steps and auditing.
type Message struct {
	Role    string         // "system" | "user" | "assistant" | "tool"
	Content string         // plain text; markdown allowed
	Name    string         // optional (tool name, agent role, etc.)
	Meta    map[string]any // optional metadata
}

// Artifact represents a document or derived data produced during a run.
// For large payloads, use URI to reference external storage rather than
// embedding the data directly.
type Artifact struct {
	ID       string         // stable within a run; may be empty if not needed
	Type     string         // e.g. "document", "chunk", "citation", "json", "file"
	MimeType string         // optional: "text/plain", "application/json", etc.
	Text     string         // optional textual content
	Bytes    []byte         // optional binary content (avoid for large data)
	URI      string         // optional pointer to external storage
	Meta     map[string]any // flexible metadata (source, offsets, confidence, etc.)
}

// TraceInfo is propagated by the runtime for observability and replay.
type TraceInfo struct {
	RunID    string    // unique identifier for this run
	ParentID string    // optional: for subgraphs or map/fanout
	SpanID   string    // optional: for node-level tracing
	Started  time.Time // when the run started
}

// NodeError is recorded when nodes fail but the graph continues
// (when ContinueOnError is enabled).
type NodeError struct {
	NodeID  string         // ID of the node that failed
	Kind    NodeKind       // kind of the node
	Message string         // error message
	Attempt int            // which attempt this was (1-indexed)
	At      time.Time      // when the error occurred
	Details map[string]any // additional error context
	Cause   error          // underlying error (may be nil)
}

// Error implements the error interface for NodeError.
func (e NodeError) Error() string {
	return e.Message
}

// Unwrap returns the underlying cause for error unwrapping.
func (e NodeError) Unwrap() error {
	return e.Cause
}

// RouteDecision is produced by RouterNode to indicate which targets to activate.
type RouteDecision struct {
	Targets    []string       // node IDs to route to
	Reason     string         // explanation for the decision
	Confidence *float64       // optional confidence score (0.0-1.0)
	Meta       map[string]any // additional routing metadata
}

// RetryPolicy configures retry behavior for nodes that call external systems.
type RetryPolicy struct {
	MaxAttempts int           // maximum number of attempts (1 = no retries)
	Backoff     time.Duration // base backoff duration between attempts
}

// DefaultRetryPolicy returns a sensible default retry policy.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 3,
		Backoff:     time.Second,
	}
}

// Budget is an optional guardrail for LLM calls to limit resource usage.
type Budget struct {
	MaxInputTokens  int     // maximum input tokens allowed
	MaxOutputTokens int     // maximum output tokens allowed
	MaxTotalTokens  int     // maximum total tokens allowed
	MaxCostUSD      float64 // maximum cost in USD allowed
}

// TokenUsage tracks token consumption for cost tracking and budgeting.
type TokenUsage struct {
	InputTokens  int     // tokens consumed by input/prompt
	OutputTokens int     // tokens generated in output
	TotalTokens  int     // total tokens (input + output)
	CostUSD      float64 // cost in USD (if computable)
}

// Add combines two TokenUsage values.
func (u TokenUsage) Add(other TokenUsage) TokenUsage {
	return TokenUsage{
		InputTokens:  u.InputTokens + other.InputTokens,
		OutputTokens: u.OutputTokens + other.OutputTokens,
		TotalTokens:  u.TotalTokens + other.TotalTokens,
		CostUSD:      u.CostUSD + other.CostUSD,
	}
}

// ErrorPolicy defines how a node handles errors.
type ErrorPolicy string

const (
	// ErrorPolicyFail causes the node to fail and abort the run (default).
	ErrorPolicyFail ErrorPolicy = "fail"
	// ErrorPolicyContinue records the error and continues execution.
	ErrorPolicyContinue ErrorPolicy = "continue"
	// ErrorPolicyRecord records the error in the envelope and continues.
	ErrorPolicyRecord ErrorPolicy = "record"
)
