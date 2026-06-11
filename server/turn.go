package server

import (
	"context"
	"sync"

	"github.com/ekhodzitsky/kimi-wire/protocol"
)

// Emitter emits wire events to the client.
type Emitter interface {
	Emit(ctx context.Context, event protocol.Event) error
}

// Turn is the interface exposed to an Agent during a prompt turn.
type Turn interface {
	Emitter
	RequestApproval(ctx context.Context, req protocol.ApprovalRequest) (protocol.ApprovalResponse, error)
	CallExternalTool(ctx context.Context, req protocol.ToolCallRequest) (protocol.ToolCallResponse, error)
	AskQuestion(ctx context.Context, req protocol.QuestionRequest) (protocol.QuestionResponse, error)
	TriggerHook(ctx context.Context, req protocol.HookRequest) (protocol.HookResponse, error)
}

// turn represents one active agent turn.
type turn struct {
	s       *Server
	ctx     context.Context
	cancel  context.CancelFunc
	steerCh chan protocol.UserInput
	done    chan struct{}

	mu     sync.Mutex
	closed bool
}

func newTurn(s *Server, input protocol.UserInput) *turn {
	ctx, cancel := context.WithCancel(s.serveCtx)
	return &turn{
		s:       s,
		ctx:     ctx,
		cancel:  cancel,
		steerCh: make(chan protocol.UserInput, 1),
		done:    make(chan struct{}),
	}
}

func (t *turn) Emit(ctx context.Context, event protocol.Event) error {
	return t.s.emitEvent(ctx, event)
}

// SteerInput returns the next steer input delivered to this turn.
func (t *turn) SteerInput(ctx context.Context) (protocol.UserInput, error) {
	select {
	case in := <-t.steerCh:
		return in, nil
	case <-t.ctx.Done():
		return protocol.UserInput{}, t.ctx.Err()
	case <-ctx.Done():
		return protocol.UserInput{}, ctx.Err()
	}
}

func (t *turn) close() {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return
	}
	t.closed = true
	t.mu.Unlock()
	t.cancel()
	close(t.done)
}
