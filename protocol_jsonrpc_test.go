package wire

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestRawWireMessageRoundtrip(t *testing.T) {
	original := RawWireMessage{
		JSONRPC: "2.0",
		ID:      "req-1",
		Method:  "prompt",
		Params:  json.RawMessage(`{"user_input":"hello"}`),
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var parsed RawWireMessage
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(original, parsed) {
		t.Fatalf("roundtrip mismatch:\noriginal: %+v\nparsed: %+v", original, parsed)
	}
}

func TestJSONRPCRequestRoundtrip(t *testing.T) {
	original := JSONRPCRequest[PromptParams]{
		JSONRPC: "2.0",
		Method:  "prompt",
		ID:      "1",
		Params:  PromptParams{UserInput: UserInput{Text: "hi"}},
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var parsed JSONRPCRequest[PromptParams]
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(original, parsed) {
		t.Fatalf("roundtrip mismatch")
	}
}

func TestJSONRPCSuccessResponseRoundtrip(t *testing.T) {
	original := JSONRPCSuccessResponse[PromptResult]{
		JSONRPC: "2.0",
		ID:      "1",
		Result:  PromptResult{Status: PromptStatusFinished},
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var parsed JSONRPCSuccessResponse[PromptResult]
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(original, parsed) {
		t.Fatalf("roundtrip mismatch")
	}
}

func TestJSONRPCErrorResponseRoundtrip(t *testing.T) {
	original := JSONRPCErrorResponse{
		JSONRPC: "2.0",
		ID:      "1",
		Error:   &JSONRPCError{Code: MethodNotFound, Message: "method not found"},
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var parsed JSONRPCErrorResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(original, parsed) {
		t.Fatalf("roundtrip mismatch")
	}
}
