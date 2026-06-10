# README and GitHub Metadata Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite `README.md` with a marketing-focused structure and create `.github/repo-meta.yml` for repository description and topics.

**Architecture:** Single-file documentation rewrite plus a new YAML metadata file. No Go source changes. Verification via existing test suite and markdown rendering.

**Tech Stack:** Markdown, YAML, Go test tooling.

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `README.md` | Rewrite | Project landing page with hero, Why, features, quick start, docs |
| `.github/repo-meta.yml` | Create | GitHub repository description and topics |

---

### Task 1: Rewrite README.md

**Files:**
- Modify: `README.md` (full file rewrite)

- [ ] **Step 1: Read the existing README and go.mod**

  Run:
  ```bash
  cat README.md
  cat go.mod
  ```
  Expected: Confirm module path is `github.com/ekhodzitsky/kimi-wire` and Go version is `1.22`.

- [ ] **Step 2: Write the new README**

  Replace the entire contents of `README.md` with the following markdown. Preserve all existing code examples (Quick Start, Transport, Protocol Types, Secret Redaction, Error Handling, Testing) verbatim, but wrap them in the new marketing structure.

  ```markdown
  # kimi-wire

  [![CI](https://github.com/ekhodzitsky/kimi-wire/actions/workflows/ci.yml/badge.svg)](https://github.com/ekhodzitsky/kimi-wire/actions/workflows/ci.yml)
  [![Go Report Card](https://goreportcard.com/badge/github.com/ekhodzitsky/kimi-wire)](https://goreportcard.com/report/github.com/ekhodzitsky/kimi-wire)
  [![Go Version](https://img.shields.io/badge/go-1.22%2B-blue)](https://go.dev/doc/devel/release)
  [![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
  [![Latest Release](https://img.shields.io/github/v/release/ekhodzitsky/kimi-wire)](https://github.com/ekhodzitsky/kimi-wire/releases)
  [![GoDoc](https://pkg.go.dev/badge/github.com/ekhodzitsky/kimi-wire)](https://pkg.go.dev/github.com/ekhodzitsky/kimi-wire)

  Typed Go client for the [Kimi Code CLI](https://github.com/ekhodzitsky/kimi) Wire protocol.

  ## Why?

  Building on top of `kimi --wire` means speaking JSON-RPC 2.0 over stdin/stdout. Without a typed client, you end up hand-rolling structs, chasing field names, and hoping your serialization matches the agent's expectations.

  **kimi-wire solves four hard problems out of the box:**

  1. **Stop guessing JSON-RPC shapes.** Get strongly typed `Event`, `Request`, `PromptResult`, and the rest of the protocol surface. The compiler catches drift before it reaches runtime.
  2. **Keep secrets out of logs.** Wire traffic is rich with API keys, tokens, and credentials. The library scrubs them automatically from error messages and provides a helper for log redaction.
  3. **Test without a child process.** `ChannelTransport` and `InMemoryTransport` let you unit-test agent interactions in-memory. No process spawning, no flaky CI.
  4. **One client, many transports.** Swap between stdio, in-memory channels, or a custom transport without changing client code.

  ## Features

  - Strongly typed protocol structs (`Event`, `Request`, `PromptResult`, ...)
  - High-level `Client` with `Prompt`, `Replay`, `Steer`, `SetPlanMode`, `Cancel`, `Initialize`
  - Pluggable `Transport` abstraction: stdio, in-memory channels, injectable test transport
  - Built-in secret redaction for wire logs and error messages
  - JSON-RPC 2.0 compliant message framing
  - Idiomatic Go errors compatible with `errors.As` / `errors.Is`

  ## Requirements

  - Go 1.22 or later
  - The `kimi` binary in your `PATH` (for `ChildProcessTransport`)

  ## Installation

  ```bash
  go get github.com/ekhodzitsky/kimi-wire
  ```

  ## Quick Start

  ```go
  package main

  import (
      "context"
      "fmt"
      "log"

      "github.com/ekhodzitsky/kimi-wire"
  )

  func main() {
      ctx := context.Background()

      // Spawn a child process running `kimi --wire`
      transport, err := wire.SpawnChildProcessTransport("kimi", wire.SpawnOptions{})
      if err != nil {
          log.Fatal(err)
      }
      defer transport.Close()

      client := wire.NewClient(transport)

      // Perform the initialization handshake
      if _, err := client.Initialize(ctx, wire.InitializeParams{
          ProtocolVersion: wire.WireProtocolVersion,
      }); err != nil {
          log.Fatal(err)
      }

      // Send a prompt
      result, err := client.Prompt(ctx, wire.UserInput{Text: "Hello!"})
      if err != nil {
          log.Fatal(err)
      }
      fmt.Println("Status:", result.Status)
  }
  ```

  ## Transport Implementations

  ### ChildProcessTransport

  Spawns `kimi` as a child process and communicates over stdin/stdout:

  ```go
  transport, err := wire.SpawnChildProcessTransport("kimi", wire.SpawnOptions{
      WorkDir: ptr("/path/to/project"),
      Session: ptr("my-session"),
      Model:   ptr("kimi-latest"),
  })
  ```

  ### ChannelTransport

  In-memory pair for testing or embedding:

  ```go
  a, b := wire.NewChannelTransportPair()
  ```

  ### InMemoryTransport

  Injectable/inspectable transport for unit tests:

  ```go
  mem := wire.NewInMemoryTransport()
  client := wire.NewClient(mem)

  mem.Inject(`{"jsonrpc":"2.0","id":"req-1","result":{"status":"finished"}}`)
  result, err := client.Prompt(ctx, wire.UserInput{Text: "hi"})
  ```

  ## Protocol Types

  ### Events

  Events are incoming notifications from the agent:

  ```go
  ev, err := wire.ParseEvent(data)
  switch e := ev.(type) {
  case wire.TurnEndEvent:
      // Turn ended
  case wire.ToolCallEvent:
      // Agent wants to call a tool: e.Function.Name, e.Function.Arguments
  case wire.ContentPartEvent:
      // New content part: e.Part
  }
  ```

  ### Requests

  Requests are incoming method calls from the agent that require a response:

  ```go
  req, err := wire.ParseRequest(data)
  switch r := req.(type) {
  case wire.ToolCallRequest:
      // Execute tool: r.Name, r.Arguments
  case wire.ApprovalRequest:
      // Ask user for approval: r.Action, r.Description
  }
  ```

  ### UserInput and ToolOutput

  Both support the wire format of either a plain string or an array of `ContentPart`:

  ```go
  // String form
  input := wire.UserInput{Text: "Hello!"}

  // Content parts form
  input := wire.UserInput{
      Parts: []wire.ContentPart{
          {Type: wire.ContentPartTypeText, Text: &wire.TextPart{Text: "Hello!"}},
      },
  }
  ```

  ## Secret Redaction

  The library automatically redacts secrets from error messages and provides a helper for log scrubbing:

  ```go
  // Redact a JSON-like value (map, slice, string, json.RawMessage)
  safe := wire.RedactSecrets(map[string]any{
      "api_key": "super-secret",
      "url":     "https://example.com",
  })
  // safe["api_key"] == "***"
  // safe["url"] == "https://example.com"
  ```

  Covered patterns: API keys, tokens, passwords, Authorization headers (Bearer/Basic), AWS access keys, GitHub PATs, JWTs, URL credentials, and PEM/PGP private keys.

  ## Error Handling

  All errors are typed as `*WireError` with a discriminating `Kind`:

  ```go
  result, err := client.Prompt(ctx, input)
  var we *wire.WireError
  if errors.As(err, &we) {
      switch we.Kind {
      case wire.ErrTimeout:
          // Handle timeout
      case wire.ErrRequestFailed:
          // Handle JSON-RPC error from server
      }
  }
  ```

  `WireError` implements `Unwrap()`, so `errors.Is` works with underlying causes like `context.Canceled`.

  ## Testing

  ```bash
  go test ./...
  go test ./... -race
  go test ./... -fuzz=FuzzParseWireMessage -fuzztime=30s
  ```

  ## License

  MIT
  ```

