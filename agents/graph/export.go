package graph

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// GraphSpec represents an agent graph specification loaded from YAML.
type GraphSpec struct {
	Name  string     `yaml:"name" json:"name"`
	Entry string     `yaml:"entry" json:"entry"`
	Nodes []NodeSpec `yaml:"nodes" json:"nodes"`
	Edges []EdgeSpec `yaml:"edges" json:"edges"`
}

// NodeSpec represents a node in the graph specification.
type NodeSpec struct {
	ID     string         `yaml:"id" json:"id"`
	Type   string         `yaml:"type" json:"type"`
	Config map[string]any `yaml:"config,omitempty" json:"config,omitempty"`
}

// EdgeSpec represents an edge between nodes in the graph specification.
type EdgeSpec struct {
	From string `yaml:"from" json:"from"`
	To   string `yaml:"to" json:"to"`
}

// LoadGraphSpec loads a GraphSpec from a YAML file.
func LoadGraphSpec(path string) (*GraphSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var spec GraphSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if err := spec.Validate(); err != nil {
		return nil, err
	}

	return &spec, nil
}

// Validate checks that the GraphSpec is valid.
func (g *GraphSpec) Validate() error {
	if g.Name == "" {
		return fmt.Errorf("graph spec missing name")
	}
	if g.Entry == "" {
		return fmt.Errorf("graph spec missing entry")
	}
	if len(g.Nodes) == 0 {
		return fmt.Errorf("graph spec has no nodes")
	}

	// Check that entry node exists
	nodeIDs := make(map[string]bool)
	for _, node := range g.Nodes {
		if node.ID == "" {
			return fmt.Errorf("node missing id")
		}
		nodeIDs[node.ID] = true
	}

	if !nodeIDs[g.Entry] {
		return fmt.Errorf("entry node %q not found in nodes", g.Entry)
	}

	// Check that all edges reference valid nodes
	for _, edge := range g.Edges {
		if !nodeIDs[edge.From] {
			return fmt.Errorf("edge references unknown node %q", edge.From)
		}
		if !nodeIDs[edge.To] {
			return fmt.Errorf("edge references unknown node %q", edge.To)
		}
	}

	return nil
}

// ToMermaid exports the graph to Mermaid diagram syntax.
func (g *GraphSpec) ToMermaid() string {
	var sb strings.Builder

	sb.WriteString("graph TD\n")

	// Write nodes
	for _, node := range g.Nodes {
		// Use square brackets for node labels
		sb.WriteString(fmt.Sprintf("    %s[%s]\n", node.ID, node.ID))
	}

	// Write edges
	for _, edge := range g.Edges {
		sb.WriteString(fmt.Sprintf("    %s --> %s\n", edge.From, edge.To))
	}

	return sb.String()
}

// ToJSON exports the graph to JSON format.
func (g *GraphSpec) ToJSON() ([]byte, error) {
	return json.MarshalIndent(g, "", "  ")
}
