# Design: Marketing-Focused README and GitHub Metadata for kimi-wire

## Status
Approved by user on 2026-06-10.

## Goal
Make the project landing page (README) more compelling for Go developers who want to integrate with the Kimi Code CLI Wire protocol, and add GitHub repository metadata (description + topics) for discoverability.

## Scope
1. Rewrite `README.md` with a marketing tone while preserving all existing technical content.
2. Create `.github/repo-meta.yml` containing repository description and topics.
3. Verify no tests or links are broken after the changes.

## README Structure

### 1. Hero Section
- Title: `# kimi-wire`
- Tagline: *Typed Go client for the Kimi Code CLI Wire protocol.*
- Badges (single row):
  - CI
  - Go Report Card
  - Go Version (1.22+)
  - Coverage
  - License MIT
  - Latest Release
  - GoDoc / pkg.go.dev

### 2. Why
Four pain/solution pairs:
1. Manual JSON-RPC serialization is error-prone → strongly typed `Event`, `Request`, `PromptResult`.
2. Wire logs can leak API keys → built-in secret redaction.
3. Integration tests against a child process are slow and brittle → `ChannelTransport` and `InMemoryTransport` out of the box.
4. Need to support CLI, embedding, and testing scenarios → pluggable `Transport` abstraction.

### 3. Features
- Strongly typed protocol structs
- High-level client (`Prompt`, `Replay`, `Steer`, `SetPlanMode`, `Cancel`, `Initialize`)
- Pluggable transports (stdio, in-memory, custom)
- Built-in secret redaction
- JSON-RPC 2.0 compliant
- Idiomatic Go errors compatible with `errors.As` / `errors.Is`

### 4. Installation
Keep existing `go get` command.

### 5. Quick Start
Keep existing runnable example, tighten formatting.

### 6. Transport Implementations
Keep existing `ChildProcessTransport`, `ChannelTransport`, `InMemoryTransport` content with clearer subheadings.

### 7. Protocol Types
Keep existing `Events`, `Requests`, `UserInput`/`ToolOutput` content.

### 8. Secret Redaction
Keep existing content and covered patterns list.

### 9. Error Handling
Keep existing `*WireError` and `errors.As` example.

### 10. Testing
Keep existing test commands.

### 11. License
MIT.

## GitHub Metadata

File: `.github/repo-meta.yml`

```yaml
description: "Typed Go client for the Kimi Code CLI Wire protocol. JSON-RPC 2.0, secret redaction, pluggable transports."
topics:
  - kimi
  - ai-agent
  - llm
  - agent-sdk
  - model-context-protocol
  - mcp
  - wire-protocol
  - cli-automation
  - go
  - golang
  - sdk
  - client
  - json-rpc
```

## Non-Goals
- Change Go source code or public API.
- Add new GitHub Actions workflows.
- Modify `go.mod`.

## Verification
- `go test ./...` must pass.
- README markdown must render without broken badge links.
- All existing technical examples must remain copy-pasteable.

## Implementation Plan
1. Delegate README rewrite to a coder subagent.
2. Delegate `.github/repo-meta.yml` creation to a coder subagent.
3. Review both outputs and run `go test ./...`.
