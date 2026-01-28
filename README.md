# Iris
[![Build Status](https://github.com//erikhoward/iris/actions/workflows/ci.yml/badge.svg)](https://github.com/erikhoward/iris/actions/workflows/ci.yml)&nbsp;
[![Go Report Card](https://goreportcard.com/badge/github.com/erikhoward/iris?style=flat)](https://goreportcard.com/report/github.com/erikhoward/iris)&nbsp;
[![GoDoc](https://godoc.org/github.com/erikhoward/iris?status.svg)](https://godoc.org/github.com/erikhoward/iris)&nbsp;
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/erikhoward/iris/blob/main/LICENSE)

Iris is a Go SDK and CLI for building AI-powered applications and agent workflows. It provides a unified interface for working with large language models (LLMs), making it easy to integrate AI capabilities into your Go projects.

## Why Iris?

Building AI applications often requires:
- Managing multiple LLM provider APIs with different interfaces
- Handling streaming responses, retries, and error normalization
- Securely storing and managing API keys
- Creating reusable agent workflows

Iris solves these problems by providing:
- **Unified SDK**: A consistent Go API across providers (OpenAI, Anthropic, Google Gemini, xAI Grok, Z.ai GLM, Ollama)
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
- **Responses API support** for GPT-5+ models with reasoning, built-in tools (web search, code interpreter), and response chaining
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
go get github.com/erikhoward/iris
```

### CLI

```bash
go install github.com/erikhoward/iris/cli/cmd/iris@v0.1.0
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

### Using Anthropic Claude

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/erikhoward/iris/core"
    "github.com/erikhoward/iris/providers/anthropic"
)

func main() {
    // Create an Anthropic provider
    provider := anthropic.New(os.Getenv("ANTHROPIC_API_KEY"))

    // Create a client
    client := core.NewClient(provider)

    // Send a chat request
    resp, err := client.Chat("claude-sonnet-4-5").
        System("You are a helpful assistant.").
        User("What is the capital of France?").
        GetResponse(context.Background())

    if err != nil {
        fmt.Fprintln(os.Stderr, "Error:", err)
        os.Exit(1)
    }

    fmt.Println(resp.Output)
}
```

### Using Google Gemini

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/erikhoward/iris/core"
    "github.com/erikhoward/iris/providers/gemini"
)

func main() {
    // Create a Gemini provider
    provider := gemini.New(os.Getenv("GEMINI_API_KEY"))

    // Create a client
    client := core.NewClient(provider)

    // Send a chat request
    resp, err := client.Chat("gemini-2.5-flash").
        System("You are a helpful assistant.").
        User("What is the capital of France?").
        GetResponse(context.Background())

    if err != nil {
        fmt.Fprintln(os.Stderr, "Error:", err)
        os.Exit(1)
    }

    fmt.Println(resp.Output)
}
```

Gemini models with thinking/reasoning support:

```go
// Use reasoning with Gemini 2.5 models (budget-based)
resp, err := client.Chat("gemini-2.5-pro").
    User("Solve this complex problem step by step").
    ReasoningEffort(core.ReasoningEffortHigh).
    GetResponse(ctx)

// Access reasoning if available
if resp.Reasoning != nil && resp.Reasoning.Output != "" {
    fmt.Println("Thinking:", resp.Reasoning.Output)
}
```

### Using xAI Grok

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/erikhoward/iris/core"
    "github.com/erikhoward/iris/providers/xai"
)

func main() {
    // Create an xAI provider
    provider := xai.New(os.Getenv("XAI_API_KEY"))

    // Create a client
    client := core.NewClient(provider)

    // Send a chat request using Grok 4
    resp, err := client.Chat(xai.ModelGrok4).
        System("You are a helpful assistant.").
        User("What is the capital of France?").
        GetResponse(context.Background())

    if err != nil {
        fmt.Fprintln(os.Stderr, "Error:", err)
        os.Exit(1)
    }

    fmt.Println(resp.Output)
}
```

xAI Grok models with reasoning support:

```go
// Use reasoning with grok-3-mini (only model that exposes reasoning_content)
resp, err := client.Chat(xai.ModelGrok3Mini).
    User("Solve this step by step: If I have 5 apples and give away half...").
    ReasoningEffort(core.ReasoningEffortHigh).
    GetResponse(ctx)

// Access reasoning if available (grok-3-mini only)
if resp.Reasoning != nil && len(resp.Reasoning.Summary) > 0 {
    fmt.Println("Thinking:", resp.Reasoning.Summary[0])
}
```

### Using Z.ai GLM

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/erikhoward/iris/core"
    "github.com/erikhoward/iris/providers/zai"
)

func main() {
    // Create a Z.ai provider
    provider := zai.New(os.Getenv("ZAI_API_KEY"))

    // Create a client
    client := core.NewClient(provider)

    // Send a chat request using GLM-4.7
    resp, err := client.Chat(zai.ModelGLM47).
        System("You are a helpful assistant.").
        User("What is the capital of France?").
        GetResponse(context.Background())

    if err != nil {
        fmt.Fprintln(os.Stderr, "Error:", err)
        os.Exit(1)
    }

    fmt.Println(resp.Output)
}
```

Z.ai GLM models with thinking support:

```go
// Use thinking mode with GLM-4.7 (enabled by default)
resp, err := client.Chat(zai.ModelGLM47).
    User("Solve this step by step: What is 15% of 240?").
    ReasoningEffort(core.ReasoningEffortHigh).
    GetResponse(ctx)

// Access reasoning if available
if resp.Reasoning != nil && len(resp.Reasoning.Summary) > 0 {
    fmt.Println("Thinking:", resp.Reasoning.Summary[0])
}
```

### Using Ollama

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/erikhoward/iris/core"
    "github.com/erikhoward/iris/providers/ollama"
)

func main() {
    // Create a local Ollama provider (no API key needed)
    provider := ollama.New()

    // Or connect to a remote Ollama instance:
    // provider := ollama.New(ollama.WithBaseURL("http://remote-host:11434"))

    // Or use Ollama Cloud:
    // provider := ollama.New(
    //     ollama.WithCloud(),
    //     ollama.WithAPIKey(os.Getenv("OLLAMA_API_KEY")),
    // )

    // Create a client
    client := core.NewClient(provider)

    // Send a chat request - use any model you have pulled
    resp, err := client.Chat("llama3.2").
        System("You are a helpful assistant.").
        User("What is the capital of France?").
        GetResponse(context.Background())

    if err != nil {
        fmt.Fprintln(os.Stderr, "Error:", err)
        os.Exit(1)
    }

    fmt.Println(resp.Output)
}
```

Ollama models with thinking support:

```go
// Use thinking with models like qwen3
resp, err := client.Chat("qwen3").
    User("Solve this step by step: What is 15% of 240?").
    ReasoningEffort(core.ReasoningEffortHigh).
    GetResponse(ctx)

// Access reasoning if available
if resp.Reasoning != nil && len(resp.Reasoning.Summary) > 0 {
    fmt.Println("Thinking:", resp.Reasoning.Summary[0])
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

### Using the Responses API (GPT-5)

GPT-5 models automatically use OpenAI's Responses API, which provides advanced features like reasoning, built-in tools, and response chaining.

```go
// GPT-5 uses the Responses API automatically
resp, err := client.Chat("gpt-5").
    Instructions("You are a helpful research assistant.").
    User("What are the latest developments in quantum computing?").
    ReasoningEffort(core.ReasoningEffortHigh).
    WebSearch().
    GetResponse(ctx)

if err != nil {
    log.Fatal(err)
}

fmt.Println(resp.Output)

// Access reasoning summary if available
if resp.Reasoning != nil {
    for _, summary := range resp.Reasoning.Summary {
        fmt.Println("Reasoning:", summary)
    }
}

// Response chaining - continue from a previous response
followUp, err := client.Chat("gpt-5").
    ContinueFrom(resp.ID).
    User("Can you elaborate on the most promising approach?").
    GetResponse(ctx)
```

### Using the CLI

```bash
# Set up your API key (stored encrypted)
iris keys set openai
iris keys set anthropic
iris keys set gemini
iris keys set xai
iris keys set zai
iris keys set ollama  # Only needed for Ollama Cloud

# Chat with OpenAI
iris chat --provider openai --model gpt-4o --prompt "Hello, world!"

# Chat with Anthropic Claude
iris chat --provider anthropic --model claude-sonnet-4-5 --prompt "Hello, world!"

# Chat with Google Gemini
iris chat --provider gemini --model gemini-2.5-flash --prompt "Hello, world!"

# Chat with xAI Grok
iris chat --provider xai --model grok-4 --prompt "Hello, world!"

# Chat with Z.ai GLM
iris chat --provider zai --model glm-4.7-flash --prompt "Hello, world!"

# Chat with local Ollama (no API key needed)
iris chat --provider ollama --model llama3.2 --prompt "Hello, world!"

# Chat with GPT-5 (uses Responses API automatically)
iris chat --provider openai --model gpt-5 --prompt "Explain quantum entanglement"

# Stream responses
iris chat --provider openai --model gpt-4o --prompt "Tell me a story" --stream
iris chat --provider anthropic --model claude-sonnet-4-5 --prompt "Tell me a story" --stream

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
│   ├── openai/     # OpenAI provider
│   ├── anthropic/  # Anthropic Claude provider
│   ├── gemini/     # Google Gemini provider
│   ├── xai/        # xAI Grok provider
│   ├── zai/        # Z.ai GLM provider
│   └── ollama/     # Ollama provider (local and cloud)
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
default_model: gpt-5  # or gpt-4o for older models

providers:
  openai:
    api_key_env: OPENAI_API_KEY
  anthropic:
    api_key_env: ANTHROPIC_API_KEY
  gemini:
    api_key_env: GEMINI_API_KEY
  xai:
    api_key_env: XAI_API_KEY
  zai:
    api_key_env: ZAI_API_KEY
  ollama:
    # For local Ollama, no API key needed
    # For Ollama Cloud, set api_key_env: OLLAMA_API_KEY
    # Custom base URL: base_url: http://localhost:11434
```

## Supported Providers

| Provider | Status | Features |
|----------|--------|----------|
| OpenAI | Supported | Chat, Streaming, Tools, Responses API (GPT-5+) |
| Anthropic | Supported | Chat, Streaming, Tools |
| Google Gemini | Supported | Chat, Streaming, Tools, Reasoning |
| xAI Grok | Supported | Chat, Streaming, Tools, Reasoning |
| Z.ai GLM | Supported | Chat, Streaming, Tools, Thinking |
| Ollama | Supported | Chat, Streaming, Tools, Thinking |

### xAI Grok Models

| Model ID | Features |
|----------|----------|
| `grok-3` | Chat, Streaming, Tools, Reasoning |
| `grok-3-mini` | Chat, Streaming, Tools, Reasoning (exposes reasoning_content) |
| `grok-4` | Chat, Streaming, Tools, Reasoning (latest) |
| `grok-4-fast-non-reasoning` | Chat, Streaming, Tools |
| `grok-4-fast-reasoning` | Chat, Streaming, Tools, Reasoning |
| `grok-code-fast` | Chat, Streaming, Tools (code-optimized) |
| `grok-4-1-fast-non-reasoning` | Chat, Streaming, Tools (default for CLI) |
| `grok-4-1-fast-reasoning` | Chat, Streaming, Tools, Reasoning |

### Z.ai GLM Models

| Model ID | Features |
|----------|----------|
| `glm-4.7` | Chat, Streaming, Tools, Thinking (latest flagship) |
| `glm-4.7-flash` | Chat, Streaming, Tools (default for CLI) |
| `glm-4.7-flashx` | Chat, Streaming, Tools |
| `glm-4.6` | Chat, Streaming, Tools, Thinking |
| `glm-4.6v` | Chat, Streaming, Tools, Thinking, Vision |
| `glm-4.6v-flash` | Chat, Streaming, Tools, Vision |
| `glm-4.6v-flashx` | Chat, Streaming, Tools, Vision |
| `glm-4.5` | Chat, Streaming, Tools, Thinking |
| `glm-4.5v` | Chat, Streaming, Tools, Thinking, Vision |
| `glm-4.5-x` | Chat, Streaming, Tools |
| `glm-4.5-air` | Chat, Streaming, Tools |
| `glm-4.5-airx` | Chat, Streaming, Tools |
| `glm-4.5-flash` | Chat, Streaming, Tools |
| `glm-4-32b-0414-128k` | Chat, Streaming, Tools (128K context) |

### Gemini Models

| Model ID | Features |
|----------|----------|
| `gemini-3-pro-preview` | Chat, Streaming, Tools, Reasoning (thinkingLevel) |
| `gemini-3-flash-preview` | Chat, Streaming, Tools, Reasoning (thinkingLevel) |
| `gemini-2.5-pro` | Chat, Streaming, Tools, Reasoning (thinkingBudget) |
| `gemini-2.5-flash` | Chat, Streaming, Tools, Reasoning (thinkingBudget) |
| `gemini-2.5-flash-lite` | Chat, Streaming, Tools, Reasoning (thinkingBudget) |

### Ollama Models

Ollama supports any model you have pulled locally. Use `ollama pull <model>` to download models.

| Model ID | Features |
|----------|----------|
| `llama3.2` | Chat, Streaming, Tools |
| `llama3.2:70b` | Chat, Streaming, Tools |
| `mistral` | Chat, Streaming, Tools |
| `mixtral` | Chat, Streaming, Tools |
| `qwen3` | Chat, Streaming, Tools, Thinking |
| `gemma3` | Chat, Streaming |
| `deepseek-coder` | Chat, Streaming |
| `codellama` | Chat, Streaming |

See https://ollama.com/library for all available models.

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
      model: gpt-5
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
- Go 1.24 or later

### Building

```bash
# Build everything
go build ./...

# Run tests
go test ./...

# Run integration tests (requires IRIS_OPENAI_API_KEY)
go test -tags=integration ./tests/integration/...
```

### Module Structure

Iris uses a single Go module at the repository root. All packages are imported from `github.com/erikhoward/iris/*`:

```go
import (
    "github.com/erikhoward/iris/core"
    "github.com/erikhoward/iris/providers/openai"
    "github.com/erikhoward/iris/providers/anthropic"
    "github.com/erikhoward/iris/providers/gemini"
    "github.com/erikhoward/iris/providers/xai"
    "github.com/erikhoward/iris/providers/zai"
    "github.com/erikhoward/iris/providers/ollama"
    "github.com/erikhoward/iris/tools"
    "github.com/erikhoward/iris/agents/graph"
)
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.
