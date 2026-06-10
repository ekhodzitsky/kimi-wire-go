# Examples

This directory contains runnable examples for `kimi-wire`.

## Quick Start

The [`quickstart`](quickstart) example demonstrates the minimal setup to spawn a `kimi --wire` child process, perform the initialization handshake, and send a prompt.

### Prerequisites

- Go 1.22 or later
- The `kimi` binary available in your `PATH`

### Running

```bash
cd quickstart
go run main.go
```

### What it does

1. Spawns `kimi` as a child process via `wire.SpawnChildProcessTransport`.
2. Creates a `wire.Client` backed by that transport.
3. Calls `client.Initialize` to perform the Wire protocol handshake.
4. Sends a `wire.UserInput` prompt with the text `"Hello!"`.
5. Prints the `Status` field of the returned `PromptResult`.
