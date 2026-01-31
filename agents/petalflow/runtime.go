package petalflow

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"
)

// Runtime errors
var (
	ErrMaxHopsExceeded = errors.New("maximum hops exceeded")
	ErrRunCanceled     = errors.New("run was canceled")
	ErrNodeExecution   = errors.New("node execution failed")
)

// Runtime executes graphs and emits events.
type Runtime interface {
	// Run executes the graph with the given initial envelope.
	Run(ctx context.Context, g Graph, env *Envelope, opts RunOptions) (*Envelope, error)

	// Events returns a channel for receiving runtime events.
	// The channel is closed when the run completes.
	Events() <-chan Event
}

// RunOptions controls execution behavior.
type RunOptions struct {
	// MaxHops protects against infinite cycles (default: 100).
	MaxHops int

	// ContinueOnError records errors and continues when possible.
	ContinueOnError bool

	// Concurrency sets the worker pool size for parallel execution (default: 1).
	Concurrency int

	// Now provides the current time (for testing). If nil, uses time.Now.
	Now func() time.Time

	// EventHandler receives events during execution.
	EventHandler EventHandler
}

// DefaultRunOptions returns sensible default options.
func DefaultRunOptions() RunOptions {
	return RunOptions{
		MaxHops:         100,
		ContinueOnError: false,
		Concurrency:     1,
	}
}

// BasicRuntime is a simple sequential runtime implementation.
type BasicRuntime struct {
	eventCh chan Event
}

// NewRuntime creates a new runtime instance.
func NewRuntime() *BasicRuntime {
	return &BasicRuntime{
		eventCh: make(chan Event, 100), // buffered channel
	}
}

// Events returns the event channel.
func (r *BasicRuntime) Events() <-chan Event {
	return r.eventCh
}

// Run executes the graph sequentially, following edges from the entry node.
func (r *BasicRuntime) Run(ctx context.Context, g Graph, env *Envelope, opts RunOptions) (*Envelope, error) {
	// Apply defaults
	if opts.MaxHops <= 0 {
		opts.MaxHops = 100
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}

	// Validate graph
	if err := validateGraph(g); err != nil {
		return nil, err
	}

	// Initialize envelope if nil
	if env == nil {
		env = NewEnvelope()
	}

	// Generate run ID
	runID := generateRunID()
	env.Trace.RunID = runID
	env.Trace.Started = opts.Now()

	// Create event emitter
	emit := func(e Event) {
		if opts.EventHandler != nil {
			opts.EventHandler(e)
		}
		select {
		case r.eventCh <- e:
		default:
			// Drop if channel is full
		}
	}

	// Emit run started
	runStart := opts.Now()
	emit(NewEvent(EventRunStarted, runID).
		WithPayload("graph", g.Name()).
		WithPayload("entry", g.Entry()))

	// Execute graph
	result, err := r.executeGraph(ctx, g, env, opts, emit, runStart)

	// Emit run finished
	runElapsed := opts.Now().Sub(runStart)
	finishEvent := NewEvent(EventRunFinished, runID).
		WithElapsed(runElapsed)

	if err != nil {
		finishEvent = finishEvent.
			WithPayload("status", "failed").
			WithPayload("error", err.Error())
	} else {
		finishEvent = finishEvent.
			WithPayload("status", "completed")
	}
	emit(finishEvent)

	return result, err
}

// executeGraph performs the actual graph execution.
// It uses dynamic successor selection to support RouterNode decisions.
func (r *BasicRuntime) executeGraph(
	ctx context.Context,
	g Graph,
	env *Envelope,
	opts RunOptions,
	emit EventEmitter,
	runStart time.Time,
) (*Envelope, error) {
	hopCount := make(map[string]int)
	current := env

	// Use a queue for dynamic execution order
	// Start with the entry node
	queue := []string{g.Entry()}
	visited := make(map[string]bool)

	for len(queue) > 0 {
		// Pop next node from queue
		nodeID := queue[0]
		queue = queue[1:]

		// Skip if already visited in this execution path
		// (this prevents infinite loops in non-router cases)
		if visited[nodeID] && hopCount[nodeID] > 0 {
			// For cycles, check max hops
			if hopCount[nodeID] >= opts.MaxHops {
				continue
			}
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return current, fmt.Errorf("%w: %v", ErrRunCanceled, ctx.Err())
		default:
		}

		// Check max hops
		hopCount[nodeID]++
		if hopCount[nodeID] > opts.MaxHops {
			return current, fmt.Errorf("%w: node %s executed %d times", ErrMaxHopsExceeded, nodeID, hopCount[nodeID])
		}

		// Get node
		node, ok := g.NodeByID(nodeID)
		if !ok {
			return current, fmt.Errorf("%w: %s", ErrNodeNotFound, nodeID)
		}

		// Execute node
		result, err := r.executeNode(ctx, node, current, opts, emit, runStart)
		if err != nil {
			if opts.ContinueOnError {
				current.AppendError(NodeError{
					NodeID:  nodeID,
					Kind:    node.Kind(),
					Message: err.Error(),
					Attempt: hopCount[nodeID],
					At:      opts.Now(),
					Cause:   err,
				})
			} else {
				return current, fmt.Errorf("%w: node %s: %v", ErrNodeExecution, nodeID, err)
			}
		} else {
			current = result
		}

		// Mark as visited
		visited[nodeID] = true

		// Determine next nodes to execute
		nextNodes := r.determineSuccessors(g, node, current, emit, runStart, opts)
		queue = append(queue, nextNodes...)
	}

	return current, nil
}

