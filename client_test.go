package wire

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestClientPrompt(t *testing.T) {
	ctx := context.Background()
	mem := NewInMemoryTransport()
	client := NewClient(mem)

	// InMemoryTransport buffers, so inject before calling Prompt.
	mem.Inject(`{"jsonrpc":"2.0","id":"req-1","result":{"status":"finished"}}`)

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

func TestClientReadResponseTimeout(t *testing.T) {
	ctx := context.Background()
	c := NewClient(NewInMemoryTransport()).WithDefaultTimeout(10 * time.Millisecond)
	_, err := c.Prompt(ctx, UserInput{Text: "x"})
	var we *WireError
	if !errors.As(err, &we) || we.Kind != ErrTimeout {
		t.Fatalf("expected timeout, got %v", err)
	}
}

func TestClientReadResponseOutOfOrder(t *testing.T) {
	mem := NewInMemoryTransport()
	c := NewClient(mem)
	mem.Inject(`{"jsonrpc":"2.0","id":"req-2","result":{"status":"finished"}}`)
	mem.Inject(`{"jsonrpc":"2.0","id":"req-1","result":{"status":"cancelled"}}`)
	var got PromptResult
	if err := c.readResponse(context.Background(), "req-1", &got); err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Status != PromptStatusCancelled {
		t.Fatalf("expected cancelled, got %s", got.Status)
	}
}

func TestClientInitializeMethodNotFound(t *testing.T) {
	mem := NewInMemoryTransport()
	c := NewClient(mem)
	mem.Inject(`{"jsonrpc":"2.0","id":"req-1","error":{"code":-32601,"message":"method not found"}}`)
	res, err := c.Initialize(context.Background(), InitializeParams{ProtocolVersion: WireProtocolVersion})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if !c.IsHandshakeDone() || res.ProtocolVersion != WireProtocolLegacyVersion {
		t.Fatalf("unexpected legacy handshake result: %+v", res)
	}
}

func TestClientShutdownRespectsContext(t *testing.T) {
	mem := NewInMemoryTransport()
	c := NewClient(mem)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.Shutdown(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

func TestClientSendRawNil(t *testing.T) {
	mem := NewInMemoryTransport()
	c := NewClient(mem)
	if err := c.SendRaw(context.Background(), nil); err == nil {
		t.Fatal("expected error for nil raw message")
	}
}

func TestWireErrorUnwrap(t *testing.T) {
	root := context.Canceled
	err := &WireError{Kind: ErrTimeout, Message: "timed out", Cause: root}
	if !errors.Is(err, context.Canceled) {
		t.Fatal("expected errors.Is to find context.Canceled")
	}
}
