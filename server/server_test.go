package server_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ekhodzitsky/kimi-wire"
	"github.com/ekhodzitsky/kimi-wire/server"
)

// ---------------------------------------------------------------------------
// Test agents
// ---------------------------------------------------------------------------

type happyAgent struct{}

func (a *happyAgent) Prompt(ctx context.Context, input wire.UserInput, turn server.Turn) (wire.PromptResult, error) {
	_ = turn.Emit(ctx, wire.StepBeginEvent{N: 1})
	_ = turn.Emit(ctx, wire.ContentPartEvent{Part: wire.ContentPart{Type: wire.ContentPartTypeText, Text: &wire.TextPart{Text: "Echo: " + input.Text}}})
	return wire.PromptResult{Status: wire.PromptStatusFinished}, nil
}

type approvalAgent struct {
	response wire.ApprovalResponseKind
}

func (a *approvalAgent) Prompt(ctx context.Context, input wire.UserInput, turn server.Turn) (wire.PromptResult, error) {
	req := wire.ApprovalRequest{ID: "appr1", Action: "write", Description: "write file"}
	resp, err := turn.RequestApproval(ctx, req)
	if err != nil {
		return wire.PromptResult{}, err
	}
	if resp.Response == wire.ApprovalResponseKindApprove {
		return wire.PromptResult{Status: wire.PromptStatusFinished}, nil
	}
	return wire.PromptResult{Status: wire.PromptStatusCancelled}, nil
}

type toolAgent struct{}

func (a *toolAgent) Prompt(ctx context.Context, input wire.UserInput, turn server.Turn) (wire.PromptResult, error) {
	req := wire.ToolCallRequest{ID: "tc1", Name: "read_file"}
	resp, err := turn.CallExternalTool(ctx, req)
	if err != nil {
		return wire.PromptResult{}, err
	}
	_ = turn.Emit(ctx, wire.ToolResultEvent(resp))
	return wire.PromptResult{Status: wire.PromptStatusFinished}, nil
}

type questionAgent struct{}

func (a *questionAgent) Prompt(ctx context.Context, input wire.UserInput, turn server.Turn) (wire.PromptResult, error) {
	req := wire.QuestionRequest{ID: "q1", Questions: []wire.QuestionItem{{Question: "ok?"}}}
	_, err := turn.AskQuestion(ctx, req)
	if err != nil {
		return wire.PromptResult{}, err
	}
	return wire.PromptResult{Status: wire.PromptStatusFinished}, nil
}

type hookAgent struct {
	event  string
	target string
}

func (a *hookAgent) Prompt(ctx context.Context, input wire.UserInput, turn server.Turn) (wire.PromptResult, error) {
	req := wire.HookRequest{ID: "hk1", Event: a.event, Target: a.target}
	resp, err := turn.TriggerHook(ctx, req)
	if err != nil {
		return wire.PromptResult{}, err
	}
	if resp.Action == wire.HookActionBlock {
		return wire.PromptResult{Status: wire.PromptStatusCancelled}, nil
	}
	return wire.PromptResult{Status: wire.PromptStatusFinished}, nil
}

type steerAgent struct {
	steerReceived chan wire.UserInput
}

func (a *steerAgent) Prompt(ctx context.Context, input wire.UserInput, turn server.Turn) (wire.PromptResult, error) {
	select {
	case in := <-a.steerReceived:
		_ = turn.Emit(ctx, wire.ContentPartEvent{Part: wire.ContentPart{Type: wire.ContentPartTypeText, Text: &wire.TextPart{Text: "steered: " + in.Text}}})
	case <-ctx.Done():
		return wire.PromptResult{Status: wire.PromptStatusCancelled}, ctx.Err()
	}
	return wire.PromptResult{Status: wire.PromptStatusFinished}, nil
}

