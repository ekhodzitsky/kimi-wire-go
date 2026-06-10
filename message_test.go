package wire

import (
	"encoding/json"
	"testing"
)

func TestParseWireMessageEvent(t *testing.T) {
	raw := RawWireMessage{
		JSONRPC: "2.0",
		Method:  "event",
		Params:  json.RawMessage(`{"type":"TurnEnd","payload":{}}`),
	}
	msg, err := ParseWireMessage(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, ok := msg.(EventMessage)
	if !ok {
		t.Fatalf("expected EventMessage, got %T", msg)
	}
}

func TestParseWireMessageRequest(t *testing.T) {
	raw := RawWireMessage{
		JSONRPC: "2.0",
		ID:      "req-1",
		Method:  "request",
		Params:  json.RawMessage(`{"type":"ApprovalRequest","payload":{"id":"ar-1","tool_call_id":"tc-1","sender":"fs","action":"write","description":"desc"}}`),
	}
	msg, err := ParseWireMessage(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, ok := msg.(RequestMessage)
	if !ok {
		t.Fatalf("expected RequestMessage, got %T", msg)
	}
}

func TestParseWireMessageSuccessResponse(t *testing.T) {
	raw := RawWireMessage{
		JSONRPC: "2.0",
		ID:      "1",
		Result:  json.RawMessage(`{"status":"finished"}`),
	}
	msg, err := ParseWireMessage(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, ok := msg.(SuccessResponseMessage)
	if !ok {
		t.Fatalf("expected SuccessResponseMessage, got %T", msg)
	}
}

func TestParseWireMessageInvalidJSONRPC(t *testing.T) {
	raw := RawWireMessage{
		JSONRPC: "1.0",
		Method:  "event",
		Params:  json.RawMessage(`{"type":"TurnEnd","payload":{}}`),
	}
	if _, err := ParseWireMessage(raw); err == nil {
		t.Fatal("expected error for invalid jsonrpc version")
	}
}

func TestEventMessageMarshalJSON(t *testing.T) {
	m := EventMessage{
		JSONRPC: "2.0",
		Method:  "event",
		Event:   TurnEndEvent{},
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"jsonrpc":"2.0","method":"event","params":{"type":"TurnEnd","payload":{}}}`
	if string(data) != want {
		t.Fatalf("expected %s, got %s", want, data)
	}
}

func FuzzParseWireMessage(f *testing.F) {
	f.Add(`{"jsonrpc":"2.0","method":"event","params":{"type":"TurnEnd","payload":{}}}`)
	f.Add(`{"jsonrpc":"2.0","method":"request","params":{"type":"ApprovalRequest","payload":{}}}`)
	f.Add(`{"jsonrpc":"2.0","id":"1","result":{"status":"ok"}}`)
	f.Add(`{"jsonrpc":"2.0","id":"1","error":{"code":-1,"message":"err"}}`)
	f.Fuzz(func(t *testing.T, in string) {
		var raw RawWireMessage
		_ = json.Unmarshal([]byte(in), &raw)
		_, _ = ParseWireMessage(raw) // must not panic
	})
}

func TestRequestMessageMarshalJSON(t *testing.T) {
	m := RequestMessage{
		JSONRPC: "2.0",
		Method:  "request",
		ID:      "r1",
		Request: ToolCallRequest{ID: "tc1", Name: "read"},
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"jsonrpc":"2.0","method":"request","id":"r1","params":{"type":"ToolCallRequest","payload":{"id":"tc1","name":"read"}}}`
	if string(data) != want {
		t.Fatalf("expected %s, got %s", want, data)
	}
}
