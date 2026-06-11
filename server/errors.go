package server

import (
	"encoding/json"

	"github.com/ekhodzitsky/kimi-wire"
	"github.com/ekhodzitsky/kimi-wire/internal/redact"
)

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
	b, _ := json.Marshal(wire.JSONRPCErrorResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &wire.JSONRPCError{Code: code, Message: safe},
	})
	return string(b)
}
