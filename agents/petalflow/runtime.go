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
func (r *BasicRuntime) executeGraph(
	ctx context.Context,
	g Graph,
	env *Envelope,
	opts RunOptions,
	emit EventEmitter,
	runStart time.Time,
) (*Envelope, error) {
	// Get execution order
	// For sequential execution, we follow edges from entry
	// In future, this will use topological sort for parallel execution
	order, err := computeExecutionOrder(g)
	if err != nil {
		return nil, err
	}

	hopCount := make(map[string]int)
	current := env

	for _, nodeID := range order {
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
	}

	return current, nil
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

// computeExecutionOrder determines the order of node execution.
// For now, this performs a simple traversal from the entry node.
// Future versions will support parallel execution with dependency tracking.
func computeExecutionOrder(g Graph) ([]string, error) {
	entry := g.Entry()
	if entry == "" {
		return nil, ErrNoEntryNode
	}

	// Simple DFS traversal for sequential execution
	visited := make(map[string]bool)
	order := make([]string, 0)

	var visit func(id string)
	visit = func(id string) {
		if visited[id] {
			return
		}
		visited[id] = true
		order = append(order, id)

		// Visit successors
		for _, succ := range g.Successors(id) {
			visit(succ)
		}
	}

	visit(entry)
	return order, nil
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
