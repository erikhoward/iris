package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/erikhoward/iris/tools"
)

// mockTool is a mock implementation of tools.Tool for testing.
type mockTool struct {
	name        string
	description string
	callResult  any
	callError   error
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Description() string {
	return m.description
}

func (m *mockTool) Schema() tools.ToolSchema {
	return tools.ToolSchema{
		JSONSchema: json.RawMessage(`{"type": "object"}`),
	}
}

func (m *mockTool) Call(ctx context.Context, args json.RawMessage) (any, error) {
	if m.callError != nil {
		return nil, m.callError
	}
	return m.callResult, nil
}

func TestNewToolAdapter(t *testing.T) {
	mock := &mockTool{name: "test-tool", description: "A test tool"}
	adapter := NewToolAdapter(mock)

	if adapter == nil {
		t.Fatal("expected adapter to be non-nil")
	}
	if adapter.Name() != "test-tool" {
		t.Errorf("expected name 'test-tool', got %q", adapter.Name())
	}
	if adapter.Description() != "A test tool" {
		t.Errorf("expected description 'A test tool', got %q", adapter.Description())
	}
}

func TestToolAdapter_Invoke_MapResult(t *testing.T) {
	mock := &mockTool{
		name:       "test-tool",
		callResult: map[string]any{"result": "success", "count": 42},
	}

	adapter := NewToolAdapter(mock)
	result, err := adapter.Invoke(context.Background(), map[string]any{"input": "test"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["result"] != "success" {
		t.Errorf("expected result 'success', got %v", result["result"])
	}
	// When result is already a map, it's returned directly (no JSON round-trip)
	if result["count"] != 42 {
		t.Errorf("expected count 42, got %v", result["count"])
	}
}

func TestToolAdapter_Invoke_StructResult(t *testing.T) {
	type TestResult struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	mock := &mockTool{
		name:       "test-tool",
		callResult: TestResult{Name: "test", Value: 100},
	}

	adapter := NewToolAdapter(mock)
	result, err := adapter.Invoke(context.Background(), map[string]any{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["name"] != "test" {
		t.Errorf("expected name 'test', got %v", result["name"])
	}
	if result["value"] != float64(100) {
		t.Errorf("expected value 100, got %v", result["value"])
	}
}

func TestToolAdapter_Invoke_NilResult(t *testing.T) {
	mock := &mockTool{
		name:       "test-tool",
		callResult: nil,
	}

	adapter := NewToolAdapter(mock)
	result, err := adapter.Invoke(context.Background(), map[string]any{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil empty map for nil result")
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestToolAdapter_Invoke_Error(t *testing.T) {
	expectedErr := errors.New("tool failed")
	mock := &mockTool{
		name:      "test-tool",
		callError: expectedErr,
	}

	adapter := NewToolAdapter(mock)
	_, err := adapter.Invoke(context.Background(), map[string]any{})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to contain original error")
	}
}

func TestToolAdapter_Invoke_PrimitiveResult(t *testing.T) {
	// When result is a primitive that can't be unmarshaled to map,
	// it should be wrapped in {"result": value}
	mock := &mockTool{
		name:       "test-tool",
		callResult: "simple string result",
	}

	adapter := NewToolAdapter(mock)
	result, err := adapter.Invoke(context.Background(), map[string]any{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["result"] != "simple string result" {
		t.Errorf("expected wrapped result, got %v", result)
	}
}

func TestNewFuncTool(t *testing.T) {
	fn := func(ctx context.Context, args map[string]any) (map[string]any, error) {
		return map[string]any{"ok": true}, nil
	}

	tool := NewFuncTool("my-func", "A function tool", fn)

	if tool.Name() != "my-func" {
		t.Errorf("expected name 'my-func', got %q", tool.Name())
	}
}

func TestFuncTool_Invoke(t *testing.T) {
	fn := func(ctx context.Context, args map[string]any) (map[string]any, error) {
		input := args["input"].(string)
		return map[string]any{"output": "processed: " + input}, nil
	}

	tool := NewFuncTool("my-func", "A function tool", fn)
	result, err := tool.Invoke(context.Background(), map[string]any{"input": "test"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["output"] != "processed: test" {
		t.Errorf("expected 'processed: test', got %v", result["output"])
	}
}

func TestFuncTool_Invoke_NilFn(t *testing.T) {
	tool := NewFuncTool("my-func", "A function tool", nil)
	result, err := tool.Invoke(context.Background(), map[string]any{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result for nil function, got %v", result)
	}
}

func TestFuncTool_Invoke_Error(t *testing.T) {
	expectedErr := errors.New("function failed")
	fn := func(ctx context.Context, args map[string]any) (map[string]any, error) {
		return nil, expectedErr
	}

	tool := NewFuncTool("my-func", "A function tool", fn)
	_, err := tool.Invoke(context.Background(), map[string]any{})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected original error, got %v", err)
	}
}

func TestNewToolRegistry(t *testing.T) {
	registry := NewToolRegistry()

	if registry == nil {
		t.Fatal("expected registry to be non-nil")
	}
	if len(registry.List()) != 0 {
		t.Errorf("expected empty registry, got %d tools", len(registry.List()))
	}
}

func TestToolRegistry_Register(t *testing.T) {
	registry := NewToolRegistry()

	tool1 := NewFuncTool("tool1", "Tool 1", nil)
	tool2 := NewFuncTool("tool2", "Tool 2", nil)

	registry.Register(tool1)
	registry.Register(tool2)

	if len(registry.List()) != 2 {
		t.Errorf("expected 2 tools, got %d", len(registry.List()))
	}
}

func TestToolRegistry_Get(t *testing.T) {
	registry := NewToolRegistry()

	tool := NewFuncTool("my-tool", "My Tool", nil)
	registry.Register(tool)

	retrieved, ok := registry.Get("my-tool")
	if !ok {
		t.Fatal("expected tool to be found")
	}
	if retrieved.Name() != "my-tool" {
		t.Errorf("expected name 'my-tool', got %q", retrieved.Name())
	}

	_, ok = registry.Get("nonexistent")
	if ok {
		t.Error("expected nonexistent tool to not be found")
	}
}

func TestToolRegistry_Register_Override(t *testing.T) {
	registry := NewToolRegistry()

	tool1 := NewFuncTool("tool", "First version", func(ctx context.Context, args map[string]any) (map[string]any, error) {
		return map[string]any{"version": 1}, nil
	})
	tool2 := NewFuncTool("tool", "Second version", func(ctx context.Context, args map[string]any) (map[string]any, error) {
		return map[string]any{"version": 2}, nil
	})

	registry.Register(tool1)
	registry.Register(tool2)

	retrieved, ok := registry.Get("tool")
	if !ok {
		t.Fatal("expected tool to be found")
	}

	result, _ := retrieved.Invoke(context.Background(), nil)
	// FuncTool returns map directly, no JSON round-trip
	if result["version"] != 2 {
		t.Errorf("expected version 2 (latest), got %v", result["version"])
	}
}

func TestToolRegistry_List(t *testing.T) {
	registry := NewToolRegistry()

	registry.Register(NewFuncTool("alpha", "", nil))
	registry.Register(NewFuncTool("beta", "", nil))
	registry.Register(NewFuncTool("gamma", "", nil))

	names := registry.List()
	if len(names) != 3 {
		t.Errorf("expected 3 names, got %d", len(names))
	}

	// Check all names are present (order not guaranteed)
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	for _, expected := range []string{"alpha", "beta", "gamma"} {
		if !nameSet[expected] {
			t.Errorf("expected name %q to be in list", expected)
		}
	}
}

func TestToResultMap_Map(t *testing.T) {
	input := map[string]any{"key": "value"}
	result, err := toResultMap(input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("expected 'value', got %v", result["key"])
	}
}

func TestToResultMap_Nil(t *testing.T) {
	result, err := toResultMap(nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestPetalTool_InterfaceCompliance(t *testing.T) {
	var _ PetalTool = (*ToolAdapter)(nil)
	var _ PetalTool = (*FuncTool)(nil)
}