func (a *steerAgent) Steer(ctx context.Context, input wire.UserInput) error {
	select {
	case a.steerReceived <- input:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

type cancelAgent struct{}

func (a *cancelAgent) Prompt(ctx context.Context, input wire.UserInput, turn server.Turn) (wire.PromptResult, error) {
	<-ctx.Done()
	return wire.PromptResult{Status: wire.PromptStatusCancelled}, nil
}

type maxStepsAgent struct{}

func (a *maxStepsAgent) Prompt(ctx context.Context, input wire.UserInput, turn server.Turn) (wire.PromptResult, error) {
	return wire.PromptResult{Status: wire.PromptStatusMaxStepsReached}, nil
}

type blockingAgent struct {
	blocked chan struct{}
}

func (a *blockingAgent) Prompt(ctx context.Context, input wire.UserInput, turn server.Turn) (wire.PromptResult, error) {
	select {
	case <-a.blocked:
		return wire.PromptResult{Status: wire.PromptStatusFinished}, nil
	case <-ctx.Done():
		return wire.PromptResult{Status: wire.PromptStatusCancelled}, ctx.Err()
	}
}

type panicAgent struct{}

func (a *panicAgent) Prompt(ctx context.Context, input wire.UserInput, turn server.Turn) (wire.PromptResult, error) {
	panic("boom")
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestHappyPath(t *testing.T) {
	t.Parallel()
	clientTrans, cleanup := startServer(t, &happyAgent{})
	defer cleanup()

	writeReq(t, clientTrans, "initialize", "init1", wire.InitializeParams{ProtocolVersion: wire.WireProtocolVersion})
	var initRes wire.InitializeResult
	readRes(t, clientTrans, "init1", &initRes)
	if initRes.ProtocolVersion != wire.WireProtocolVersion {
		t.Fatalf("expected version %s, got %s", wire.WireProtocolVersion, initRes.ProtocolVersion)
	}

	writeReq(t, clientTrans, "prompt", "prompt1", wire.PromptParams{UserInput: wire.UserInput{Text: "hello"}})
	mustReadEventType(t, clientTrans, "TurnBegin")
	mustReadEventType(t, clientTrans, "StepBegin")
	mustReadEventType(t, clientTrans, "ContentPart")
	mustReadEventType(t, clientTrans, "TurnEnd")
	var promptRes wire.PromptResult
	readRes(t, clientTrans, "prompt1", &promptRes)
	if promptRes.Status != wire.PromptStatusFinished {
		t.Fatalf("expected finished, got %s", promptRes.Status)
	}
}

func TestApprovalApprove(t *testing.T) {
	t.Parallel()
	clientTrans, cleanup := startServer(t, &approvalAgent{response: wire.ApprovalResponseKindApprove})
	defer cleanup()

	writeReq(t, clientTrans, "initialize", "init1", wire.InitializeParams{ProtocolVersion: wire.WireProtocolVersion})
	mustReadResponse(t, clientTrans, "init1")

	writeReq(t, clientTrans, "prompt", "prompt1", wire.PromptParams{UserInput: wire.UserInput{Text: "go"}})
	mustReadEventType(t, clientTrans, "TurnBegin")
	id, req := mustReadRequest(t, clientTrans)
	ar, ok := req.(wire.ApprovalRequest)
	if !ok {
		t.Fatalf("expected ApprovalRequest, got %T", req)
	}
	if ar.Action != "write" {
		t.Fatalf("expected action write, got %s", ar.Action)
	}
	writeResponse(t, clientTrans, id, wire.ApprovalResponse{RequestID: ar.ID, Response: wire.ApprovalResponseKindApprove})
	mustReadEventType(t, clientTrans, "TurnEnd")
	var res wire.PromptResult
	readRes(t, clientTrans, "prompt1", &res)
	if res.Status != wire.PromptStatusFinished {
		t.Fatalf("expected finished, got %s", res.Status)
	}
}

func TestApprovalReject(t *testing.T) {
	t.Parallel()
	clientTrans, cleanup := startServer(t, &approvalAgent{response: wire.ApprovalResponseKindReject})
	defer cleanup()

	writeReq(t, clientTrans, "initialize", "init1", wire.InitializeParams{ProtocolVersion: wire.WireProtocolVersion})
	mustReadResponse(t, clientTrans, "init1")

	writeReq(t, clientTrans, "prompt", "prompt1", wire.PromptParams{UserInput: wire.UserInput{Text: "go"}})
	mustReadEventType(t, clientTrans, "TurnBegin")
	id, req := mustReadRequest(t, clientTrans)
	ar := req.(wire.ApprovalRequest)
	writeResponse(t, clientTrans, id, wire.ApprovalResponse{RequestID: ar.ID, Response: wire.ApprovalResponseKindReject})
	mustReadEventType(t, clientTrans, "TurnEnd")
	var res wire.PromptResult
	readRes(t, clientTrans, "prompt1", &res)
	if res.Status != wire.PromptStatusCancelled {
		t.Fatalf("expected cancelled, got %s", res.Status)
	}
}

func TestExternalTool(t *testing.T) {
	t.Parallel()
	clientTrans, cleanup := startServer(t, &toolAgent{})
	defer cleanup()

	writeReq(t, clientTrans, "initialize", "init1", wire.InitializeParams{ProtocolVersion: wire.WireProtocolVersion})
	mustReadResponse(t, clientTrans, "init1")

	writeReq(t, clientTrans, "prompt", "prompt1", wire.PromptParams{UserInput: wire.UserInput{Text: "go"}})
	mustReadEventType(t, clientTrans, "TurnBegin")
	id, req := mustReadRequest(t, clientTrans)
	tr, ok := req.(wire.ToolCallRequest)
	if !ok {
		t.Fatalf("expected ToolCallRequest, got %T", req)
	}
	if tr.Name != "read_file" {
		t.Fatalf("expected read_file, got %s", tr.Name)
	}
	writeResponse(t, clientTrans, id, wire.ToolCallResponse{
		ToolCallID:  tr.ID,
		ReturnValue: wire.ToolReturnValue{IsError: false, Output: wire.ToolOutput{Text: "contents"}, Message: "done"},
	})
	mustReadEventType(t, clientTrans, "ToolResult")
	mustReadEventType(t, clientTrans, "TurnEnd")
	var res wire.PromptResult
	readRes(t, clientTrans, "prompt1", &res)
	if res.Status != wire.PromptStatusFinished {
		t.Fatalf("expected finished, got %s", res.Status)
	}
}

func TestQuestionRequestGated(t *testing.T) {
	t.Parallel()
	clientTrans, cleanup := startServer(t, &questionAgent{})
	defer cleanup()

	writeReq(t, clientTrans, "initialize", "init1", wire.InitializeParams{
		ProtocolVersion: wire.WireProtocolVersion,
		Capabilities:    &wire.ClientCapabilities{},
	})
	mustReadResponse(t, clientTrans, "init1")

	writeReq(t, clientTrans, "prompt", "prompt1", wire.PromptParams{UserInput: wire.UserInput{Text: "go"}})
	mustReadEventType(t, clientTrans, "TurnBegin")
	mustReadEventType(t, clientTrans, "TurnEnd")
	raw := mustFindResponse(t, clientTrans, "prompt1")
	if raw.Error == nil {
		t.Fatalf("expected error response")
	}
	if raw.Error.Code != -32000 {
		t.Fatalf("expected code -32000, got %d", raw.Error.Code)
	}
	want := "Client does not support structured questions"
	if raw.Error.Message != want {
		t.Fatalf("expected message %q, got %q", want, raw.Error.Message)
	}
}

func TestHookRequestSubscription(t *testing.T) {
	t.Parallel()
	clientTrans, cleanup := startServer(t, &hookAgent{event: "before_tool", target: "read_file"}, server.WithSupportedHooks([]string{"before_tool"}))
	defer cleanup()

	writeReq(t, clientTrans, "initialize", "init1", wire.InitializeParams{
		ProtocolVersion: wire.WireProtocolVersion,
		Hooks: []wire.WireHookSubscription{
			{ID: "sub1", Event: "before_tool", Matcher: "read_.*", Timeout: 5},
		},
	})
	mustReadResponse(t, clientTrans, "init1")

	writeReq(t, clientTrans, "prompt", "prompt1", wire.PromptParams{UserInput: wire.UserInput{Text: "go"}})
	mustReadEventType(t, clientTrans, "TurnBegin")
	id, req := mustReadRequest(t, clientTrans)
	hr, ok := req.(wire.HookRequest)
	if !ok {
		t.Fatalf("expected HookRequest, got %T", req)
	}
	if hr.Event != "before_tool" {
		t.Fatalf("expected before_tool, got %s", hr.Event)
	}
	if hr.SubscriptionID != "sub1" {
		t.Fatalf("expected subscription sub1, got %s", hr.SubscriptionID)
	}
	writeResponse(t, clientTrans, id, wire.HookResponse{RequestID: hr.ID, Action: wire.HookActionAllow})
	mustReadEventType(t, clientTrans, "TurnEnd")
	var res wire.PromptResult
	readRes(t, clientTrans, "prompt1", &res)
	if res.Status != wire.PromptStatusFinished {
		t.Fatalf("expected finished, got %s", res.Status)
	}
}

func TestHookNoSubscriptionReturnsAllow(t *testing.T) {
	t.Parallel()
	clientTrans, cleanup := startServer(t, &hookAgent{event: "before_tool", target: "read_file"}, server.WithSupportedHooks([]string{"before_tool"}))
	defer cleanup()

	writeReq(t, clientTrans, "initialize", "init1", wire.InitializeParams{ProtocolVersion: wire.WireProtocolVersion})
	mustReadResponse(t, clientTrans, "init1")

	writeReq(t, clientTrans, "prompt", "prompt1", wire.PromptParams{UserInput: wire.UserInput{Text: "go"}})
	mustReadEventType(t, clientTrans, "TurnBegin")
	mustReadEventType(t, clientTrans, "TurnEnd")
	var res wire.PromptResult
	readRes(t, clientTrans, "prompt1", &res)
	if res.Status != wire.PromptStatusFinished {
		t.Fatalf("expected finished, got %s", res.Status)
	}
}

func TestSteerDuringTurn(t *testing.T) {
	t.Parallel()
	agent := &steerAgent{steerReceived: make(chan wire.UserInput, 1)}
	clientTrans, cleanup := startServer(t, agent)
	defer cleanup()

	writeReq(t, clientTrans, "initialize", "init1", wire.InitializeParams{ProtocolVersion: wire.WireProtocolVersion})
	mustReadResponse(t, clientTrans, "init1")

	writeReq(t, clientTrans, "prompt", "prompt1", wire.PromptParams{UserInput: wire.UserInput{Text: "go"}})
	mustReadEventType(t, clientTrans, "TurnBegin")

	writeReq(t, clientTrans, "steer", "steer1", wire.SteerParams{UserInput: wire.UserInput{Text: "more"}})
	var steerRes wire.SteerResult
	readRes(t, clientTrans, "steer1", &steerRes)
	if steerRes.Status != wire.SteerStatusSteered {
		t.Fatalf("expected steered, got %s", steerRes.Status)
	}

	mustReadEventType(t, clientTrans, "ContentPart")
	mustReadEventType(t, clientTrans, "TurnEnd")
	var res wire.PromptResult
	readRes(t, clientTrans, "prompt1", &res)
	if res.Status != wire.PromptStatusFinished {
		t.Fatalf("expected finished, got %s", res.Status)
	}
}

func TestCancelDuringTurn(t *testing.T) {
	t.Parallel()
	clientTrans, cleanup := startServer(t, &cancelAgent{})
	defer cleanup()

	writeReq(t, clientTrans, "initialize", "init1", wire.InitializeParams{ProtocolVersion: wire.WireProtocolVersion})
	mustReadResponse(t, clientTrans, "init1")

	writeReq(t, clientTrans, "prompt", "prompt1", wire.PromptParams{UserInput: wire.UserInput{Text: "go"}})
	mustReadEventType(t, clientTrans, "TurnBegin")

	writeReq(t, clientTrans, "cancel", "cancel1", wire.CancelParams{})
	mustReadEventType(t, clientTrans, "TurnEnd")

	responses := readResponsesInAnyOrder(t, clientTrans, "cancel1", "prompt1")
	var cancelRes wire.CancelResult
	if err := json.Unmarshal(responses["cancel1"].Result, &cancelRes); err != nil {
		t.Fatalf("unmarshal cancel result: %v", err)
	}
	var res wire.PromptResult
	if err := json.Unmarshal(responses["prompt1"].Result, &res); err != nil {
		t.Fatalf("unmarshal prompt result: %v", err)
	}
	if res.Status != wire.PromptStatusCancelled {
		t.Fatalf("expected cancelled, got %s", res.Status)
	}
}

func TestMaxStepsReached(t *testing.T) {
	t.Parallel()
	clientTrans, cleanup := startServer(t, &maxStepsAgent{})
	defer cleanup()

	writeReq(t, clientTrans, "initialize", "init1", wire.InitializeParams{ProtocolVersion: wire.WireProtocolVersion})
	mustReadResponse(t, clientTrans, "init1")

	writeReq(t, clientTrans, "prompt", "prompt1", wire.PromptParams{UserInput: wire.UserInput{Text: "go"}})
	mustReadEventType(t, clientTrans, "TurnBegin")
	mustReadEventType(t, clientTrans, "TurnEnd")
	var res wire.PromptResult
	readRes(t, clientTrans, "prompt1", &res)
	if res.Status != wire.PromptStatusMaxStepsReached {
		t.Fatalf("expected max_steps_reached, got %s", res.Status)
	}
}

func TestLegacyMode(t *testing.T) {
	t.Parallel()
	clientTrans, cleanup := startServer(t, &happyAgent{})
	defer cleanup()

	writeReq(t, clientTrans, "prompt", "prompt1", wire.PromptParams{UserInput: wire.UserInput{Text: "legacy"}})
	mustReadEventType(t, clientTrans, "TurnBegin")
	mustReadEventType(t, clientTrans, "StepBegin")
	mustReadEventType(t, clientTrans, "ContentPart")
	mustReadEventType(t, clientTrans, "TurnEnd")
	var res wire.PromptResult
	readRes(t, clientTrans, "prompt1", &res)
	if res.Status != wire.PromptStatusFinished {
		t.Fatalf("expected finished, got %s", res.Status)
	}
}

func TestUnknownMethod(t *testing.T) {
	t.Parallel()
	clientTrans, cleanup := startServer(t, &happyAgent{})
	defer cleanup()

	writeReq(t, clientTrans, "unknown", "unk1", struct{}{})
	raw := mustFindResponse(t, clientTrans, "unk1")
	if raw.Error == nil {
		t.Fatalf("expected error")
	}
	if raw.Error.Code != -32601 {
		t.Fatalf("expected code -32601, got %d", raw.Error.Code)
	}
}

func TestPromptDuringPrompt(t *testing.T) {
	t.Parallel()
	agent := &blockingAgent{blocked: make(chan struct{})}
	clientTrans, cleanup := startServer(t, agent)
	defer cleanup()

	writeReq(t, clientTrans, "initialize", "init1", wire.InitializeParams{ProtocolVersion: wire.WireProtocolVersion})
	mustReadResponse(t, clientTrans, "init1")

	writeReq(t, clientTrans, "prompt", "prompt1", wire.PromptParams{UserInput: wire.UserInput{Text: "go"}})
	mustReadEventType(t, clientTrans, "TurnBegin")

	writeReq(t, clientTrans, "prompt", "prompt2", wire.PromptParams{UserInput: wire.UserInput{Text: "again"}})
	raw := mustFindResponse(t, clientTrans, "prompt2")
	if raw.Error == nil {
		t.Fatalf("expected error")
	}
	if raw.Error.Code != -32000 {
		t.Fatalf("expected code -32000, got %d", raw.Error.Code)
	}
	if raw.Error.Message != "A turn is already in progress" {
		t.Fatalf("expected turn in progress message, got %q", raw.Error.Message)
	}

	close(agent.blocked)
	mustReadEventType(t, clientTrans, "TurnEnd")
	var res wire.PromptResult
	readRes(t, clientTrans, "prompt1", &res)
	if res.Status != wire.PromptStatusFinished {
		t.Fatalf("expected finished, got %s", res.Status)
	}
}

func TestTransportAbortMidTurn(t *testing.T) {
	t.Parallel()
	clientTrans, serverTrans := wire.NewChannelTransportPair()
	srv := server.New(serverTrans, &cancelAgent{})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = srv.Serve(ctx)
	}()

	writeReq(t, clientTrans, "initialize", "init1", wire.InitializeParams{ProtocolVersion: wire.WireProtocolVersion})
	mustReadResponse(t, clientTrans, "init1")

	writeReq(t, clientTrans, "prompt", "prompt1", wire.PromptParams{UserInput: wire.UserInput{Text: "go"}})
	mustReadEventType(t, clientTrans, "TurnBegin")

	_ = clientTrans.Close()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("server did not shut down")
	}
	cancel()
}

func TestPanicInPrompt(t *testing.T) {
	t.Parallel()
	clientTrans, cleanup := startServer(t, &panicAgent{})
	defer cleanup()

	writeReq(t, clientTrans, "initialize", "init1", wire.InitializeParams{ProtocolVersion: wire.WireProtocolVersion})
	mustReadResponse(t, clientTrans, "init1")

	writeReq(t, clientTrans, "prompt", "prompt1", wire.PromptParams{UserInput: wire.UserInput{Text: "go"}})
	mustReadEventType(t, clientTrans, "TurnBegin")
	mustReadEventType(t, clientTrans, "TurnEnd")
	raw := mustFindResponse(t, clientTrans, "prompt1")
	if raw.Error == nil {
		t.Fatalf("expected error")
	}
	if raw.Error.Code != -32603 {
		t.Fatalf("expected code -32603, got %d", raw.Error.Code)
	}

	// Server should continue: a second prompt works.
	writeReq(t, clientTrans, "prompt", "prompt2", wire.PromptParams{UserInput: wire.UserInput{Text: "again"}})
	mustReadEventType(t, clientTrans, "TurnBegin")
	mustReadEventType(t, clientTrans, "TurnEnd")
	raw = mustFindResponse(t, clientTrans, "prompt2")
	if raw.Error == nil || raw.Error.Code != -32603 {
		t.Fatalf("expected repeated -32603 error, got %v", raw.Error)
	}
}

func TestSetPlanModeGatedByCapability(t *testing.T) {
	t.Parallel()
	clientTrans, cleanup := startServer(t, &happyAgent{})
	defer cleanup()

	writeReq(t, clientTrans, "initialize", "init1", wire.InitializeParams{
		ProtocolVersion: wire.WireProtocolVersion,
		Capabilities:    &wire.ClientCapabilities{},
	})
	mustReadResponse(t, clientTrans, "init1")

	writeReq(t, clientTrans, "set_plan_mode", "spm1", wire.SetPlanModeParams{Enabled: true})
	raw := mustFindResponse(t, clientTrans, "spm1")
	if raw.Error == nil {
		t.Fatalf("expected error")
	}
	if raw.Error.Code != -32000 {
		t.Fatalf("expected code -32000, got %d", raw.Error.Code)
	}
	if raw.Error.Message != "Plan mode is not supported" {
		t.Fatalf("expected plan mode not supported, got %q", raw.Error.Message)
	}
}

func TestSetPlanModeOptionalInterface(t *testing.T) {
	t.Parallel()
	supports := true
	clientTrans, cleanup := startServer(t, &planModeAgent{enabled: &supports})
	defer cleanup()

	writeReq(t, clientTrans, "initialize", "init1", wire.InitializeParams{
		ProtocolVersion: wire.WireProtocolVersion,
		Capabilities:    &wire.ClientCapabilities{SupportsPlanMode: &supports},
	})
	mustReadResponse(t, clientTrans, "init1")

	writeReq(t, clientTrans, "set_plan_mode", "spm1", wire.SetPlanModeParams{Enabled: true})
	var res wire.SetPlanModeResult
	readRes(t, clientTrans, "spm1", &res)
	if res.Status != wire.SetPlanModeStatusOk {
		t.Fatalf("expected ok, got %s", res.Status)
	}
	if !res.PlanMode {
		t.Fatalf("expected plan_mode true")
	}
}

func TestReplayOptionalInterface(t *testing.T) {
	t.Parallel()
	clientTrans, cleanup := startServer(t, &replayAgent{})
	defer cleanup()

	writeReq(t, clientTrans, "initialize", "init1", wire.InitializeParams{ProtocolVersion: wire.WireProtocolVersion})
	mustReadResponse(t, clientTrans, "init1")

	writeReq(t, clientTrans, "replay", "replay1", wire.ReplayParams{})
	var res wire.ReplayResult
	readRes(t, clientTrans, "replay1", &res)
	if res.Status != wire.ReplayStatusFinished {
		t.Fatalf("expected finished, got %s", res.Status)
	}
}

type planModeAgent struct {
	enabled *bool
}

func (a *planModeAgent) Prompt(ctx context.Context, input wire.UserInput, turn server.Turn) (wire.PromptResult, error) {
	return wire.PromptResult{Status: wire.PromptStatusFinished}, nil
}

func (a *planModeAgent) SetPlanMode(ctx context.Context, enabled bool, emitter server.Emitter) (wire.SetPlanModeResult, error) {
	return wire.SetPlanModeResult{Status: wire.SetPlanModeStatusOk, PlanMode: enabled}, nil
}

type replayAgent struct{}

func (a *replayAgent) Prompt(ctx context.Context, input wire.UserInput, turn server.Turn) (wire.PromptResult, error) {
	return wire.PromptResult{Status: wire.PromptStatusFinished}, nil
}

func (a *replayAgent) Replay(ctx context.Context, emitter server.Emitter) (wire.ReplayResult, error) {
	return wire.ReplayResult{Status: wire.ReplayStatusFinished, Events: 1, Requests: 0}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func startServer(t *testing.T, agent server.Agent, opts ...server.Option) (wire.Transport, func()) {
	t.Helper()
	clientTrans, serverTrans := wire.NewChannelTransportPair()
	srv := server.New(serverTrans, agent, opts...)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = srv.Serve(ctx)
	}()
	return clientTrans, func() {
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Logf("server did not shut down in cleanup")
		}
		_ = clientTrans.Close()
		_ = serverTrans.Close()
	}
}

