package server

import (
	"context"
	"sync"

	"github.com/ekhodzitsky/kimi-wire"
)

// Emitter emits wire events to the client.
type Emitter interface {
	Emit(ctx context.Context, event wire.Event) error
}

// Turn is the interface exposed to an Agent during a prompt turn.
type Turn interface {
	Emitter
	RequestApproval(ctx context.Context, req wire.ApprovalRequest) (wire.ApprovalResponse, error)
	CallExternalTool(ctx context.Context, req wire.ToolCallRequest) (wire.ToolCallResponse, error)
	AskQuestion(ctx context.Context, req wire.QuestionRequest) (wire.QuestionResponse, error)
	TriggerHook(ctx context.Context, req wire.HookRequest) (wire.HookResponse, error)
}

// turn represents one active agent turn.
type turn struct {
	s       *Server
	ctx     context.Context
	cancel  context.CancelFunc
	input   wire.UserInput
	steerCh chan wire.UserInput
	done    chan struct{}
	result  wire.PromptResult
	err     error

	mu     sync.Mutex
	closed bool
}

func newTurn(s *Server, input wire.UserInput) *turn {
	ctx, cancel := context.WithCancel(s.serveCtx)
	return &turn{
		s:       s,
		ctx:     ctx,
		cancel:  cancel,
		input:   input,
		steerCh: make(chan wire.UserInput, 1),
		done:    make(chan struct{}),
	}
}

func (t *turn) Emit(ctx context.Context, event wire.Event) error {
	return t.s.emitEvent(ctx, event)
}

// SteerInput returns the next steer input delivered to this turn.
func (t *turn) SteerInput(ctx context.Context) (wire.UserInput, error) {
	select {
	case in := <-t.steerCh:
		return in, nil
	case <-t.ctx.Done():
		return wire.UserInput{}, t.ctx.Err()
	case <-ctx.Done():
		return wire.UserInput{}, ctx.Err()
	}
}

func (t *turn) close(result wire.PromptResult, err error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return
	}
	t.closed = true
	t.result = result
	t.err = err
	t.mu.Unlock()
	t.cancel()
	close(t.done)
}
