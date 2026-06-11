package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ekhodzitsky/kimi-wire/internal/redact"
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
		s.mu.Unlock()
		select {
		case <-ch:
		default:
		}
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
			return nil, &codedError{code: msg.Error.Code, msg: redact.RedactString(msg.Error.Message)}
		}
		return msg.Result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.serveCtx.Done():
		return nil, s.serveCtx.Err()
	}
}

// jsonrpcNotification is a JSON-RPC 2.0 notification (no id field).
type jsonrpcNotification[T any] struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  T      `json:"params"`
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
	envelope := jsonrpcNotification[json.RawMessage]{
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
	if resp.RequestID != req.ID {
		return protocol.ApprovalResponse{}, fmt.Errorf("approval response id mismatch: got %q, want %q", resp.RequestID, req.ID)
	}
	return resp, nil
}

func (t *turn) CallExternalTool(ctx context.Context, req protocol.ToolCallRequest) (protocol.ToolCallResponse, error) {
	ctx, cancel := t.s.applyDefaultTimeout(ctx)
	defer cancel()
	result, err := t.s.sendRequestAndWait(ctx, req)
	if err != nil {
		return protocol.ToolCallResponse{}, err
	}
	var resp protocol.ToolCallResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return protocol.ToolCallResponse{}, err
	}
	if resp.ToolCallID != req.ID {
		return protocol.ToolCallResponse{}, fmt.Errorf("tool response id mismatch: got %q, want %q", resp.ToolCallID, req.ID)
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
	ctx, cancel := t.s.applyDefaultTimeout(ctx)
	defer cancel()
	result, err := t.s.sendRequestAndWait(ctx, req)
	if err != nil {
		return protocol.QuestionResponse{}, err
	}
	var resp protocol.QuestionResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return protocol.QuestionResponse{}, err
	}
	if resp.RequestID != req.ID {
		return protocol.QuestionResponse{}, fmt.Errorf("question response id mismatch: got %q, want %q", resp.RequestID, req.ID)
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
	ctx, cancel := t.s.applyHookTimeout(ctx, sub.timeout)
	defer cancel()
	result, err := t.s.sendRequestAndWait(ctx, req)
	if err != nil {
		return protocol.HookResponse{}, err
	}
	var resp protocol.HookResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return protocol.HookResponse{}, err
	}
	if resp.RequestID != req.ID {
		return protocol.HookResponse{}, fmt.Errorf("hook response id mismatch: got %q, want %q", resp.RequestID, req.ID)
	}
	return resp, nil
}

func (s *Server) applyDefaultTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if s.defaultTimeout <= 0 {
		return ctx, func() {}
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, s.defaultTimeout)
}

func (s *Server) applyHookTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if d <= 0 {
		d = 30 * time.Second
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}
