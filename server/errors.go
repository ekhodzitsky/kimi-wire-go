package server

import (
	"encoding/json"

	"github.com/ekhodzitsky/kimi-wire/internal/redact"
	"github.com/ekhodzitsky/kimi-wire/protocol"
)

// CodedError is an error that carries a JSON-RPC error code.
type CodedError interface {
	error
	Code() int
}

type codedError struct {
	code int
	msg  string
}

func (e *codedError) Error() string { return e.msg }
func (e *codedError) Code() int     { return e.code }

const (
	codeTurnInProgress       = -32000
	codeQuestionNotSupported = -32000
	codePlanModeNotSupported = -32000
	codeParseError           = -32700
	codeInvalidRequest       = -32600
	codeMethodNotFound       = -32601
	codeInvalidParams        = -32602
	codeInternalError        = -32603
)

func (s *Server) sendError(id string, code int, msg string) error {
	return s.transport.WriteLine(s.serveCtx, marshalJSONRPCError(id, code, msg))
}

func marshalJSONRPCError(id string, code int, msg string) string {
	safe := redact.RedactString(msg)
	b, _ := json.Marshal(protocol.JSONRPCErrorResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &protocol.JSONRPCError{Code: code, Message: safe},
	})
	return string(b)
}
