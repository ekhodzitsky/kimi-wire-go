# kimi-wire (Go)

[![CI](https://github.com/ekhodzitsky/kimi-wire/actions/workflows/ci.yml/badge.svg)](https://github.com/ekhodzitsky/kimi-wire/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ekhodzitsky/kimi-wire)](https://goreportcard.com/report/github.com/ekhodzitsky/kimi-wire)

Typed Go client for the [Kimi Code CLI](https://github.com/ekhodzitsky/kimi) Wire protocol.

## Overview

The Wire protocol is a JSON-RPC 2.0 based bidirectional communication protocol exposed by `kimi --wire`. This library provides:

- **Strongly typed protocol structs** (`Event`, `Request`, `PromptResult`, ...)
- **High-level `Client`** with methods: `Prompt`, `Replay`, `Steer`, `SetPlanMode`, `Cancel`, `Initialize`
- **`Transport` abstraction** for stdio (child process), in-memory channels, or custom backends
- **Secret redaction** for wire logs and error messages
- **Serde roundtrip guarantees** for all protocol types

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
