package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ekhodzitsky/kimi-wire/protocol"
)

func (s *Server) nextRequestID() string {
	return fmt.Sprintf("srv-%d", atomic.AddUint64(&s.requestCounter, 1))
}

func (s *Server) sendRequestAndWait(ctx context.Context, req protocol.Request) (json.RawMessage, error) {
	id := s.nextRequestID()
	ch := make(chan *protocol.RawWireMessage, 1)

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

	payload, err := protocol.MarshalRequest(req)
	if err != nil {
		return nil, err
	}
	envelope := protocol.JSONRPCRequest[json.RawMessage]{
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
			return nil, &codedError{code: msg.Error.Code, msg: msg.Error.Message}
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

func (s *Server) emitEvent(ctx context.Context, event protocol.Event) error {
	payload, err := protocol.MarshalEvent(event)
	if err != nil {
		return err
	}
	envelope := protocol.JSONRPCRequest[json.RawMessage]{
		JSONRPC: "2.0",
		Method:  "event",
		Params:  payload,
	}
	return s.writeMessage(ctx, envelope)
}

func (t *turn) RequestApproval(ctx context.Context, req protocol.ApprovalRequest) (protocol.ApprovalResponse, error) {
	result, err := t.s.sendRequestAndWait(ctx, req)
	if err != nil {
		return protocol.ApprovalResponse{}, err
	}
	var resp protocol.ApprovalResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return protocol.ApprovalResponse{}, err
	}
	return resp, nil
}

func (t *turn) CallExternalTool(ctx context.Context, req protocol.ToolCallRequest) (protocol.ToolCallResponse, error) {
	ctx = t.s.applyDefaultTimeout(ctx)
	result, err := t.s.sendRequestAndWait(ctx, req)
	if err != nil {
		return protocol.ToolCallResponse{}, err
	}
	var resp protocol.ToolCallResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return protocol.ToolCallResponse{}, err
	}
	return resp, nil
}

func (t *turn) AskQuestion(ctx context.Context, req protocol.QuestionRequest) (protocol.QuestionResponse, error) {
	t.s.mu.Lock()
	supports := t.s.clientCaps.SupportsQuestion != nil && *t.s.clientCaps.SupportsQuestion
	t.s.mu.Unlock()
	if !supports {
		return protocol.QuestionResponse{}, &codedError{
			code: codeQuestionNotSupported,
			msg:  "Client does not support structured questions",
		}
	}
	ctx = t.s.applyDefaultTimeout(ctx)
	result, err := t.s.sendRequestAndWait(ctx, req)
	if err != nil {
		return protocol.QuestionResponse{}, err
	}
	var resp protocol.QuestionResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return protocol.QuestionResponse{}, err
	}
	return resp, nil
}

func (t *turn) TriggerHook(ctx context.Context, req protocol.HookRequest) (protocol.HookResponse, error) {
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
		return protocol.HookResponse{RequestID: req.ID, Action: protocol.HookActionAllow}, nil
	}

	req.SubscriptionID = sub.id
	ctx = t.s.applyHookTimeout(ctx, sub.timeout)
	result, err := t.s.sendRequestAndWait(ctx, req)
	if err != nil {
		return protocol.HookResponse{}, err
	}
	var resp protocol.HookResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return protocol.HookResponse{}, err
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
