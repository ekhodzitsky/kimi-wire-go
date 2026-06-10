package transport

import (
	"context"
	"fmt"
	"io"
	"sync"
)

// InMemoryTransport is a transport for unit tests that supports injection and inspection.
type InMemoryTransport struct {
	incoming chan string
	outgoing []string
	mu       sync.Mutex
	closed   bool
}

// NewInMemoryTransport creates a new in-memory transport.
func NewInMemoryTransport() *InMemoryTransport {
	return &InMemoryTransport{incoming: make(chan string, 1024)}
}

func (t *InMemoryTransport) ReadLine(ctx context.Context) (string, error) {
	select {
	case line, ok := <-t.incoming:
		if !ok {
			return "", io.EOF
		}
		return line, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (t *InMemoryTransport) WriteLine(ctx context.Context, line string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return fmt.Errorf("transport closed")
	}
	t.outgoing = append(t.outgoing, line)
	return nil
}

func (t *InMemoryTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.closed {
		t.closed = true
		close(t.incoming)
	}
	return nil
}

// Inject adds an incoming line for the client to read.
func (t *InMemoryTransport) Inject(line string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return fmt.Errorf("transport closed")
	}
	select {
	case t.incoming <- line:
		return nil
	default:
		return fmt.Errorf("incoming buffer full")
	}
}

// Outgoing returns all lines written by the client.
func (t *InMemoryTransport) Outgoing() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]string, len(t.outgoing))
	copy(out, t.outgoing)
	return out
}
