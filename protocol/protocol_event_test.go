package protocol

import (
	"encoding/json"
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

func TestEventContentPartEnvelope(t *testing.T) {
	original := ContentPartEvent{Part: ContentPart{Type: ContentPartTypeText, Text: &TextPart{Text: "hi"}}}
	data, err := MarshalEvent(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var env struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if env.Type != "ContentPart" {
		t.Fatalf("expected type ContentPart, got %s", env.Type)
	}
	var part ContentPart
	if err := json.Unmarshal(env.Payload, &part); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if part.Type != ContentPartTypeText || part.Text == nil || part.Text.Text != "hi" {
		t.Fatalf("payload mismatch: %+v", part)
	}
}

func TestTypeName(t *testing.T) {
	if got := TypeName(TurnEndEvent{}); got != "TurnEnd" {
		t.Fatalf("expected TurnEnd, got %q", got)
	}
	if got := TypeName(nil); got != "" {
		t.Fatalf("expected empty string for nil, got %q", got)
	}
}

func TestMarshalEventPointerTypes(t *testing.T) {
	cp := &ContentPartEvent{Part: ContentPart{Type: ContentPartTypeText, Text: &TextPart{Text: "ptr"}}}
	data, err := MarshalEvent(cp)
	if err != nil {
		t.Fatalf("marshal ptr ContentPartEvent: %v", err)
	}
	var env struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Type != "ContentPart" {
		t.Fatalf("expected type ContentPart, got %s", env.Type)
	}

	args := `{}`
	tc := &ToolCallEvent{ID: "tc1", Function: ToolCallFunction{Name: "read", Arguments: &args}}
	data, err = MarshalEvent(tc)
	if err != nil {
		t.Fatalf("marshal ptr ToolCallEvent: %v", err)
	}
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Type != "ToolCall" {
		t.Fatalf("expected type ToolCall, got %s", env.Type)
	}
	var payload struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Type != "function" {
		t.Fatalf("expected inner type=function, got %q", payload.Type)
	}
}

func TestParseEventToolCallInvalidInnerType(t *testing.T) {
	_, err := ParseEvent([]byte(`{"type":"ToolCall","payload":{"type":"not_function","id":"tc1","function":{"name":"read"}}}`))
	if err == nil {
		t.Fatal("expected error for invalid inner tool call type")
	}
}
