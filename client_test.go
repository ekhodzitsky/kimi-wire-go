package wire

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestClientPrompt(t *testing.T) {
	ctx := context.Background()
	mem := NewInMemoryTransport()
	client := NewClient(mem)

	go func() {
		time.Sleep(10 * time.Millisecond)
		mem.Inject(`{"jsonrpc":"2.0","id":"req-1","result":{"status":"finished"}}`)
	}()

	result, err := client.Prompt(ctx, UserInput{Text: "hello"})
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}
	if result.Status != PromptStatusFinished {
		t.Fatalf("expected finished, got %s", result.Status)
	}

	out := mem.Outgoing()
	if len(out) != 1 {
		t.Fatalf("expected 1 outgoing message, got %d", len(out))
	}
	var req JSONRPCRequest[PromptParams]
	if err := json.Unmarshal([]byte(out[0]), &req); err != nil {
		t.Fatalf("unmarshal outgoing: %v", err)
	}
	if req.Method != "prompt" {
		t.Fatalf("expected method prompt, got %s", req.Method)
	}
}
