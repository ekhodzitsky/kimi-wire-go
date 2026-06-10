// Package wire provides a typed Go client for the Kimi Code CLI Wire protocol.
//
// The library speaks JSON-RPC 2.0 over newline-delimited transports and offers
// strongly typed protocol structs, pluggable transports (stdio, in-memory,
// channels), automatic secret redaction, and idiomatic Go error handling.
//
// Use NewClient to create a high-level client backed by a Transport
// implementation, then call Initialize and Prompt to interact with the agent.
package wire
