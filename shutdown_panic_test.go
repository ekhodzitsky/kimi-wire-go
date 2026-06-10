package wire

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type slowHangTransport struct {
	mu     sync.Mutex
	closed bool
	readCh chan string
}

func newSlowHangTransport() *slowHangTransport {
	return &slowHangTransport{readCh: make(chan string)}
}

func (t *slowHangTransport) ReadLine(ctx context.Context) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case line := <-t.readCh:
		return line, nil
	}
}
func (t *slowHangTransport) WriteLine(ctx context.Context, line string) error { return nil }
func (t *slowHangTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed = true
	return nil
}

func TestShutdownDoesNotPanicOnSlowTransport(t *testing.T) {
	tr := newSlowHangTransport()
	c := NewClient(tr)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := c.Shutdown(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	// Simulate a late message arriving after dispatchCh is closed.
	tr.readCh <- `{"jsonrpc":"2.0","method":"event","params":{"type":"TurnEnd","payload":{}}}`

	time.Sleep(200 * time.Millisecond)
}
