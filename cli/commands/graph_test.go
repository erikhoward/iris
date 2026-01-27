package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGraphExportMermaid(t *testing.T) {
	// Create test YAML file
	content := `
name: test-agent
entry: start

nodes:
  - id: start
    type: prompt
  - id: end
    type: output

edges:
  - from: start
    to: end
`
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "agent.yaml")
	if err := os.WriteFile(specPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test with output file
	outputPath := filepath.Join(tmpDir, "graph.md")
	graphFormat = "mermaid"
	graphOutput = outputPath

	err := runGraphExport(nil, []string{specPath})
	if err != nil {
		t.Fatalf("runGraphExport() error = %v", err)
	}

	// Verify output file
	output, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if !strings.Contains(string(output), "graph TD") {
		t.Error("Output should contain 'graph TD'")
	}
	if !strings.Contains(string(output), "start[start]") {
		t.Error("Output should contain 'start[start]'")
	}
	if !strings.Contains(string(output), "start --> end") {
		t.Error("Output should contain 'start --> end'")
	}
}

func TestGraphExportJSON(t *testing.T) {
	content := `
name: test-agent
entry: start

nodes:
  - id: start
    type: prompt
  - id: end
    type: output

edges:
  - from: start
    to: end
`
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "agent.yaml")
	if err := os.WriteFile(specPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	outputPath := filepath.Join(tmpDir, "graph.json")
	graphFormat = "json"
	graphOutput = outputPath

	err := runGraphExport(nil, []string{specPath})
	if err != nil {
		t.Fatalf("runGraphExport() error = %v", err)
	}

	output, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if !strings.Contains(string(output), `"name": "test-agent"`) {
		t.Error("Output should contain name")
	}
	if !strings.Contains(string(output), `"entry": "start"`) {
		t.Error("Output should contain entry")
	}
}

func TestGraphExportInvalidFormat(t *testing.T) {
	content := `
name: test-agent
entry: start
nodes:
  - id: start
    type: prompt
`
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "agent.yaml")
	if err := os.WriteFile(specPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	graphFormat = "invalid"
	graphOutput = ""

	err := runGraphExport(nil, []string{specPath})
	if err == nil {
		t.Error("runGraphExport() should return error for invalid format")
	}

	if !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("Error should mention unsupported format, got: %v", err)
	}
}

func TestGraphExportMissingFile(t *testing.T) {
	graphFormat = "mermaid"
	graphOutput = ""

	err := runGraphExport(nil, []string{"/nonexistent/file.yaml"})
	if err == nil {
		t.Error("runGraphExport() should return error for missing file")
	}
}

func TestGraphExportInvalidYAML(t *testing.T) {
	content := `
name: [invalid yaml syntax
`
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(specPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	graphFormat = "mermaid"
	graphOutput = ""

	err := runGraphExport(nil, []string{specPath})
	if err == nil {
		t.Error("runGraphExport() should return error for invalid YAML")
	}
}

func TestGraphExportInvalidSpec(t *testing.T) {
	// Missing required fields
	content := `
name: test-agent
`
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "incomplete.yaml")
	if err := os.WriteFile(specPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	graphFormat = "mermaid"
	graphOutput = ""

	err := runGraphExport(nil, []string{specPath})
	if err == nil {
		t.Error("runGraphExport() should return error for invalid spec")
	}
}
