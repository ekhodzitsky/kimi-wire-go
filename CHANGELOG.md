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

## [0.2.0] - 2026-06-10

### Added

- Runnable `examples/quickstart/` demonstrating child-process transport, initialization, and prompting.
- `Makefile` with targets: `build`, `test`, `test-race`, `coverage`, `fmt`, `vet`.
- Package-level `doc.go` for root godoc.
- `.github/dependabot.yml` for automated Go module updates.
- CI cache (`cache: true`) and `go-version-file: go.mod` in setup-go.
- `golangci-lint` step in CI workflow.

### Changed

- **Restructured into sub-packages while preserving backward compatibility via type aliases:**
  - `transport/` — `Transport`, `ChannelTransport`, `InMemoryTransport`, `ChildProcessTransport`.
  - `protocol/` — all protocol types (`Event`, `Request`, `UserInput`, `RawWireMessage`, etc.).
  - `internal/redact/` — secret redaction internals.
  - Root aliases (`transport.go`, `protocol.go`, `internal.go`) keep existing `wire.XYZ` imports working without changes.
- `.gitignore` extended to exclude `.omc/`, `docs/superpowers/`, build directories, and example binaries.

### Removed

- `docs/superpowers/` — generated superpowers artifacts.
- `.github/repo-meta.yml` — unused metadata file.
- `coverage.out` from working directory.

[0.2.0]: https://github.com/ekhodzitsky/kimi-wire/releases/tag/v0.2.0
[0.1.0]: https://github.com/ekhodzitsky/kimi-wire/releases/tag/v0.1.0
