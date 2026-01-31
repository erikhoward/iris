package petalflow

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewRuntime(t *testing.T) {
	rt := NewRuntime()

	if rt == nil {
		t.Fatal("NewRuntime() returned nil")
	}

	// Events channel should be available
	ch := rt.Events()
	if ch == nil {
		t.Error("Events() returned nil channel")
	}
}

func TestDefaultRunOptions(t *testing.T) {
	opts := DefaultRunOptions()

	if opts.MaxHops != 100 {
		t.Errorf("DefaultRunOptions().MaxHops = %v, want 100", opts.MaxHops)
	}
	if opts.ContinueOnError != false {
		t.Error("DefaultRunOptions().ContinueOnError should be false")
	}
	if opts.Concurrency != 1 {
		t.Errorf("DefaultRunOptions().Concurrency = %v, want 1", opts.Concurrency)
	}
}

func TestRuntime_Run_SimpleLinear(t *testing.T) {
	// Create a simple linear graph: a -> b -> c
	g := NewGraph("linear")
	g.AddNode(NewFuncNode("a", func(ctx context.Context, env *Envelope) (*Envelope, error) {
		env.SetVar("a", "processed")
		return env, nil
	}))
	g.AddNode(NewFuncNode("b", func(ctx context.Context, env *Envelope) (*Envelope, error) {
		env.SetVar("b", "processed")
		return env, nil
	}))
	g.AddNode(NewFuncNode("c", func(ctx context.Context, env *Envelope) (*Envelope, error) {
		env.SetVar("c", "processed")
		return env, nil
	}))
	g.AddEdge("a", "b")
	g.AddEdge("b", "c")
	g.SetEntry("a")

	rt := NewRuntime()
	env := NewEnvelope()

	result, err := rt.Run(context.Background(), g, env, DefaultRunOptions())

	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	// Check all nodes were executed
	for _, key := range []string{"a", "b", "c"} {
		if v, ok := result.GetVar(key); !ok || v != "processed" {
			t.Errorf("Node %s was not executed correctly", key)
		}
	}

	// Check trace was set
	if result.Trace.RunID == "" {
		t.Error("RunID was not set")
	}
}

func TestRuntime_Run_Events(t *testing.T) {
	g := NewGraph("events-test")
	g.AddNode(NewNoopNode("start"))
	g.SetEntry("start")

	rt := NewRuntime()
	events := make([]Event, 0)

	opts := DefaultRunOptions()
	opts.EventHandler = func(e Event) {
		events = append(events, e)
	}

	_, err := rt.Run(context.Background(), g, NewEnvelope(), opts)

	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	// Should have: run_started, node_started, node_finished, run_finished
	expectedKinds := []EventKind{
		EventRunStarted,
		EventNodeStarted,
		EventNodeFinished,
		EventRunFinished,
	}

	if len(events) != len(expectedKinds) {
		t.Errorf("len(events) = %v, want %v", len(events), len(expectedKinds))
	}

	for i, expected := range expectedKinds {
		if i < len(events) && events[i].Kind != expected {
			t.Errorf("events[%d].Kind = %v, want %v", i, events[i].Kind, expected)
		}
	}
}

func TestRuntime_Run_NilEnvelope(t *testing.T) {
	g := NewGraph("nil-env")
	g.AddNode(NewNoopNode("start"))
	g.SetEntry("start")

	rt := NewRuntime()

	result, err := rt.Run(context.Background(), g, nil, DefaultRunOptions())

	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
	if result == nil {
		t.Error("Run() should create envelope if nil")
	}
}

func TestRuntime_Run_NilGraph(t *testing.T) {
	rt := NewRuntime()

	_, err := rt.Run(context.Background(), nil, NewEnvelope(), DefaultRunOptions())

	if err == nil {
		t.Error("Run() should error on nil graph")
	}
}

func TestRuntime_Run_EmptyGraph(t *testing.T) {
	g := NewGraph("empty")
	rt := NewRuntime()

	_, err := rt.Run(context.Background(), g, NewEnvelope(), DefaultRunOptions())

	if !errors.Is(err, ErrEmptyGraph) {
		t.Errorf("Run() error = %v, want %v", err, ErrEmptyGraph)
	}
}

func TestRuntime_Run_NoEntry(t *testing.T) {
	g := NewGraph("no-entry")
	g.AddNode(NewNoopNode("orphan"))

	rt := NewRuntime()

	_, err := rt.Run(context.Background(), g, NewEnvelope(), DefaultRunOptions())

	if !errors.Is(err, ErrNoEntryNode) {
		t.Errorf("Run() error = %v, want %v", err, ErrNoEntryNode)
	}
}

func TestRuntime_Run_ContextCancellation(t *testing.T) {
	g := NewGraph("cancel-test")
	g.AddNode(NewFuncNode("slow", func(ctx context.Context, env *Envelope) (*Envelope, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(10 * time.Second):
			return env, nil
		}
	}))
	g.SetEntry("slow")

	rt := NewRuntime()
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := rt.Run(ctx, g, NewEnvelope(), DefaultRunOptions())

	if err == nil {
		t.Error("Run() should error on context cancellation")
	}
}

func TestRuntime_Run_NodeError(t *testing.T) {
	expectedErr := errors.New("node failed")
	g := NewGraph("error-test")
	g.AddNode(NewFuncNode("fail", func(ctx context.Context, env *Envelope) (*Envelope, error) {
		return nil, expectedErr
	}))
	g.SetEntry("fail")

	rt := NewRuntime()

	_, err := rt.Run(context.Background(), g, NewEnvelope(), DefaultRunOptions())

	if !errors.Is(err, ErrNodeExecution) {
		t.Errorf("Run() error = %v, want wrapped %v", err, ErrNodeExecution)
	}
}

