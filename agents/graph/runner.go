package graph

import (
	"context"
	"errors"
)

// Runner executes a graph.
type Runner struct {
	graph *Graph
}

// NewRunner creates a new runner for the given graph.
// Returns an error if the graph is invalid.
func NewRunner(g *Graph) (*Runner, error) {
	if g == nil {
		return nil, errors.New("graph cannot be nil")
	}

	if err := g.Validate(); err != nil {
		return nil, err
	}

	return &Runner{graph: g}, nil
}

// Execute runs the graph starting from the entry node.
// Returns the final state after all nodes have executed, or an error if any node fails.
//
// Execution behavior:
//  1. Start at entry node
//  2. Run node with current state
//  3. Get successor nodes from edges
//  4. If multiple successors, execute first (v0.1 - no branching logic)
//  5. If no successors, return final state
//  6. Propagate context cancellation
//  7. Return error if any node fails
func (r *Runner) Execute(ctx context.Context, input *State) (*State, error) {
	if input == nil {
		input = NewState()
	}

	// Handle empty graph (no nodes)
	if len(r.graph.Nodes) == 0 {
		return input, nil
	}

	state := input
	currentID := r.graph.Entry

	for currentID != "" {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Get current node
		node, exists := r.graph.Nodes[currentID]
		if !exists {
			// Should not happen if graph was validated
			return nil, ErrNodeNotFound
		}

		// Execute node
		newState, err := node.Run(ctx, state)
		if err != nil {
			return nil, err
		}
		state = newState

		// Get next node
		successors := r.graph.Edges[currentID]
		if len(successors) == 0 {
			// No more nodes, we're done
			break
		}

		// v0.1: Take first successor (no conditional routing)
		currentID = successors[0]
	}

	return state, nil
}