func writeReq(t *testing.T, tr wire.Transport, method, id string, params any) {
	t.Helper()
	b, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}
	req := wire.JSONRPCRequest[json.RawMessage]{
		JSONRPC: "2.0",
		Method:  method,
		ID:      id,
		Params:  b,
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := tr.WriteLine(ctx, string(data)); err != nil {
		t.Fatalf("write request: %v", err)
	}
}

func writeResponse(t *testing.T, tr wire.Transport, id string, result any) {
	t.Helper()
	data, err := json.Marshal(wire.JSONRPCSuccessResponse[any]{JSONRPC: "2.0", ID: id, Result: result})
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := tr.WriteLine(ctx, string(data)); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

func readRes(t *testing.T, tr wire.Transport, id string, result any) {
	t.Helper()
	raw := mustFindResponse(t, tr, id)
	if raw.Error != nil {
		t.Fatalf("unexpected error: %d %s", raw.Error.Code, raw.Error.Message)
	}
	if err := json.Unmarshal(raw.Result, result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
}

func mustReadResponse(t *testing.T, tr wire.Transport, id string) *wire.RawWireMessage {
	t.Helper()
	raw := mustFindResponse(t, tr, id)
	if raw.Error != nil {
		t.Fatalf("unexpected error: %d %s", raw.Error.Code, raw.Error.Message)
	}
	return raw
}

func mustFindResponse(t *testing.T, tr wire.Transport, id string) *wire.RawWireMessage {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		ctx, cancel := context.WithDeadline(context.Background(), deadline)
		line, err := tr.ReadLine(ctx)
		cancel()
		if err != nil {
			t.Fatalf("read line: %v", err)
		}
		raw := new(wire.RawWireMessage)
		if err := json.Unmarshal([]byte(line), raw); err != nil {
			t.Fatalf("unmarshal raw: %v", err)
		}
		if raw.ID == id && raw.Method == "" {
			return raw
		}
	}
}

func mustReadEventType(t *testing.T, tr wire.Transport, wantType string) wire.Event {
	t.Helper()
	raw := readRaw(t, tr)
	if raw.Method != "event" {
		t.Fatalf("expected event, got method %q", raw.Method)
	}
	ev, err := wire.ParseEvent(raw.Params)
	if err != nil {
		t.Fatalf("parse event: %v", err)
	}
	if wire.TypeName(ev) != wantType {
		t.Fatalf("expected event type %s, got %s", wantType, wire.TypeName(ev))
	}
	return ev
}

func mustReadRequest(t *testing.T, tr wire.Transport) (string, wire.Request) {
	t.Helper()
	raw := readRaw(t, tr)
	if raw.Method != "request" {
		t.Fatalf("expected request, got method %q", raw.Method)
	}
	req, err := wire.ParseRequest(raw.Params)
	if err != nil {
		t.Fatalf("parse request: %v", err)
	}
	return raw.ID, req
}

func readRaw(t *testing.T, tr wire.Transport) *wire.RawWireMessage {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	line, err := tr.ReadLine(ctx)
	if err != nil {
		t.Fatalf("read line: %v", err)
	}
	raw := new(wire.RawWireMessage)
	if err := json.Unmarshal([]byte(line), raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	return raw
}

func readResponsesInAnyOrder(t *testing.T, tr wire.Transport, ids ...string) map[string]*wire.RawWireMessage {
	t.Helper()
	result := make(map[string]*wire.RawWireMessage, len(ids))
	deadline := time.Now().Add(5 * time.Second)
	for len(result) < len(ids) {
		ctx, cancel := context.WithDeadline(context.Background(), deadline)
		line, err := tr.ReadLine(ctx)
		cancel()
		if err != nil {
			t.Fatalf("read line: %v", err)
		}
		raw := new(wire.RawWireMessage)
		if err := json.Unmarshal([]byte(line), raw); err != nil {
			t.Fatalf("unmarshal raw: %v", err)
		}
		if raw.Method != "" {
			continue
		}
		for _, id := range ids {
			if raw.ID == id {
				result[id] = raw
			}
		}
	}
	return result
}
