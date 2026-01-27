# AGENTS.md — Iris Project Overview & Architecture

## 1. Project Overview

**Iris** is a Go-native framework and CLI for building, orchestrating, and deploying AI agents and multimodal workflows.  
It provides a **provider-agnostic SDK**, a **streaming-first execution model**, and a **typed, concurrent-safe API surface** for interacting with LLMs, tools, vector stores, and agents.

Iris is designed to be:

- **Production-grade**: stable APIs, telemetry hooks, retry policies, typed errors.
- **Go-first**: idiomatic concurrency, strong typing, minimal reflection.
- **Streaming-native**: first-class support for incremental output and SSE-style streams.
- **Provider-agnostic**: OpenAI, Anthropic, Ollama, and future providers via a unified contract.
- **Agent-oriented**: supports DAG-based workflows, MCP, A2A, and dynamic skills.
- **Enterprise-ready**: guardrails, secrets management, telemetry, config layering.

Primary use cases:

- Chatbots and copilots  
- Autonomous agents  
- RAG pipelines  
- Tool-augmented workflows  
- Multi-agent collaboration systems  

---

## 2. Design Principles

1. **Strong typing over reflection**  
   Prefer explicit Go structs and interfaces. Avoid dynamic maps where possible.

2. **Provider isolation**  
   All provider-specific logic lives in `providers/<name>`.  
   Core never imports provider packages directly.

3. **Streaming-first API**  
   All providers MUST support streaming if available.  
   Core APIs treat streaming as a first-class primitive (`ChatStream`).

4. **Concurrency safety**  
   `Client` is safe for concurrent use.  
   Builders are per-call and not shared across goroutines.

5. **Minimal magic**  
   No global singletons.  
   No hidden background goroutines.  
   No hard-coded logging frameworks.

6. **Protocol over platform**  
   MCP and A2A are first-class protocols.  
   Iris favors open protocols over closed abstractions.

7. **Pluggability**  
   Providers, vector stores, tools, guardrails, skills, and transports are all interface-driven.

---

## 3. High-Level Architecture

```
                        ┌────────────────────┐
                        │        CLI         │
                        │   (cobra-based)    │
                        └─────────┬──────────┘
                                  │
                                  ▼
                        ┌────────────────────┐
                        │        CORE        │
                        │ Client + Builders │
                        │ Streaming + Retry │
                        │ Telemetry + Errors│
                        └─────────┬──────────┘
                                  │
            ┌─────────────────────┼─────────────────────┐
            ▼                     ▼                     ▼
   ┌────────────────┐   ┌────────────────┐   ┌────────────────┐
   │   PROVIDERS    │   │     TOOLS       │   │    VECTORS     │
   │ openai         │   │ registry        │   │ qdrant         │
   │ anthropic      │   │ schemas         │   │ pgvector       │
   │ ollama         │   └────────────────┘   └────────────────┘
   └────────────────┘
            │
            ▼
   ┌────────────────────┐
   │     AGENTS         │
   │ DAG runner         │
   │ Router nodes       │
   │ Mermaid export     │
   └────────────────────┘

   ┌────────────────────┐    ┌────────────────────┐
   │        MCP         │    │        A2A         │
   │ Tool exposure      │    │ Cross-agent comms │
   │ HTTP / stdio       │    │ Discovery         │
   └────────────────────┘    └────────────────────┘

   ┌────────────────────┐
   │    GUARDRAILS      │
   │ Interceptors      │
   │ Policy engine     │
   │ Redaction         │
   └────────────────────┘

   ┌────────────────────┐
   │ CONFIG + SECRETS   │
   │ Layered config     │
   │ OS keychain        │
   │ Encrypted file     │
   └────────────────────┘
```

---

## 4. Repository Layout

```
iris/
  go.work
  go.work.sum

  core/
    client.go
    types.go
    streaming.go
    telemetry.go
    retry.go
    errors.go

  providers/
    provider.go
    openai/
    anthropic/
    ollama/

  tools/
    tool.go
    registry.go
    parser.go

  vectors/            # v0.2+
    vector.go
    qdrant/
    pgvector/
    chroma/

  agents/
    graph/
      graph.go
      node.go
      runner.go
      router.go
      mermaid.go

  mcp/                # v0.2+
    server.go
    client.go
    protocol.go
    transport_http.go
    transport_stdio.go

  a2a/                # v0.2+
    protocol.go
    router.go
    client.go
    server.go
    discovery.go

  guardrails/         # v0.2+
    policy.go
    engine.go
    interceptors.go
    redaction.go

  skills/             # v0.2+
    skill.go
    loader.go
    registry.go
    manifest.go

  config/             # v0.2+
    config.go
    loader.go
    store.go
    keychain.go
    file.go

  cli/
    main.go
    commands/
      root.go
      chat.go
      init.go
      keys.go
      graph.go

  examples/
    chat/
    agent/
    rag/
    a2a/
    mcp/
```

