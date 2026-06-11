package wire

import (
	"context"
	"testing"
	"time"
)

type testHandler struct {
	events []Event
}

func (h *testHandler) HandleEvent(ctx context.Context, event Event) error {
	h.events = append(h.events, event)
	return nil
}

func (h *testHandler) HandleRequest(ctx context.Context, req Request) (any, error) {
	return nil, nil
}

func TestDispatchEvent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	mem := NewInMemoryTransport()
	client := NewClient(mem)

	h := &testHandler{}
	go func() {
		if err := mem.Inject(`{"jsonrpc":"2.0","method":"event","params":{"type":"TurnEnd","payload":{}}}`); err != nil {
			panic(err)
		}
	}()

	err := client.Dispatch(ctx, h)
	if err != nil && err != context.DeadlineExceeded {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(h.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(h.events))
	}
}
