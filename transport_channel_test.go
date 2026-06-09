package wire

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