---

## 5. Core Execution Flow

### 5.1 Non-Streaming Chat

```
Client.Chat(model)
  → Builder.User/System/Tools
    → Validate request
      → Guardrails (pre)
        → Telemetry start
          → Retry loop
            → provider.Chat()
          → Telemetry end
        → Guardrails (post)
      → Return ChatResponse
```

---

### 5.2 Streaming Chat

```
Client.Chat(model).Stream(ctx)
  → Guardrails (pre)
    → provider.StreamChat()
      → ChatStream (Ch, Err, Final)
        → Deltas flow through Ch
        → Tool calls assembled by provider
        → Final ChatResponse sent on Final
    → Guardrails (post) [optional v0.2]
```

---

## 6. Major Architectural Decisions

### 6.1 Provider Interface as the Primary Abstraction

All inference flows through:

```go
type Provider interface {
  ID() string
  Models() []ModelInfo
  Supports(feature Feature) bool

  Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
  StreamChat(ctx context.Context, req *ChatRequest) (*ChatStream, error)
}
```

**Rationale**

- Prevents SDK lock-in.  
- Enables uniform retry, telemetry, and guardrails.  
- Makes adding providers a self-contained task.  

---

### 6.2 Streaming as a First-Class Primitive

Streaming returns a structured type:

```go
type ChatStream struct {
  Ch    <-chan ChatChunk
  Err   <-chan error
  Final <-chan *ChatResponse
}
```

**Rationale**

- Avoids hidden buffering.  
- Supports partial UI rendering.  
- Enables mid-stream policy enforcement.  

---

### 6.3 Typed Tool Calls

Tools are defined as:

```go
type Tool interface {
  Name() string
  Description() string
  Schema() ToolSchema
  Call(ctx context.Context, args json.RawMessage) (any, error)
}
```

With type-safe parsing:

```go
func ParseArgs[T any](call ToolCall) (*T, error)
```

**Rationale**

- No stringly-typed tool arguments.  
- Clean mapping across OpenAI, Anthropic, MCP.  

---

### 6.4 Minimal Agent Graph Runtime

Nodes implement:

```go
type Node interface {
  ID() string
  Run(ctx context.Context, state *State) (*State, error)
}
```

Graphs are explicit DAGs.

**Rationale**

- Deterministic execution.  
- Testable workflows.  
- No hidden schedulers or magic state.  

---

### 6.5 Protocol-First Extensibility (MCP + A2A)

- MCP exposes tools to external LLMs and runtimes.  
- A2A enables cross-agent delegation and collaboration.  

**Rationale**

- Avoid proprietary orchestration layers.  
- Enable distributed agent systems.  
- Interop with external runtimes.  

---

### 6.6 Layered Configuration + Secret References

Secrets are never stored inline.

```yaml
providers:
  openai:
    api_key_ref: openai_key
```

Resolved via:

```go
secrets.Get(ctx, "openai_key")
```

**Rationale**

- Prevents secret leakage.  
- Enables secure CLI + SDK + server deployments.  
- Supports enterprise key management.  

---

### 6.7 Guardrails as Interceptors

Guardrails run before and after provider calls.

```text
PreRequest → Provider → PostResponse
```

**Rationale**

- Provider-agnostic enforcement.  
- Supports block / warn / monitor modes.  
- Streaming-compatible.  

---

## 7. Versioning Strategy

- v0.1: Core SDK, OpenAI/Anthropic/Ollama, CLI, minimal agents.  
- v0.2: MCP, A2A, Guardrails, Vectors, Skills, Config.  
- v0.3+: Enterprise auth, plugin ABI stability, cloud-native orchestration.  

---

## 8. Stability Guarantees

- Public APIs in `core/` are stable within a minor version.  
- Provider adapters may evolve but must preserve interface contracts.  
- MCP and A2A versions are embedded in message envelopes.  

---

## 9. Contribution Guidelines (High-Level)

- No breaking changes to `core.Provider` without major version bump.  
- All new providers must include:
  - Unit tests  
  - Integration tests (optional)  
  - Feature matrix updates  
- No global state.  
- All exported types require GoDoc comments.  

---

## 10. Intended Audience

This file is written for:

- AI coding agents implementing Iris  
- Human maintainers  
- Provider adapter authors  
- Plugin / skill authors  
- Enterprise integrators  

---

## 13. Concurrent Sessions (MANDATORY)

Multiple Claude sessions may be working on the same branch or worktree simultaneously. To avoid destroying other sessions' work, the following rules are **mandatory**.

### 13.1 File Safety Rules