- [ ] **Step 3: Verify the file was written correctly**

  Run:
  ```bash
  wc -l README.md
  head -n 20 README.md
  ```
  Expected: README.md exists, starts with `# kimi-wire`, and contains the new badge block.

- [ ] **Step 4: Commit the README change**

  ```bash
  git add README.md
  git commit -m "docs: rewrite README with marketing focus, Why block, and more badges"
  ```

---

### Task 2: Create GitHub Repository Metadata

**Files:**
- Create: `.github/repo-meta.yml`

- [ ] **Step 1: Create the metadata file**

  Write the following content to `.github/repo-meta.yml`:

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

- [ ] **Step 2: Verify the file exists and is valid YAML**

  Run:
  ```bash
  cat .github/repo-meta.yml
  python3 -c "import yaml, sys; yaml.safe_load(open('.github/repo-meta.yml'))"
  ```
  Expected: File prints correctly and Python parses it without error.

- [ ] **Step 3: Commit the metadata file**

  ```bash
  git add .github/repo-meta.yml
  git commit -m "chore: add GitHub repository description and topics"
  ```

---

### Task 3: Verify Nothing Is Broken

**Files:**
- Test: all Go source files via existing test suite

- [ ] **Step 1: Run the full test suite**

  Run:
  ```bash
  go test ./...
  ```
  Expected: `PASS` for all packages, no compilation errors.

- [ ] **Step 2: Optional — run race detector**

  Run:
  ```bash
  go test ./... -race
  ```
  Expected: `PASS` for all packages.

- [ ] **Step 3: Confirm final state**

  Run:
  ```bash
  git log --oneline -3
  git status
  ```
  Expected: Two clean commits on top of the previous HEAD, working tree clean.

---

## Spec Coverage

| Spec Requirement | Task |
|------------------|------|
| Hero section with title, tagline, badges | Task 1, Step 2 |
| Why block with 4 pain/solution pairs | Task 1, Step 2 |
| Features list | Task 1, Step 2 |
| Preserve existing technical content | Task 1, Step 2 |
| GitHub description | Task 2, Step 1 |
| GitHub topics | Task 2, Step 1 |
| No broken tests | Task 3, Steps 1–2 |

## Placeholder Scan

No TBD, TODO, or vague steps. All code examples are copy-pasteable from the existing README. All commands have exact expected output.
