package graph

import "context"

// Node represents a processing unit in a graph.
// Nodes receive state, perform operations, and return modified state.
type Node interface {
	// ID returns the unique identifier for this node.
	ID() string

	// Run executes the node's logic with the given state.
	// Returns the modified state or an error to halt execution.
	Run(ctx context.Context, state *State) (*State, error)
}

// FuncNode is a simple Node implementation backed by a function.
type FuncNode struct {
	id string
	fn func(ctx context.Context, state *State) (*State, error)
}

// NewFuncNode creates a node from a function.
func NewFuncNode(id string, fn func(ctx context.Context, state *State) (*State, error)) *FuncNode {
	return &FuncNode{id: id, fn: fn}
}

// ID returns the node identifier.
func (n *FuncNode) ID() string {
	return n.id
}

// Run executes the node's function.
func (n *FuncNode) Run(ctx context.Context, state *State) (*State, error) {
	return n.fn(ctx, state)
}
