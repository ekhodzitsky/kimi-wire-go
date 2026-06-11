package server

import (
	"encoding/json"
	"testing"

	wire "github.com/ekhodzitsky/kimi-wire"
)

func FuzzParseServerMessage(f *testing.F) {
	f.Add(`{"jsonrpc":"2.0","method":"event","params":{"type":"TurnEnd","payload":{}}}`)
	f.Add(`{"jsonrpc":"2.0","method":"request","params":{"type":"ApprovalRequest","payload":{}}}`)
	f.Add(`{"jsonrpc":"2.0","id":"1","result":{"status":"ok"}}`)
	f.Add(`{"jsonrpc":"2.0","id":"1","error":{"code":-1,"message":"err"}}`)
	f.Fuzz(func(t *testing.T, in string) {
		var raw wire.RawWireMessage
		_ = json.Unmarshal([]byte(in), &raw)
		_, _ = wire.ParseWireMessage(raw) // must not panic
	})
}
