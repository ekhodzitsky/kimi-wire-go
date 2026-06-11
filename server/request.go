package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ekhodzitsky/kimi-wire"
)

func (s *Server) nextRequestID() string {
	return fmt.Sprintf("srv-%d", atomic.AddUint64(&s.requestCounter, 1))
}

func (s *Server) sendRequestAndWait(ctx context.Context, req wire.Request) (json.RawMessage, error) {
	id := s.nextRequestID()
	ch := make(chan *wire.RawWireMessage, 1)

	s.mu.Lock()
	s.pending[id] = ch
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.pending, id)
		select {
		case <-ch:
		default:
		}
		s.mu.Unlock()
	}()

	payload, err := wire.MarshalRequest(req)
	if err != nil {
		return nil, err
	}
	envelope := wire.JSONRPCRequest[json.RawMessage]{
		JSONRPC: "2.0",
		Method:  "request",
		ID:      id,
		Params:  payload,
	}
	if err := s.writeMessage(ctx, envelope); err != nil {
		return nil, err
	}

	select {
	case msg := <-ch:
		if msg.Error != nil {
			return nil, &wire.WireError{Kind: wire.ErrRequestFailed, Message: msg.Error.Message, Code: msg.Error.Code}
		}
		return msg.Result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.serveCtx.Done():
		return nil, s.serveCtx.Err()
	}
}

func (s *Server) writeMessage(ctx context.Context, msg any) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return s.transport.WriteLine(ctx, string(b))
}

func (s *Server) emitEvent(ctx context.Context, event wire.Event) error {
	payload, err := wire.MarshalEvent(event)
	if err != nil {
		return err
	}
	envelope := wire.JSONRPCRequest[json.RawMessage]{
		JSONRPC: "2.0",
		Method:  "event",
		Params:  payload,
	}
	return s.writeMessage(ctx, envelope)
}

func (t *turn) RequestApproval(ctx context.Context, req wire.ApprovalRequest) (wire.ApprovalResponse, error) {
	result, err := t.s.sendRequestAndWait(ctx, req)
	if err != nil {
		return wire.ApprovalResponse{}, err
	}
	var resp wire.ApprovalResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return wire.ApprovalResponse{}, err
	}
	return resp, nil
}

func (t *turn) CallExternalTool(ctx context.Context, req wire.ToolCallRequest) (wire.ToolCallResponse, error) {
	ctx = t.s.applyDefaultTimeout(ctx)
	result, err := t.s.sendRequestAndWait(ctx, req)
	if err != nil {
		return wire.ToolCallResponse{}, err
	}
	var resp wire.ToolCallResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return wire.ToolCallResponse{}, err
	}
	return resp, nil
}

func (t *turn) AskQuestion(ctx context.Context, req wire.QuestionRequest) (wire.QuestionResponse, error) {
	t.s.mu.Lock()
	supports := t.s.clientCaps.SupportsQuestion != nil && *t.s.clientCaps.SupportsQuestion
	t.s.mu.Unlock()
	if !supports {
		return wire.QuestionResponse{}, &wire.WireError{
			Kind:    wire.ErrRequestFailed,
			Message: "Client does not support structured questions",
			Code:    codeQuestionNotSupported,
		}
	}
	ctx = t.s.applyDefaultTimeout(ctx)
	result, err := t.s.sendRequestAndWait(ctx, req)
	if err != nil {
		return wire.QuestionResponse{}, err
	}
	var resp wire.QuestionResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return wire.QuestionResponse{}, err
	}
	return resp, nil
}

func (t *turn) TriggerHook(ctx context.Context, req wire.HookRequest) (wire.HookResponse, error) {
	t.s.mu.Lock()
	var sub *hookSubscription
	for _, candidate := range t.s.hooks {
		if candidate.event != req.Event {
			continue
		}
		if candidate.matcher != nil && !candidate.matcher.MatchString(req.Target) {
			continue
		}
		sub = candidate
		break
	}
	t.s.mu.Unlock()

	if sub == nil {
		return wire.HookResponse{RequestID: req.ID, Action: wire.HookActionAllow}, nil
	}

	req.SubscriptionID = sub.id
	ctx = t.s.applyHookTimeout(ctx, sub.timeout)
	result, err := t.s.sendRequestAndWait(ctx, req)
	if err != nil {
		return wire.HookResponse{}, err
	}
	var resp wire.HookResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return wire.HookResponse{}, err
	}
	return resp, nil
}

func (s *Server) applyDefaultTimeout(ctx context.Context) context.Context {
	if s.defaultTimeout <= 0 {
		return ctx
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx
	}
	ctx, cancel := context.WithTimeout(ctx, s.defaultTimeout)
	_ = cancel // caller owns the original context; timeout applies to this call only
	return ctx
}

func (s *Server) applyHookTimeout(ctx context.Context, d time.Duration) context.Context {
	if d <= 0 {
		d = 30 * time.Second
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx
	}
	ctx, cancel := context.WithTimeout(ctx, d)
	_ = cancel
	return ctx
}
