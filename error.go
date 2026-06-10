package wire

import (
	"fmt"
	"time"
)

// WireError is a typed error for wire protocol failures.
type WireError struct {
	Kind     ErrorKind
	Message  string
	Cause    error
	Code     int
	Expected string
	Got      string
	Duration time.Duration
}

// ErrorKind discriminates wire errors.
type ErrorKind int

const (
	ErrStreamClosed ErrorKind = iota
	ErrTimeout
	ErrSpawnFailed
	ErrJSONParse
	ErrJSONSerialize
	ErrRequestFailed
	ErrUnexpectedResponseID
	ErrMethodNotFound
	ErrUnknownMessageType
	ErrInvalidPayloadType
	ErrIO
	ErrInternal
)

// String returns the error kind name.
func (k ErrorKind) String() string {
	switch k {
	case ErrStreamClosed:
		return "stream_closed"
	case ErrTimeout:
		return "timeout"
	case ErrSpawnFailed:
		return "spawn_failed"
	case ErrJSONParse:
		return "json_parse"
	case ErrJSONSerialize:
		return "json_serialize"
	case ErrRequestFailed:
		return "request_failed"
	case ErrUnexpectedResponseID:
		return "unexpected_response_id"
	case ErrMethodNotFound:
		return "method_not_found"
	case ErrUnknownMessageType:
		return "unknown_message_type"
	case ErrInvalidPayloadType:
		return "invalid_payload_type"
	case ErrIO:
		return "io"
	case ErrInternal:
		return "internal"
	default:
		return fmt.Sprintf("unknown(%d)", k)
	}
}

// Unwrap returns the underlying cause of the error.
func (e *WireError) Unwrap() error { return e.Cause }

func (e WireError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("wire error: %s", e.Kind)
	}
	return fmt.Sprintf("wire error %s: %s", e.Kind, e.Message)
}
