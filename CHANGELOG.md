# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-06-09

### Added

- Initial implementation of the Kimi Code CLI Wire Protocol v1.10 client.
- Strongly typed protocol structs: `Event`, `Request`, `ContentPart`, `DisplayBlock`, `ToolReturnValue`, etc.
- High-level `Client` with methods: `Initialize`, `Prompt`, `Replay`, `Steer`, `SetPlanMode`, `Cancel`, `Shutdown`.
- `Transport` abstraction with three implementations:
  - `ChildProcessTransport` — spawns `kimi --wire` as a child process.
  - `ChannelTransport` — in-memory pair for tests.
  - `InMemoryTransport` — injectable/inspectable transport for unit tests.
- `Dispatch` loop for handling incoming events and requests via the `Handler` interface.
- Secret redaction (`RedactSecrets`, `redactString`) covering API keys, tokens, passwords, Authorization headers, AWS keys, GitHub PATs, JWTs, URL credentials, and PEM/PGP private keys.
- JSON-RPC 2.0 envelope format support with serde roundtrip guarantees.
- Fuzz test for `ParseWireMessage`.
- CI workflow for build, test (with race detector), vet, and format check.

### Fixed

- `UserInput` and `ToolOutput` correctly marshal/unmarshal as either a plain string or an array of `ContentPart`.
- `ContentPartEvent` and `ToolCallEvent` serialize to the correct wire envelope format.
- `MarshalEvent` handles both value and pointer receivers.
- `ParseEvent` validates `ToolCall` inner `type: "function"` discriminator.
- `Client.Shutdown` no longer panics on concurrent readerLoop access.
- `readerLoop` applies backpressure instead of silently dropping events when `dispatchCh` is full.
- `ChildProcessTransport` uses `SetReadDeadline`/`SetWriteDeadline` for context-aware I/O without goroutine leaks.
- `ChildProcessTransport.logStderr` correctly redacts multi-line PEM and PGP private keys.
- `RedactSecrets` preserves exact integer values when processing `json.RawMessage` via `json.Number`.
- `WireError` now implements `Unwrap()` for `errors.Is` compatibility.
- All transport methods consistently return typed `*WireError` with `Cause` chaining.

[0.1.0]: https://github.com/ekhodzitsky/kimi-wire/releases/tag/v0.1.0
