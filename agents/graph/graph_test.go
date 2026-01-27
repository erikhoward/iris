package graph

import (
	"context"
	"errors"
	"testing"
)

// mockNode is a simple test node.
type mockNode struct {
	id string
}

func (n *mockNode) ID() string { return n.id }
func (n *mockNode) Run(ctx context.Context, state *State) (*State, error) {
	return state, nil
}

func TestNewGraph(t *testing.T) {
	g := NewGraph()

	if g == nil {
		t.Fatal("NewGraph() returned nil")
	}
	if g.Nodes == nil {
		t.Error("Nodes map is nil")
	}
	if g.Edges == nil {
		t.Error("Edges map is nil")
	}
	if g.Entry != "" {
		t.Errorf("Entry = %q, want empty", g.Entry)
	}
}

func TestGraphAddNode(t *testing.T) {
	g := NewGraph()
	node := &mockNode{id: "node1"}

	err := g.AddNode(node)
	if err != nil {
		t.Fatalf("AddNode() error = %v", err)
	}

	if _, exists := g.Nodes["node1"]; !exists {
		t.Error("Node not added to graph")
	}
}

func TestGraphAddNodeNil(t *testing.T) {
	g := NewGraph()

	err := g.AddNode(nil)
	if err == nil {
		t.Error("AddNode(nil) should return error")
	}
}

func TestGraphAddNodeDuplicate(t *testing.T) {
	g := NewGraph()
	node1 := &mockNode{id: "same"}
	node2 := &mockNode{id: "same"}

	_ = g.AddNode(node1)
	err := g.AddNode(node2)

	if !errors.Is(err, ErrNodeExists) {
		t.Errorf("expected ErrNodeExists, got %v", err)
	}
}

func TestGraphAddEdge(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&mockNode{id: "a"})
	_ = g.AddNode(&mockNode{id: "b"})

	err := g.AddEdge("a", "b")
	if err != nil {
		t.Fatalf("AddEdge() error = %v", err)
	}

	if len(g.Edges["a"]) != 1 || g.Edges["a"][0] != "b" {
		t.Error("Edge not added correctly")
	}
}

func TestGraphAddEdgeFromMissing(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&mockNode{id: "b"})

	err := g.AddEdge("a", "b")
	if !errors.Is(err, ErrNodeNotFound) {
		t.Errorf("expected ErrNodeNotFound, got %v", err)
	}
}

func TestGraphAddEdgeToMissing(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&mockNode{id: "a"})

	err := g.AddEdge("a", "b")
	if !errors.Is(err, ErrNodeNotFound) {
		t.Errorf("expected ErrNodeNotFound, got %v", err)
	}
}

func TestGraphSetEntry(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&mockNode{id: "start"})

	err := g.SetEntry("start")
	if err != nil {
		t.Fatalf("SetEntry() error = %v", err)
	}

	if g.Entry != "start" {
		t.Errorf("Entry = %q, want %q", g.Entry, "start")
	}
}

func TestGraphSetEntryMissing(t *testing.T) {
	g := NewGraph()

	err := g.SetEntry("nonexistent")
	if !errors.Is(err, ErrNodeNotFound) {
		t.Errorf("expected ErrNodeNotFound, got %v", err)
	}
}

func TestGraphValidateNoEntry(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&mockNode{id: "node"})

	err := g.Validate()
	if !errors.Is(err, ErrNoEntry) {
		t.Errorf("expected ErrNoEntry, got %v", err)
	}
}

func TestGraphValidateSuccess(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&mockNode{id: "a"})
	_ = g.AddNode(&mockNode{id: "b"})
	_ = g.AddEdge("a", "b")
	_ = g.SetEntry("a")

	err := g.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestGraphValidateSingleNode(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&mockNode{id: "only"})
	_ = g.SetEntry("only")

	err := g.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestGraphValidateOrphanNode(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&mockNode{id: "a"})
	_ = g.AddNode(&mockNode{id: "orphan"})
	_ = g.SetEntry("a")

	err := g.Validate()
	if !errors.Is(err, ErrOrphanNode) {
		t.Errorf("expected ErrOrphanNode, got %v", err)
	}
}

func TestGraphValidateChain(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&mockNode{id: "a"})
	_ = g.AddNode(&mockNode{id: "b"})
	_ = g.AddNode(&mockNode{id: "c"})
	_ = g.AddEdge("a", "b")
	_ = g.AddEdge("b", "c")
	_ = g.SetEntry("a")

	err := g.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestGraphValidateBranching(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&mockNode{id: "a"})
	_ = g.AddNode(&mockNode{id: "b"})
	_ = g.AddNode(&mockNode{id: "c"})
	_ = g.AddEdge("a", "b")
	_ = g.AddEdge("a", "c")
	_ = g.SetEntry("a")

	err := g.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestGraphMultipleEdges(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&mockNode{id: "a"})
	_ = g.AddNode(&mockNode{id: "b"})
	_ = g.AddNode(&mockNode{id: "c"})
	_ = g.AddEdge("a", "b")
	_ = g.AddEdge("a", "c")

	if len(g.Edges["a"]) != 2 {
		t.Errorf("len(Edges[a]) = %d, want 2", len(g.Edges["a"]))
	}
}
