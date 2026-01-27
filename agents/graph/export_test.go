package graph

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadGraphSpec(t *testing.T) {
	content := `
name: test-agent
entry: start

nodes:
  - id: start
    type: prompt
    config:
      template: "Hello {{.input}}"
  - id: process
    type: llm
    config:
      model: gpt-4o
  - id: end
    type: output

edges:
  - from: start
    to: process
  - from: process
    to: end
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "agent.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	spec, err := LoadGraphSpec(path)
	if err != nil {
		t.Fatalf("LoadGraphSpec() error = %v", err)
	}

	if spec.Name != "test-agent" {
		t.Errorf("Name = %q, want 'test-agent'", spec.Name)
	}
	if spec.Entry != "start" {
		t.Errorf("Entry = %q, want 'start'", spec.Entry)
	}
	if len(spec.Nodes) != 3 {
		t.Errorf("len(Nodes) = %d, want 3", len(spec.Nodes))
	}
	if len(spec.Edges) != 2 {
		t.Errorf("len(Edges) = %d, want 2", len(spec.Edges))
	}
}

func TestLoadGraphSpecMissingFile(t *testing.T) {
	_, err := LoadGraphSpec("/nonexistent/file.yaml")
	if err == nil {
		t.Error("LoadGraphSpec() should return error for missing file")
	}
}

func TestLoadGraphSpecInvalidYAML(t *testing.T) {
	content := `
name: [invalid yaml
entry: start
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err := LoadGraphSpec(path)
	if err == nil {
		t.Error("LoadGraphSpec() should return error for invalid YAML")
	}
}

func TestGraphSpecValidate(t *testing.T) {
	tests := []struct {
		name    string
		spec    GraphSpec
		wantErr bool
	}{
		{
			name: "valid",
			spec: GraphSpec{
				Name:  "test",
				Entry: "start",
				Nodes: []NodeSpec{{ID: "start", Type: "prompt"}},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			spec: GraphSpec{
				Entry: "start",
				Nodes: []NodeSpec{{ID: "start", Type: "prompt"}},
			},
			wantErr: true,
		},
		{
			name: "missing entry",
			spec: GraphSpec{
				Name:  "test",
				Nodes: []NodeSpec{{ID: "start", Type: "prompt"}},
			},
			wantErr: true,
		},
		{
			name: "no nodes",
			spec: GraphSpec{
				Name:  "test",
				Entry: "start",
			},
			wantErr: true,
		},
		{
			name: "entry not in nodes",
			spec: GraphSpec{
				Name:  "test",
				Entry: "missing",
				Nodes: []NodeSpec{{ID: "start", Type: "prompt"}},
			},
			wantErr: true,
		},
		{
			name: "edge references unknown from",
			spec: GraphSpec{
				Name:  "test",
				Entry: "start",
				Nodes: []NodeSpec{{ID: "start", Type: "prompt"}},
				Edges: []EdgeSpec{{From: "unknown", To: "start"}},
			},
			wantErr: true,
		},
		{
			name: "edge references unknown to",
			spec: GraphSpec{
				Name:  "test",
				Entry: "start",
				Nodes: []NodeSpec{{ID: "start", Type: "prompt"}},
				Edges: []EdgeSpec{{From: "start", To: "unknown"}},
			},
			wantErr: true,
		},
		{
			name: "node missing id",
			spec: GraphSpec{
				Name:  "test",
				Entry: "start",
				Nodes: []NodeSpec{{Type: "prompt"}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGraphSpecToMermaid(t *testing.T) {
	spec := &GraphSpec{
		Name:  "test-agent",
		Entry: "start",
		Nodes: []NodeSpec{
			{ID: "start", Type: "prompt"},
			{ID: "process", Type: "llm"},
			{ID: "end", Type: "output"},
		},
		Edges: []EdgeSpec{
			{From: "start", To: "process"},
			{From: "process", To: "end"},
		},
	}

	result := spec.ToMermaid()

	// Check header
	if !strings.HasPrefix(result, "graph TD\n") {
		t.Error("Mermaid output should start with 'graph TD'")
	}

	// Check nodes are present
	if !strings.Contains(result, "start[start]") {
		t.Error("Mermaid output should contain 'start[start]'")
	}
	if !strings.Contains(result, "process[process]") {
		t.Error("Mermaid output should contain 'process[process]'")
	}
	if !strings.Contains(result, "end[end]") {
		t.Error("Mermaid output should contain 'end[end]'")
	}

	// Check edges are present
	if !strings.Contains(result, "start --> process") {
		t.Error("Mermaid output should contain 'start --> process'")
	}
	if !strings.Contains(result, "process --> end") {
		t.Error("Mermaid output should contain 'process --> end'")
	}
}

func TestGraphSpecToJSON(t *testing.T) {
	spec := &GraphSpec{
		Name:  "test-agent",
		Entry: "start",
		Nodes: []NodeSpec{
			{ID: "start", Type: "prompt"},
			{ID: "end", Type: "output"},
		},
		Edges: []EdgeSpec{
			{From: "start", To: "end"},
		},
	}

	data, err := spec.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// Verify it's valid JSON by parsing it back
	var parsed GraphSpec
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON output is not valid: %v", err)
	}

	if parsed.Name != spec.Name {
		t.Errorf("Parsed Name = %q, want %q", parsed.Name, spec.Name)
	}
	if parsed.Entry != spec.Entry {
		t.Errorf("Parsed Entry = %q, want %q", parsed.Entry, spec.Entry)
	}
	if len(parsed.Nodes) != len(spec.Nodes) {
		t.Errorf("Parsed len(Nodes) = %d, want %d", len(parsed.Nodes), len(spec.Nodes))
	}
	if len(parsed.Edges) != len(spec.Edges) {
		t.Errorf("Parsed len(Edges) = %d, want %d", len(parsed.Edges), len(spec.Edges))
	}
}

func TestGraphSpecToJSONWithConfig(t *testing.T) {
	spec := &GraphSpec{
		Name:  "test-agent",
		Entry: "start",
		Nodes: []NodeSpec{
			{
				ID:   "start",
				Type: "prompt",
				Config: map[string]any{
					"template": "Hello {{.input}}",
					"count":    42,
				},
			},
		},
	}

	data, err := spec.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// Verify config is included
	if !strings.Contains(string(data), "template") {
		t.Error("JSON should contain config template")
	}
	if !strings.Contains(string(data), "Hello {{.input}}") {
		t.Error("JSON should contain config value")
	}
}