func TestRuntime_Run_ContinueOnError(t *testing.T) {
	g := NewGraph("continue-test")
	g.AddNode(NewFuncNode("fail", func(ctx context.Context, env *Envelope) (*Envelope, error) {
		return nil, errors.New("expected failure")
	}))
	g.AddNode(NewFuncNode("after", func(ctx context.Context, env *Envelope) (*Envelope, error) {
		env.SetVar("after", "reached")
		return env, nil
	}))
	g.AddEdge("fail", "after")
	g.SetEntry("fail")

	rt := NewRuntime()
	opts := DefaultRunOptions()
	opts.ContinueOnError = true

	result, err := rt.Run(context.Background(), g, NewEnvelope(), opts)

	if err != nil {
		t.Errorf("Run() with ContinueOnError error = %v, want nil", err)
	}

	// Error should be recorded
	if !result.HasErrors() {
		t.Error("Error should be recorded in envelope")
	}
	if len(result.Errors) != 1 {
		t.Errorf("len(Errors) = %v, want 1", len(result.Errors))
	}

	// After node should have executed
	if v, ok := result.GetVar("after"); !ok || v != "reached" {
		t.Error("Node 'after' should have executed")
	}
}

func TestRuntime_Run_EventNodeFailed(t *testing.T) {
	g := NewGraph("fail-event")
	g.AddNode(NewFuncNode("fail", func(ctx context.Context, env *Envelope) (*Envelope, error) {
		return nil, errors.New("boom")
	}))
	g.SetEntry("fail")

	rt := NewRuntime()
	var failedEvent *Event

	opts := DefaultRunOptions()
	opts.EventHandler = func(e Event) {
		if e.Kind == EventNodeFailed {
			failedEvent = &e
		}
	}

	rt.Run(context.Background(), g, NewEnvelope(), opts)

	if failedEvent == nil {
		t.Error("EventNodeFailed should be emitted")
	}
	if failedEvent != nil && failedEvent.Payload["error"] != "boom" {
		t.Error("EventNodeFailed should contain error message")
	}
}

func TestRuntime_Run_ExecutionOrder(t *testing.T) {
	order := make([]string, 0)

	g := NewGraph("order-test")
	for _, id := range []string{"a", "b", "c"} {
		nodeID := id
		g.AddNode(NewFuncNode(nodeID, func(ctx context.Context, env *Envelope) (*Envelope, error) {
			order = append(order, nodeID)
			return env, nil
		}))
	}
	g.AddEdge("a", "b")
	g.AddEdge("b", "c")
	g.SetEntry("a")

	rt := NewRuntime()
	rt.Run(context.Background(), g, NewEnvelope(), DefaultRunOptions())

	expected := []string{"a", "b", "c"}
	if len(order) != len(expected) {
		t.Errorf("len(order) = %v, want %v", len(order), len(expected))
	}
	for i, id := range expected {
		if i < len(order) && order[i] != id {
			t.Errorf("order[%d] = %v, want %v", i, order[i], id)
		}
	}
}

func TestRuntime_Run_BranchingGraph(t *testing.T) {
	// Graph: a -> b, a -> c, b -> d, c -> d
	executed := make(map[string]bool)

	g := NewGraph("branch")
	for _, id := range []string{"a", "b", "c", "d"} {
		nodeID := id
		g.AddNode(NewFuncNode(nodeID, func(ctx context.Context, env *Envelope) (*Envelope, error) {
			executed[nodeID] = true
			return env, nil
		}))
	}
	g.AddEdge("a", "b")
	g.AddEdge("a", "c")
	g.AddEdge("b", "d")
	g.AddEdge("c", "d")
	g.SetEntry("a")

	rt := NewRuntime()
	rt.Run(context.Background(), g, NewEnvelope(), DefaultRunOptions())

	// All nodes should be executed
	for _, id := range []string{"a", "b", "c", "d"} {
		if !executed[id] {
			t.Errorf("Node %s was not executed", id)
		}
	}
}

func TestRuntime_Run_RunIDGenerated(t *testing.T) {
	g := NewGraph("runid-test")
	g.AddNode(NewNoopNode("start"))
	g.SetEntry("start")

	rt := NewRuntime()
	result, _ := rt.Run(context.Background(), g, NewEnvelope(), DefaultRunOptions())

	if result.Trace.RunID == "" {
		t.Error("RunID should be generated")
	}

	// Run again and verify different ID
	result2, _ := rt.Run(context.Background(), g, NewEnvelope(), DefaultRunOptions())
	if result2.Trace.RunID == result.Trace.RunID {
		t.Error("Different runs should have different RunIDs")
	}
}

func TestRuntime_Run_CustomNow(t *testing.T) {
	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	g := NewGraph("time-test")
	g.AddNode(NewNoopNode("start"))
	g.SetEntry("start")

	rt := NewRuntime()
	opts := DefaultRunOptions()
	opts.Now = func() time.Time { return fixedTime }

	result, _ := rt.Run(context.Background(), g, NewEnvelope(), opts)

	if result.Trace.Started != fixedTime {
		t.Errorf("Trace.Started = %v, want %v", result.Trace.Started, fixedTime)
	}
}

func TestRuntime_InterfaceCompliance(t *testing.T) {
	var _ Runtime = (*BasicRuntime)(nil)
}

func TestGenerateRunID(t *testing.T) {
	id1 := generateRunID()
	id2 := generateRunID()

	if id1 == "" {
		t.Error("generateRunID() returned empty string")
	}
	if id1 == id2 {
		t.Error("generateRunID() should return unique IDs")
	}
}
