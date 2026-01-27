package commands

import (
	"fmt"
	"os"

	"github.com/erikhoward/iris/agents/graph"
	"github.com/spf13/cobra"
)

var (
	graphFormat string
	graphOutput string
)

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Agent graph operations",
	Long:  `Commands for working with agent graphs, including exporting to visualization formats.`,
}

var graphExportCmd = &cobra.Command{
	Use:   "export <spec-file>",
	Short: "Export agent graph to visualization format",
	Long: `Export an agent graph specification to Mermaid or JSON format.

Examples:
  iris graph export agent.yaml
  iris graph export agent.yaml --format json
  iris graph export agent.yaml --format mermaid --output graph.md`,
	Args: cobra.ExactArgs(1),
	RunE: runGraphExport,
}

func init() {
	rootCmd.AddCommand(graphCmd)
	graphCmd.AddCommand(graphExportCmd)

	graphExportCmd.Flags().StringVar(&graphFormat, "format", "mermaid", "Output format: mermaid, json")
	graphExportCmd.Flags().StringVar(&graphOutput, "output", "", "Output file (default: stdout)")
}

func runGraphExport(cmd *cobra.Command, args []string) error {
	specPath := args[0]

	// Load the graph spec
	spec, err := graph.LoadGraphSpec(specPath)
	if err != nil {
		return fmt.Errorf("failed to load graph spec: %w", err)
	}

	// Generate output based on format
	var output []byte
	switch graphFormat {
	case "mermaid":
		output = []byte(spec.ToMermaid())
	case "json":
		jsonData, err := spec.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to generate JSON: %w", err)
		}
		output = jsonData
	default:
		return fmt.Errorf("unsupported format: %s (use 'mermaid' or 'json')", graphFormat)
	}

	// Write output
	if graphOutput != "" {
		if err := os.WriteFile(graphOutput, output, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("Graph exported to %s\n", graphOutput)
	} else {
		fmt.Print(string(output))
	}

	return nil
}
