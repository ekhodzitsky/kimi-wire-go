package wire

import (
	"testing"
)

func TestEventTurnBeginRoundtrip(t *testing.T) {
	original := TurnBeginEvent{UserInput: UserInput{Text: "hello"}}
	data, err := MarshalEvent(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	parsed, err := ParseEvent(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	ev, ok := parsed.(TurnBeginEvent)
	if !ok {
		t.Fatalf("expected TurnBeginEvent, got %T", parsed)
	}
	if ev.UserInput.Text != "hello" {
		t.Fatalf("payload mismatch")
	}
}

func TestEventToolCallRoundtrip(t *testing.T) {
	args := `{"path": "/tmp/foo"}`
	original := ToolCallEvent{
		ID: "tc-1",
		Function: ToolCallFunction{
			Name:      "read_file",
			Arguments: &args,
		},
	}
	data, err := MarshalEvent(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	parsed, err := ParseEvent(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	ev, ok := parsed.(ToolCallEvent)
	if !ok {
		t.Fatalf("expected ToolCallEvent, got %T", parsed)
	}
	if ev.Function.Name != "read_file" {
		t.Fatalf("function name mismatch")
	}
}

func TestEventTurnEndEnvelope(t *testing.T) {
	data, err := MarshalEvent(TurnEndEvent{})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	parsed, err := ParseEvent(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, ok := parsed.(TurnEndEvent); !ok {
		t.Fatalf("expected TurnEndEvent, got %T", parsed)
	}
}
