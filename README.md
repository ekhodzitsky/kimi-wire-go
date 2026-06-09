# kimi-wire (Go)

Typed Go client for the Kimi Code CLI Wire protocol.

## Overview

The Wire protocol is a JSON-RPC 2.0 based bidirectional communication protocol exposed by `kimi --wire`. This library provides:

- Strongly typed protocol structs (`Event`, `Request`, `PromptResult`, ...)
- A `Client` with high-level methods (`Prompt`, `Replay`, `Steer`, `SetPlanMode`, `Cancel`)
- A `Transport` abstraction for stdio, in-memory channels, or custom backends
- Optional secret redaction for wire logs

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
    transport, err := wire.SpawnChildProcessTransport("kimi", nil, nil, nil)
    if err != nil {
        log.Fatal(err)
    }
    client := wire.NewClient(transport)

    result, err := client.Prompt(ctx, wire.UserInput{Text: "Hello!"})
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Status:", result.Status)
}
```

## Testing

```bash
go test ./...
```

## License

MIT
