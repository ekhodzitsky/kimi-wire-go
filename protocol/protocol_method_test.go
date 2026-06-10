package protocol

import (
	"encoding/json"
	"testing"
)

func TestInitializeParamsRoundtrip(t *testing.T) {
	original := InitializeParams{ProtocolVersion: "1.10"}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var parsed InitializeParams
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed.ProtocolVersion != "1.10" {
		t.Fatalf("expected 1.10, got %s", parsed.ProtocolVersion)
	}
}

func TestPromptResultRoundtrip(t *testing.T) {
	steps := uint64(3)
	original := PromptResult{Status: PromptStatusFinished, Steps: &steps}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var parsed PromptResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed.Status != PromptStatusFinished {
		t.Fatalf("status mismatch")
	}
}
