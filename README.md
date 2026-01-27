# Iris

Iris is a Go SDK and CLI for building AI-powered applications and agent workflows. It provides a unified interface for working with large language models (LLMs), making it easy to integrate AI capabilities into your Go projects.

## Why Iris?

Building AI applications often requires:
- Managing multiple LLM provider APIs with different interfaces
- Handling streaming responses, retries, and error normalization
- Securely storing and managing API keys
- Creating reusable agent workflows

Iris solves these problems by providing:
- **Unified SDK**: A consistent Go API across providers (OpenAI, with more coming)
- **Fluent Builder Pattern**: Intuitive, chainable API for constructing requests
- **Built-in Streaming**: First-class support for streaming responses with proper channel handling
- **Secure Key Management**: Encrypted local storage for API keys
- **Agent Graphs**: A framework for building stateful, multi-step AI workflows
- **CLI Tool**: Quickly test models and manage projects from the command line

## Features

### SDK Features
- Fluent chat builder with `System()`, `User()`, `Assistant()`, `Temperature()`, `MaxTokens()`, and `Tools()`
- Non-streaming and streaming response modes
- Tool/function calling support
- Automatic retry with exponential backoff
- Telemetry hooks for observability
- Normalized error types across providers

### CLI Features
- `iris chat` - Send chat completions from the terminal
- `iris keys` - Securely manage API keys with AES-256-GCM encryption
- `iris init` - Scaffold new Iris projects
- `iris graph export` - Export agent graphs to Mermaid or JSON

### Agent Framework
- Directed graph-based workflow execution
- Stateful execution with shared state across nodes
- YAML-based graph definitions
- Visual export to Mermaid diagrams

## Installation

### SDK

```bash
go get github.com/erikhoward/iris/core
go get github.com/erikhoward/iris/providers
```

### CLI

```bash
go install github.com/erikhoward/iris/cli/cmd/iris@latest
```

## Quick Start

### Using the SDK

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/erikhoward/iris/core"
    "github.com/erikhoward/iris/providers/openai"
)

func main() {
    // Create a provider
    provider := openai.New(os.Getenv("OPENAI_API_KEY"))

    // Create a client
    client := core.NewClient(provider)

    // Send a chat request
    resp, err := client.Chat("gpt-4o").
        System("You are a helpful assistant.").
        User("What is the capital of France?").
        Temperature(0.7).
        GetResponse(context.Background())

    if err != nil {
        fmt.Fprintln(os.Stderr, "Error:", err)
        os.Exit(1)
    }

    fmt.Println(resp.Output)
    fmt.Printf("Tokens used: %d\n", resp.Usage.TotalTokens)
}
```

### Streaming Responses

```go
stream, err := client.Chat("gpt-4o").
    User("Write a short poem about Go.").
    Stream(context.Background())

if err != nil {
    log.Fatal(err)
}

// Print chunks as they arrive
for chunk := range stream.Ch {
    fmt.Print(chunk.Delta)
}
fmt.Println()

// Or use DrainStream to collect everything
resp, err := core.DrainStream(ctx, stream)
```

### Using Tools

```go
// Define a tool
weatherTool := mytools.NewWeatherTool()

resp, err := client.Chat("gpt-4o").
    User("What's the weather in San Francisco?").
    Tools(weatherTool).
    GetResponse(ctx)

if len(resp.ToolCalls) > 0 {
    // Handle tool calls
    for _, call := range resp.ToolCalls {
        fmt.Printf("Tool: %s, Args: %s\n", call.Name, call.Arguments)
    }
}
```

### Using the CLI

```bash
# Set up your API key (stored encrypted)
iris keys set openai

# Chat with a model
iris chat --provider openai --model gpt-4o --prompt "Hello, world!"

# Stream responses
iris chat --provider openai --model gpt-4o --prompt "Tell me a story" --stream

# Get JSON output
iris chat --provider openai --model gpt-4o --prompt "Hello" --json

# Initialize a new project
iris init myproject

# Export an agent graph to Mermaid
iris graph export agent.yaml --format mermaid
```

## Project Structure

```
iris/
├── core/           # Core SDK types and client
├── providers/      # LLM provider implementations
│   └── openai/     # OpenAI provider
├── tools/          # Tool/function calling framework
├── agents/         # Agent graph framework
│   └── graph/      # Graph execution engine
├── cli/            # Command-line interface
│   ├── cmd/iris/   # CLI entry point
│   ├── commands/   # CLI commands
│   ├── config/     # Configuration loading
│   └── keystore/   # Encrypted key storage
└── tests/          # Integration tests
```

## Configuration

Iris looks for configuration at `~/.iris/config.yaml`:

```yaml
default_provider: openai
default_model: gpt-4o

providers:
  openai:
    api_key_env: OPENAI_API_KEY
```

## Supported Providers

| Provider | Status | Features |
|----------|--------|----------|
| OpenAI | Supported | Chat, Streaming, Tools |
| Anthropic | Planned | - |
| Ollama | Planned | - |

## Agent Graphs

Define agent workflows in YAML:

```yaml
name: my-agent
entry: start

nodes:
  - id: start
    type: prompt
    config:
      template: "Process: {{.input}}"
  - id: process
    type: llm
    config:
      model: gpt-4o
  - id: output
    type: output

edges:
  - from: start
    to: process
  - from: process
    to: output
```

Export to Mermaid for visualization:

```bash
iris graph export agent.yaml
```

```mermaid
graph TD
    start[start]
    process[process]
    output[output]
    start --> process
    process --> output
```

## Development

### Prerequisites
- Go 1.21 or later

### Building

```bash
# Build everything
go build ./...

# Run tests
go test ./...

# Run integration tests (requires IRIS_OPENAI_API_KEY)
go test -tags=integration ./tests/integration/...
```

### Project Layout

Iris uses Go workspaces to manage multiple modules:

```bash
go work use ./core ./providers ./tools ./agents ./cli ./tests
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.
