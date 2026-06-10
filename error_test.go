package wire

import (
	"errors"
	"testing"
	"time"
)

func TestWireError_Error(t *testing.T) {
	err := &WireError{Kind: ErrStreamClosed, Message: "wire stream closed"}
	want := "wire error stream_closed: wire stream closed"
	if err.Error() != want {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}
}

func TestWireError_ValueReceiver(t *testing.T) {
	var err error = WireError{Kind: ErrStreamClosed, Message: "closed"}
	want := "wire error stream_closed: closed"
	if err.Error() != want {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}
}

func TestWireError_As(t *testing.T) {
	err := &WireError{Kind: ErrTimeout, Message: "timeout", Duration: time.Second}
	var target *WireError
	if !errors.As(err, &target) {
		t.Fatal("expected errors.As to succeed")
	}
	if target.Kind != ErrTimeout {
		t.Fatalf("expected kind ErrTimeout, got %v", target.Kind)
	}
}
