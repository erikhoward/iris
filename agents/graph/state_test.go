package graph

import "testing"

func TestNewState(t *testing.T) {
	s := NewState()

	if s == nil {
		t.Fatal("NewState() returned nil")
	}

	if s.Data == nil {
		t.Error("NewState().Data is nil")
	}

	if len(s.Data) != 0 {
		t.Errorf("len(NewState().Data) = %d, want 0", len(s.Data))
	}
}

func TestStateGetSet(t *testing.T) {
	s := NewState()

	// Set a value
	s.Set("key", "value")

	// Get it back
	v, ok := s.Get("key")
	if !ok {
		t.Error("Get() returned false for existing key")
	}
	if v != "value" {
		t.Errorf("Get() = %v, want %q", v, "value")
	}
}

func TestStateGetMissing(t *testing.T) {
	s := NewState()

	v, ok := s.Get("nonexistent")
	if ok {
		t.Error("Get() returned true for nonexistent key")
	}
	if v != nil {
		t.Errorf("Get() = %v, want nil", v)
	}
}

func TestStateGetNilData(t *testing.T) {
	s := &State{Data: nil}

	v, ok := s.Get("key")
	if ok {
		t.Error("Get() on nil Data returned true")
	}
	if v != nil {
		t.Errorf("Get() on nil Data = %v, want nil", v)
	}
}

func TestStateSetNilData(t *testing.T) {
	s := &State{Data: nil}

	// Should not panic
	s.Set("key", "value")

	v, ok := s.Get("key")
	if !ok {
		t.Error("Get() returned false after Set() on nil Data")
	}
	if v != "value" {
		t.Errorf("Get() = %v, want %q", v, "value")
	}
}

func TestStateSetOverwrite(t *testing.T) {
	s := NewState()

	s.Set("key", "first")
	s.Set("key", "second")

	v, _ := s.Get("key")
	if v != "second" {
		t.Errorf("Get() = %v, want %q", v, "second")
	}
}

func TestStateSetDifferentTypes(t *testing.T) {
	s := NewState()

	s.Set("string", "hello")
	s.Set("int", 42)
	s.Set("bool", true)
	s.Set("slice", []int{1, 2, 3})

	if v, _ := s.Get("string"); v != "hello" {
		t.Errorf("string = %v, want hello", v)
	}
	if v, _ := s.Get("int"); v != 42 {
		t.Errorf("int = %v, want 42", v)
	}
	if v, _ := s.Get("bool"); v != true {
		t.Errorf("bool = %v, want true", v)
	}
	if v, _ := s.Get("slice"); len(v.([]int)) != 3 {
		t.Errorf("slice length = %v, want 3", len(v.([]int)))
	}
}

func TestStateClone(t *testing.T) {
	s := NewState()
	s.Set("key1", "value1")
	s.Set("key2", "value2")

	clone := s.Clone()

	// Verify clone has same values
	if v, _ := clone.Get("key1"); v != "value1" {
		t.Errorf("clone key1 = %v, want value1", v)
	}
	if v, _ := clone.Get("key2"); v != "value2" {
		t.Errorf("clone key2 = %v, want value2", v)
	}
}

func TestStateCloneIndependent(t *testing.T) {
	s := NewState()
	s.Set("key", "original")

	clone := s.Clone()

	// Modify original
	s.Set("key", "modified")

	// Clone should not be affected
	if v, _ := clone.Get("key"); v != "original" {
		t.Errorf("clone key = %v, want original", v)
	}
}

func TestStateCloneModifyClone(t *testing.T) {
	s := NewState()
	s.Set("key", "original")

	clone := s.Clone()

	// Modify clone
	clone.Set("key", "modified")
	clone.Set("new", "value")

	// Original should not be affected
	if v, _ := s.Get("key"); v != "original" {
		t.Errorf("original key = %v, want original", v)
	}
	if _, ok := s.Get("new"); ok {
		t.Error("original has 'new' key after modifying clone")
	}
}

func TestStateCloneEmpty(t *testing.T) {
	s := NewState()
	clone := s.Clone()

	if clone == nil {
		t.Fatal("Clone() returned nil")
	}
	if len(clone.Data) != 0 {
		t.Errorf("Clone of empty state has %d items", len(clone.Data))
	}
}
