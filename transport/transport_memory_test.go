package transport

import (
	"context"
	"testing"
)

func TestInMemoryTransportInjectAndRead(t *testing.T) {
	ctx := context.Background()
	mem := NewInMemoryTransport()
	mem.Inject("hello")
	line, err := mem.ReadLine(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if line != "hello" {
		t.Fatalf("expected hello, got %s", line)
	}
}

func TestInMemoryTransportOutgoing(t *testing.T) {
	ctx := context.Background()
	mem := NewInMemoryTransport()
	_ = mem.WriteLine(ctx, "world")
	out := mem.Outgoing()
	if len(out) != 1 || out[0] != "world" {
		t.Fatalf("expected [world], got %v", out)
	}
}
