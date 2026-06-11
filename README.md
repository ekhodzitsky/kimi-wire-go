# kimi-wire

[![CI](https://github.com/ekhodzitsky/kimi-wire-go/actions/workflows/ci.yml/badge.svg)](https://github.com/ekhodzitsky/kimi-wire-go/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ekhodzitsky/kimi-wire-go)](https://goreportcard.com/report/github.com/ekhodzitsky/kimi-wire-go)
[![Go Version](https://img.shields.io/badge/go-1.22%2B-blue)](https://go.dev/doc/devel/release)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Latest Release](https://img.shields.io/github/v/release/ekhodzitsky/kimi-wire-go)](https://github.com/ekhodzitsky/kimi-wire-go/releases)
[![GoDoc](https://pkg.go.dev/badge/github.com/ekhodzitsky/kimi-wire)](https://pkg.go.dev/github.com/ekhodzitsky/kimi-wire)

Typed Go client for the [Kimi Code CLI](https://github.com/ekhodzitsky/kimi) Wire protocol.

> **Current version:** latest release is [`v0.2.0`](https://github.com/ekhodzitsky/kimi-wire-go/releases/tag/v0.2.0).

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
- Pluggable `Transport` abstraction: stdio, in-memory channels, custom transport
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

## Server side

The same package also exposes a Wire protocol server so any Go binary can act as an agent backend.

### Agent and Turn interfaces

A server is constructed from a `Transport` and an `Agent`. During a prompt turn the agent receives a `Turn` value that can emit events and make server-to-client requests:

```go
type Agent interface {
    Prompt(ctx context.Context, input wire.UserInput, turn wire.Turn) (wire.PromptResult, error)
}

type Turn interface {
    Emit(ctx context.Context, event wire.Event) error
    RequestApproval(ctx context.Context, req wire.ApprovalRequest) (wire.ApprovalResponse, error)
    CallExternalTool(ctx context.Context, req wire.ToolCallRequest) (wire.ToolCallResponse, error)
    AskQuestion(ctx context.Context, req wire.QuestionRequest) (wire.QuestionResponse, error)
    TriggerHook(ctx context.Context, req wire.HookRequest) (wire.HookResponse, error)
}
```

### Minimal echo server

```go
type echoAgent struct{}

func (a *echoAgent) Prompt(ctx context.Context, input wire.UserInput, turn wire.Turn) (wire.PromptResult, error) {
    _ = turn.Emit(ctx, wire.StepBeginEvent{N: 1})
    _ = turn.Emit(ctx, wire.ContentPartEvent{
        Part: wire.ContentPart{
            Type: wire.ContentPartTypeText,
            Text: &wire.TextPart{Text: "Echo: " + input.Text},
        },
    })
    return wire.PromptResult{Status: wire.PromptStatusFinished}, nil
}

func main() {
    transport := // any wire.Transport, e.g. stdio, channel, or in-memory
    srv := wire.NewServer(transport, &echoAgent{}, wire.WithServerInfo("echo", "0.1.0"))
    log.Fatal(srv.Serve(context.Background()))
}
```

The server automatically frames `Agent.Prompt` with `TurnBegin`/`TurnEnd` events, enforces one turn at a time, and recovers from panics in the agent callback.

### Optional agent capabilities

Agents may optionally implement:

- `Steerer` — receive mid-turn `steer` input from the client.
- `Replayer` — handle `replay` requests.
- `PlanModeSwitcher` — handle `set_plan_mode` requests.

Missing optional capabilities return a JSON-RPC `-32601 method not found` response.

### Capability negotiation and hooks

During `initialize` the server stores client capabilities such as `supports_question` and `supports_plan_mode`. `Turn.AskQuestion` returns an error when the client does not advertise `supports_question`, and `set_plan_mode` is rejected with `-32000` when `supports_plan_mode` is absent.

Hook subscriptions declared by the client at initialize are matched by event name and optional regex. `Turn.TriggerHook` returns `allow` when no subscription matches; otherwise it sends a `HookRequest` and waits for the client response using the subscription timeout.

## Transport Implementations

### ChildProcessTransport

Spawns `kimi` as a child process and communicates over stdin/stdout:

```go
workDir := "/path/to/project"
session := "my-session"
model := "kimi-latest"
transport, err := wire.SpawnChildProcessTransport("kimi", wire.SpawnOptions{
    WorkDir: &workDir,
    Session: &session,
    Model:   &model,
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