// determineSuccessors decides which nodes to execute next after the current node.
// For RouterNodes, it uses the RouteDecision; for others, it uses all graph successors.
func (r *BasicRuntime) determineSuccessors(
	g Graph,
	node Node,
	env *Envelope,
	emit EventEmitter,
	runStart time.Time,
	opts RunOptions,
) []string {
	nodeID := node.ID()
	graphSuccessors := g.Successors(nodeID)

	// If no successors in graph, nothing to execute next
	if len(graphSuccessors) == 0 {
		return nil
	}

	// Check if this is a RouterNode
	router, isRouter := node.(RouterNode)
	if !isRouter {
		// Not a router, use all graph successors
		return graphSuccessors
	}

	// Get the routing decision from the envelope
	// RouterNodes store their decision in the envelope during Run()
	decisionKey := nodeID + "_decision"
	decisionVal, ok := env.GetVar(decisionKey)
	if !ok {
		// No decision stored, fall back to all successors
		return graphSuccessors
	}

	decision, ok := decisionVal.(RouteDecision)
	if !ok {
		// Invalid decision type, fall back to all successors
		return graphSuccessors
	}

	// Emit routing decision event
	emit(NewEvent(EventRouteDecision, env.Trace.RunID).
		WithNode(nodeID, router.Kind()).
		WithElapsed(opts.Now().Sub(runStart)).
		WithPayload("targets", decision.Targets).
		WithPayload("reason", decision.Reason).
		WithPayload("confidence", decision.Confidence))

	// Filter successors to only those in the decision targets
	// The decision targets are node IDs that should be executed
	var filteredSuccessors []string
	targetSet := make(map[string]bool)
	for _, t := range decision.Targets {
		targetSet[t] = true
	}

	for _, succ := range graphSuccessors {
		if targetSet[succ] {
			filteredSuccessors = append(filteredSuccessors, succ)
		}
	}

	return filteredSuccessors
}

// executeNode executes a single node with event emission.
func (r *BasicRuntime) executeNode(
	ctx context.Context,
	node Node,
	env *Envelope,
	opts RunOptions,
	emit EventEmitter,
	runStart time.Time,
) (*Envelope, error) {
	nodeID := node.ID()
	nodeKind := node.Kind()
	runID := env.Trace.RunID

	// Emit node started
	nodeStart := opts.Now()
	emit(NewEvent(EventNodeStarted, runID).
		WithNode(nodeID, nodeKind).
		WithElapsed(nodeStart.Sub(runStart)))

	// Execute node
	result, err := node.Run(ctx, env)

	// Calculate elapsed time
	nodeElapsed := opts.Now().Sub(nodeStart)

	if err != nil {
		// Emit node failed
		emit(NewEvent(EventNodeFailed, runID).
			WithNode(nodeID, nodeKind).
			WithElapsed(nodeElapsed).
			WithPayload("error", err.Error()))
		return nil, err
	}

	// Emit node finished
	emit(NewEvent(EventNodeFinished, runID).
		WithNode(nodeID, nodeKind).
		WithElapsed(nodeElapsed))

	return result, nil
}

// validateGraph performs basic validation.
func validateGraph(g Graph) error {
	if g == nil {
		return errors.New("graph is nil")
	}

	nodes := g.Nodes()
	if len(nodes) == 0 {
		return ErrEmptyGraph
	}

	entry := g.Entry()
	if entry == "" {
		return ErrNoEntryNode
	}

	if _, ok := g.NodeByID(entry); !ok {
		return fmt.Errorf("%w: entry node %q", ErrNodeNotFound, entry)
	}

	return nil
}

// generateRunID creates a unique run identifier.
// Uses crypto/rand for secure random generation.
func generateRunID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("run-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Ensure interface compliance at compile time.
var _ Runtime = (*BasicRuntime)(nil)
