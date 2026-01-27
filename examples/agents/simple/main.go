// Example: Simple Agent Graph
//
// This example demonstrates how to build and execute a simple
// agent workflow using the graph framework.
//
// Run with:
//
//	go run main.go
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/erikhoward/iris/agents/graph"
)

func main() {
	// Create a new graph
	g := graph.NewGraph()

	// Node 1: Input processing
	inputNode := graph.NewFuncNode("input", func(ctx context.Context, state *graph.State) (*graph.State, error) {
		fmt.Println("[input] Processing input...")

		// Get input from state (or use default)
		text := "Hello, World!"
		if v, ok := state.Get("text"); ok {
			if s, ok := v.(string); ok && s != "" {
				text = s
			}
		}

		// Store processed input
		newState := state.Clone()
		newState.Set("original", text)
		newState.Set("processed", strings.TrimSpace(text))

		fmt.Printf("[input] Original: %q\n", text)
		return newState, nil
	})

	// Node 2: Transform the text
	transformNode := graph.NewFuncNode("transform", func(ctx context.Context, state *graph.State) (*graph.State, error) {
		fmt.Println("[transform] Transforming text...")

		processed, _ := state.Get("processed")
		processedStr := processed.(string)

		newState := state.Clone()
		newState.Set("uppercase", strings.ToUpper(processedStr))
		newState.Set("lowercase", strings.ToLower(processedStr))
		newState.Set("length", len(processedStr))

		fmt.Printf("[transform] Uppercase: %s\n", strings.ToUpper(processedStr))
		return newState, nil
	})

	// Node 3: Output results
	outputNode := graph.NewFuncNode("output", func(ctx context.Context, state *graph.State) (*graph.State, error) {
		fmt.Println("[output] Final results:")
		original, _ := state.Get("original")
		uppercase, _ := state.Get("uppercase")
		lowercase, _ := state.Get("lowercase")
		length, _ := state.Get("length")
		fmt.Printf("  Original:  %v\n", original)
		fmt.Printf("  Uppercase: %v\n", uppercase)
		fmt.Printf("  Lowercase: %v\n", lowercase)
		fmt.Printf("  Length:    %v\n", length)
		return state, nil
	})

	// Add nodes to graph
	if err := g.AddNode(inputNode); err != nil {
		fmt.Fprintln(os.Stderr, "Error adding node:", err)
		os.Exit(1)
	}
	if err := g.AddNode(transformNode); err != nil {
		fmt.Fprintln(os.Stderr, "Error adding node:", err)
		os.Exit(1)
	}
	if err := g.AddNode(outputNode); err != nil {
		fmt.Fprintln(os.Stderr, "Error adding node:", err)
		os.Exit(1)
	}

	// Connect nodes with edges
	if err := g.AddEdge("input", "transform"); err != nil {
		fmt.Fprintln(os.Stderr, "Error adding edge:", err)
		os.Exit(1)
	}
	if err := g.AddEdge("transform", "output"); err != nil {
		fmt.Fprintln(os.Stderr, "Error adding edge:", err)
		os.Exit(1)
	}

	// Set entry point
	if err := g.SetEntry("input"); err != nil {
		fmt.Fprintln(os.Stderr, "Error setting entry:", err)
		os.Exit(1)
	}

	// Create runner
	runner, err := graph.NewRunner(g)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating runner:", err)
		os.Exit(1)
	}

	// Create initial state with input
	initialState := graph.NewState()
	initialState.Set("text", "  Iris Agent Framework  ")

	// Execute the graph
	fmt.Println("=== Starting Graph Execution ===")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	finalState, err := runner.Execute(ctx, initialState)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Execution error:", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("=== Execution Complete ===")
	fmt.Printf("Final state has %d values\n", 5) // We know we set 5 values
	_ = finalState // Use the final state if needed
}
