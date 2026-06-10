package transport

import (
	"context"
	"testing"
)

func TestChannelTransportReadWrite(t *testing.T) {
	ctx := context.Background()
	a, b := NewChannelTransportPair()
	go func() {
		if err := a.WriteLine(ctx, "hello"); err != nil {
			t.Errorf("write: %v", err)
		}
	}()
	line, err := b.ReadLine(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if line != "hello" {
		t.Fatalf("expected hello, got %s", line)
	}
}

func TestChannelTransportWriteAfterClose(t *testing.T) {
	ctx := context.Background()
	a, _ := NewChannelTransportPair()
	if err := a.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := a.WriteLine(ctx, "hello"); err == nil {
		t.Fatal("expected error writing to closed transport")
	}
}

func TestChannelTransportCloseIdempotent(t *testing.T) {
	a, _ := NewChannelTransportPair()
	if err := a.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := a.Close(); err != nil {
		t.Fatalf("second close should be idempotent: %v", err)
	}
}
