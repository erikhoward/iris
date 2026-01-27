package graph

import (
	"errors"
	"fmt"
)

// Graph validation errors.
var (
	ErrNodeExists       = errors.New("node already exists")
	ErrNodeNotFound     = errors.New("node not found")
	ErrNoEntry          = errors.New("entry node not set")
	ErrEntryNotFound    = errors.New("entry node not found")
	ErrOrphanNode       = errors.New("orphan node detected")
	ErrEdgeTargetNotFound = errors.New("edge target not found")
)

// Graph represents a directed acyclic graph of nodes.
type Graph struct {
	Nodes map[string]Node
	Edges map[string][]string // node ID -> successor node IDs
	Entry string              // entry node ID
}

// NewGraph creates a new empty graph.
func NewGraph() *Graph {
	return &Graph{
		Nodes: make(map[string]Node),
		Edges: make(map[string][]string),
	}
}

// AddNode adds a node to the graph.
// Returns ErrNodeExists if a node with the same ID already exists.
func (g *Graph) AddNode(n Node) error {
	if n == nil {
		return errors.New("node cannot be nil")
	}

	id := n.ID()
	if _, exists := g.Nodes[id]; exists {
		return fmt.Errorf("%w: %s", ErrNodeExists, id)
	}

	g.Nodes[id] = n
	return nil
}

// AddEdge adds a directed edge from one node to another.
// Both nodes must exist in the graph.
func (g *Graph) AddEdge(from, to string) error {
	if _, exists := g.Nodes[from]; !exists {
		return fmt.Errorf("%w: %s", ErrNodeNotFound, from)
	}
	if _, exists := g.Nodes[to]; !exists {
		return fmt.Errorf("%w: %s", ErrNodeNotFound, to)
	}

	g.Edges[from] = append(g.Edges[from], to)
	return nil
}

// SetEntry sets the entry node for graph execution.
// The node must exist in the graph.
func (g *Graph) SetEntry(nodeID string) error {
	if _, exists := g.Nodes[nodeID]; !exists {
		return fmt.Errorf("%w: %s", ErrNodeNotFound, nodeID)
	}

	g.Entry = nodeID
	return nil
}

// Validate checks the graph for correctness.
// Returns an error if:
// - Entry node is not set
// - Entry node does not exist
// - Edge targets do not exist
// - Orphan nodes exist (not reachable from entry)
func (g *Graph) Validate() error {
	// Check entry node
	if g.Entry == "" {
		return ErrNoEntry
	}
	if _, exists := g.Nodes[g.Entry]; !exists {
		return fmt.Errorf("%w: %s", ErrEntryNotFound, g.Entry)
	}

	// Check all edge targets exist
	for from, targets := range g.Edges {
		for _, to := range targets {
			if _, exists := g.Nodes[to]; !exists {
				return fmt.Errorf("%w: edge %s -> %s", ErrEdgeTargetNotFound, from, to)
			}
		}
	}

	// Check for orphan nodes (not reachable from entry)
	reachable := g.findReachable()
	for id := range g.Nodes {
		if !reachable[id] {
			return fmt.Errorf("%w: %s", ErrOrphanNode, id)
		}
	}

	return nil
}

// findReachable returns a set of node IDs reachable from the entry node.
func (g *Graph) findReachable() map[string]bool {
	reachable := make(map[string]bool)
	if g.Entry == "" {
		return reachable
	}

	// BFS from entry
	queue := []string{g.Entry}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if reachable[current] {
			continue
		}
		reachable[current] = true

		for _, next := range g.Edges[current] {
			if !reachable[next] {
				queue = append(queue, next)
			}
		}
	}

	return reachable
}
