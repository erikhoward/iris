package graph

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewRunnerNilGraph(t *testing.T) {
	_, err := NewRunner(nil)
	if err == nil {
		t.Error("NewRunner(nil) should return error")
	}
}

func TestNewRunnerInvalidGraph(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&mockNode{id: "node"})
	// No entry set

	_, err := NewRunner(g)
	if err == nil {
		t.Error("NewRunner with invalid graph should return error")
	}
}

func TestRunnerExecuteSingleNode(t *testing.T) {
	g := NewGraph()
	node := NewFuncNode("single", func(ctx context.Context, state *State) (*State, error) {
		state.Set("executed", true)
		return state, nil
	})
	_ = g.AddNode(node)
	_ = g.SetEntry("single")

	runner, err := NewRunner(g)
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	input := NewState()
	output, err := runner.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if v, _ := output.Get("executed"); v != true {
		t.Error("Node did not execute")
	}
}

func TestRunnerExecuteSequential(t *testing.T) {
	g := NewGraph()

	node1 := NewFuncNode("first", func(ctx context.Context, state *State) (*State, error) {
		state.Set("order", "1")
		return state, nil
	})
	node2 := NewFuncNode("second", func(ctx context.Context, state *State) (*State, error) {
		v, _ := state.Get("order")
		state.Set("order", v.(string)+"2")
		return state, nil
	})
	node3 := NewFuncNode("third", func(ctx context.Context, state *State) (*State, error) {
		v, _ := state.Get("order")
		state.Set("order", v.(string)+"3")
		return state, nil
	})

	_ = g.AddNode(node1)
	_ = g.AddNode(node2)
	_ = g.AddNode(node3)
	_ = g.AddEdge("first", "second")
	_ = g.AddEdge("second", "third")
	_ = g.SetEntry("first")

	runner, _ := NewRunner(g)
	output, err := runner.Execute(context.Background(), NewState())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if v, _ := output.Get("order"); v != "123" {
		t.Errorf("order = %v, want 123", v)
	}
}

func TestRunnerExecuteStatePropagation(t *testing.T) {
	g := NewGraph()

	node1 := NewFuncNode("setter", func(ctx context.Context, state *State) (*State, error) {
		state.Set("value", 42)
		return state, nil
	})
	node2 := NewFuncNode("reader", func(ctx context.Context, state *State) (*State, error) {
		v, ok := state.Get("value")
		if !ok {
			return nil, errors.New("value not found")
		}
		state.Set("doubled", v.(int)*2)
		return state, nil
	})

	_ = g.AddNode(node1)
	_ = g.AddNode(node2)
	_ = g.AddEdge("setter", "reader")
	_ = g.SetEntry("setter")

	runner, _ := NewRunner(g)
	output, err := runner.Execute(context.Background(), NewState())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if v, _ := output.Get("doubled"); v != 84 {
		t.Errorf("doubled = %v, want 84", v)
	}
}

func TestRunnerExecuteNodeError(t *testing.T) {
	g := NewGraph()

	expectedErr := errors.New("node failed")
	node1 := NewFuncNode("failing", func(ctx context.Context, state *State) (*State, error) {
		return nil, expectedErr
	})
	node2 := NewFuncNode("never", func(ctx context.Context, state *State) (*State, error) {
		t.Error("Second node should not execute after error")
		return state, nil
	})

	_ = g.AddNode(node1)
	_ = g.AddNode(node2)
	_ = g.AddEdge("failing", "never")
	_ = g.SetEntry("failing")

	runner, _ := NewRunner(g)
	_, err := runner.Execute(context.Background(), NewState())
	if err != expectedErr {
		t.Errorf("Execute() error = %v, want %v", err, expectedErr)
	}
}

func TestRunnerExecuteContextCancellation(t *testing.T) {
	g := NewGraph()

	started := make(chan struct{})
	node := NewFuncNode("blocking", func(ctx context.Context, state *State) (*State, error) {
		close(started)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
			return state, nil
		}
	})

	_ = g.AddNode(node)
	_ = g.SetEntry("blocking")

	runner, _ := NewRunner(g)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error)
	go func() {
		_, err := runner.Execute(ctx, NewState())
		done <- err
	}()

	// Wait for node to start, then cancel
	<-started
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Execute() error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Error("Execute() did not return after context cancellation")
	}
}

func TestRunnerExecuteNilInput(t *testing.T) {
	g := NewGraph()
	node := NewFuncNode("node", func(ctx context.Context, state *State) (*State, error) {
		if state == nil {
			return nil, errors.New("state is nil")
		}
		state.Set("ok", true)
		return state, nil
	})
	_ = g.AddNode(node)
	_ = g.SetEntry("node")

	runner, _ := NewRunner(g)
	output, err := runner.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if v, _ := output.Get("ok"); v != true {
		t.Error("Node did not execute with default state")
	}
}

func TestRunnerExecuteEmptyGraph(t *testing.T) {
	// This is a tricky case - an empty graph with no nodes
	// can't have a valid entry, so NewRunner should fail
	g := NewGraph()

	_, err := NewRunner(g)
	if err == nil {
		t.Error("NewRunner with empty graph should fail validation")
	}
}

func TestRunnerExecuteInputPreserved(t *testing.T) {
	g := NewGraph()
	node := NewFuncNode("node", func(ctx context.Context, state *State) (*State, error) {
		state.Set("new", "value")
		return state, nil
	})
	_ = g.AddNode(node)
	_ = g.SetEntry("node")

	runner, _ := NewRunner(g)

	input := NewState()
	input.Set("existing", "data")

	output, err := runner.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if v, _ := output.Get("existing"); v != "data" {
		t.Error("Input state not preserved")
	}
	if v, _ := output.Get("new"); v != "value" {
		t.Error("New state not added")
	}
}

func TestRunnerExecuteBranchingTakesFirst(t *testing.T) {
	g := NewGraph()

	entry := NewFuncNode("entry", func(ctx context.Context, state *State) (*State, error) {
		return state, nil
	})
	branch1 := NewFuncNode("branch1", func(ctx context.Context, state *State) (*State, error) {
		state.Set("branch", "first")
		return state, nil
	})
	branch2 := NewFuncNode("branch2", func(ctx context.Context, state *State) (*State, error) {
		state.Set("branch", "second")
		return state, nil
	})

	_ = g.AddNode(entry)
	_ = g.AddNode(branch1)
	_ = g.AddNode(branch2)
	_ = g.AddEdge("entry", "branch1") // First edge
	_ = g.AddEdge("entry", "branch2") // Second edge
	_ = g.SetEntry("entry")

	runner, _ := NewRunner(g)
	output, err := runner.Execute(context.Background(), NewState())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Should take first branch
	if v, _ := output.Get("branch"); v != "first" {
		t.Errorf("branch = %v, want first", v)
	}
}

func TestFuncNode(t *testing.T) {
	called := false
	node := NewFuncNode("test", func(ctx context.Context, state *State) (*State, error) {
		called = true
		return state, nil
	})

	if node.ID() != "test" {
		t.Errorf("ID() = %q, want test", node.ID())
	}

	_, _ = node.Run(context.Background(), NewState())
	if !called {
		t.Error("Function was not called")
	}
}
