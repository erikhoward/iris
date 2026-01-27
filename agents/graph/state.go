// Package graph provides a minimal agent graph framework for Iris.
package graph

// State holds data that flows between nodes in a graph.
// State is not safe for concurrent use.
type State struct {
	Data map[string]any
}

// NewState creates a new empty State.
func NewState() *State {
	return &State{
		Data: make(map[string]any),
	}
}

// Get retrieves a value from the state by key.
// Returns the value and true if found, or nil and false if not found.
func (s *State) Get(key string) (any, bool) {
	if s.Data == nil {
		return nil, false
	}
	v, ok := s.Data[key]
	return v, ok
}

// Set stores a value in the state by key.
func (s *State) Set(key string, value any) {
	if s.Data == nil {
		s.Data = make(map[string]any)
	}
	s.Data[key] = value
}

// Clone creates a shallow copy of the state.
// The map is copied, but values are not deep-copied.
func (s *State) Clone() *State {
	clone := NewState()
	for k, v := range s.Data {
		clone.Data[k] = v
	}
	return clone
}