* ALWAYS create a new feature branch before starting to work on a new feature or bug-fix.
* NEVER delete, reset, or discard files you did not create.
* NEVER run `git reset`, `git checkout --`, or `git clean` without explicit user approval.
* ALWAYS ask the user before removing untracked files.
* When you see unfamiliar files, assume another session created them and ask the user what to do.
* If pre-commit hooks fail due to files you did not touch, ask the user how to proceed.

**Why this matters**

The user may have multiple Claude sessions working in parallel on different aspects of a feature. Deleting unknown files destroys that work.

---

## 13.2 Feature/Bugfix Workflow (MANDATORY)

When working on a feature or bug fix, ALWAYS follow this workflow.

### 13.3 Create a Feature/Fix Branch

```
git checkout main
git pull origin main
git checkout -b <type>/<short-description>

# Example:
# feat/42-add-new-embedder
```

---

## 13.4 Task Workflow (MANDATORY)

When working on tasks, ALWAYS follow this workflow.

### 13.5 Pre-Flight Checks (MANDATORY)

* The task is identified by a file under `.spec/tasks`
* Verify the task has not been completed by looking at completed tasks under `.spec/completed`
* The agent has read the ENTIRE task file
* Clean working tree (no accidental edits). STOP if not clean
* Install dependencies

### 13.6 Task Completion (MANDATORY)

Every completed task must be documented in a file under `.spec/completed`.

Use the following template to document task completion

```markdown
## ✅ [TASK-ID] - [TASK NAME]
**Completed On**: YYYY-mm-dd
**Completed By**: [person or agent]
**Source Task**: `.spec/tasks/IRIS-T0021-ollama-provider.md`
**Related PR/Commit**: [COMMIT or PR]

### Scope

Implemented Ollma provider support...

**Backend Changes**:

* List of all backend changes

**Frontend Changes**:

* List of all frontend changes

### Verification

Commands executed to verify feature acceptance

### Deviations

* User-approved support for optional provider implementation

### Files Created:

* src/internal/provider/ollama.go

### Files Modified:

* src/internal/core/types.go - Added 1 provider enum and 3 model enums
```

---

## 14. Git Rules (MANDATORY)

### 14.1 Commit Content Rules

* NEVER commit: todos, research notes, scratch files.
* ALWAYS commit: code, tests, requested documentation, schemas.
* Update `.gitignore` for file patterns only.

---

### 14.2 Commit Message Rules

* ALWAYS use conventional commits.
* Subject line MUST be 72 characters or fewer.
* No trailing period.
* No markdown, emojis, or backticks.
* All quotes and brackets MUST be closed.
* Never output unterminated strings.
* One logical change per commit.
* If any rule is violated, rewrite the message before output.

---

### 14.3 Destructive Command Rules

NEVER run destructive git commands without explicit user confirmation:

* `git reset HEAD` or `git reset --hard`
* `git checkout HEAD -- .` or `git checkout -- .`
* `git clean -fd`
* `git stash drop`

---

### 14.4 Path Handling in Tests (MANDATORY)

Windows and Unix use different path separators. Hardcoded paths will fail on Windows CI.

Rules:

* NEVER use forward-slash concatenation:

  ```go
  tempDir + "/components/terraform/vpc"
  ```

* ALWAYS use `filepath.Join()` with separate arguments:

  ```go
  filepath.Join(tempDir, "components", "terraform", "vpc")
  ```

* NEVER use forward slashes inside `filepath.Join()`:

  ```go
  filepath.Join(dir, "a/b/c") // WRONG
  filepath.Join(dir, "a", "b", "c") // CORRECT
  ```

* For suffix checks, normalize paths first:

  ```go
  strings.HasSuffix(filepath.ToSlash(path), "expected/suffix")
  ```

---

## 15. Issue Workflow (MANDATORY)

When working on a GitHub issue, ALWAYS follow this workflow.

### 15.1 Create a Feature Branch

```
git checkout main
git pull origin main
git checkout -b <type>/<issue-number>-<short-description>

# Example:
# feat/42-add-new-embedder
```

---

### 15.2 Implement and Commit

* Implement the changes.
* Commit following the commit conventions above.

---

### 15.3 Push and Create a Pull Request

* DO NOT include a generated by claude statement.

```
git push -u origin <branch-name>

gh pr create \
  --title "<type>(scope): description" \
  --body "Closes #<issue-number>"
```

---

### 15.4 Before Merging

Before merging, ensure:

* CI pipeline passes (all checks green).
* Code has been reviewed if required.
* No merge conflicts with `main`.

---

### 15.5 Main Branch Protection

* NEVER push directly to `main`.
* ALWAYS use feature branches and pull requests.

---

## 16. Summary

Iris is:

- A Go-native, production-grade AI agent framework.  
- Streaming-first and provider-agnostic.  
- Protocol-driven (MCP + A2A).  
- Enterprise-ready (guardrails, secrets, telemetry).  
- Designed for long-term extensibility.  

This document is the canonical architectural guide for all Iris development.
