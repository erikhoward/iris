//go:build integration

// Package integration provides integration tests for the Iris SDK.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/erikhoward/iris/tools"
)

// skipIfNoAPIKey skips the test if IRIS_OPENAI_API_KEY is not set.
func skipIfNoAPIKey(t *testing.T) {
	t.Helper()
	if os.Getenv("IRIS_OPENAI_API_KEY") == "" {
		t.Skip("IRIS_OPENAI_API_KEY not set")
	}
}

// getAPIKey returns the OpenAI API key from environment.
func getAPIKey(t *testing.T) string {
	t.Helper()
	key := os.Getenv("IRIS_OPENAI_API_KEY")
	if key == "" {
		t.Fatal("IRIS_OPENAI_API_KEY not set")
	}
	return key
}

// skipIfNoAnthropicKey skips the test if ANTHROPIC_API_KEY is not set.
func skipIfNoAnthropicKey(t *testing.T) {
	t.Helper()
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}
}

// getAnthropicKey returns the Anthropic API key from environment.
func getAnthropicKey(t *testing.T) string {
	t.Helper()
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		t.Fatal("ANTHROPIC_API_KEY not set")
	}
	return key
}

// skipIfNoGeminiKey skips the test if GEMINI_API_KEY is not set.
func skipIfNoGeminiKey(t *testing.T) {
	t.Helper()
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("GEMINI_API_KEY not set")
	}
}

// getGeminiKey returns the Gemini API key from environment.
func getGeminiKey(t *testing.T) string {
	t.Helper()
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		t.Fatal("GEMINI_API_KEY not set")
	}
	return key
}

// skipIfNoZaiKey skips the test if ZAI_API_KEY is not set.
func skipIfNoZaiKey(t *testing.T) {
	t.Helper()
	if os.Getenv("ZAI_API_KEY") == "" {
		t.Skip("ZAI_API_KEY not set")
	}
}

// getZaiKey returns the Z.ai API key from environment.
func getZaiKey(t *testing.T) string {
	t.Helper()
	key := os.Getenv("ZAI_API_KEY")
	if key == "" {
		t.Fatal("ZAI_API_KEY not set")
	}
	return key
}

// cliResult holds the result of running a CLI command.
type cliResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// runCLI executes the iris CLI with the given arguments.
// It builds the CLI binary first if needed.
func runCLI(t *testing.T, args ...string) cliResult {
	t.Helper()

	// Find the CLI directory
	cliDir := findCLIDir(t)

	// Build the CLI
	buildCmd := exec.Command("go", "build", "-o", "iris-test", "./cmd/iris")
	buildCmd.Dir = cliDir
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build CLI: %v\n%s", err, output)
	}

	// Clean up binary after test
	binaryPath := filepath.Join(cliDir, "iris-test")
	t.Cleanup(func() {
		os.Remove(binaryPath)
	})

	// Run the CLI
	cmd := exec.Command(binaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("Failed to run CLI: %v", err)
		}
	}

	return cliResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
}

// runCLIWithStdin executes the iris CLI with stdin input.
func runCLIWithStdin(t *testing.T, stdin string, args ...string) cliResult {
	t.Helper()

	cliDir := findCLIDir(t)

	// Build the CLI
	buildCmd := exec.Command("go", "build", "-o", "iris-test", "./cmd/iris")
	buildCmd.Dir = cliDir
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build CLI: %v\n%s", err, output)
	}

	binaryPath := filepath.Join(cliDir, "iris-test")
	t.Cleanup(func() {
		os.Remove(binaryPath)
	})

	cmd := exec.Command(binaryPath, args...)
	cmd.Stdin = bytes.NewBufferString(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("Failed to run CLI: %v", err)
		}
	}

	return cliResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
}

// findCLIDir locates the CLI directory relative to the test.
func findCLIDir(t *testing.T) string {
	t.Helper()

	// Try relative paths from tests/integration
	candidates := []string{
		"../../cli",
		"../cli",
		"cli",
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, "cmd", "iris", "main.go")); err == nil {
			abs, _ := filepath.Abs(candidate)
			return abs
		}
	}

	t.Fatal("Could not find CLI directory")
	return ""
}

// findTestDataDir locates the testdata directory.
func findTestDataDir(t *testing.T) string {
	t.Helper()

	candidates := []string{
		"../testdata",
		"testdata",
		"tests/testdata",
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			abs, _ := filepath.Abs(candidate)
			return abs
		}
	}

	t.Fatal("Could not find testdata directory")
	return ""
}

// testTool implements tools.Tool for testing.
type testTool struct {
	name        string
	description string
	schema      json.RawMessage
}

func (t *testTool) Name() string        { return t.name }
func (t *testTool) Description() string { return t.description }
func (t *testTool) Schema() tools.ToolSchema {
	return tools.ToolSchema{JSONSchema: t.schema}
}
func (t *testTool) Call(ctx context.Context, args json.RawMessage) (any, error) {
	return map[string]string{"result": "test result"}, nil
}

// createTestTool creates a simple tool for testing.
func createTestTool() tools.Tool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"location": map[string]any{
				"type":        "string",
				"description": "The city and state, e.g. San Francisco, CA",
			},
		},
		"required": []string{"location"},
	}

	schemaJSON, _ := json.Marshal(schema)

	return &testTool{
		name:        "get_weather",
		description: "Get the current weather in a given location",
		schema:      schemaJSON,
	}
}

// tempDir creates a temporary directory for testing.
func tempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir
}
