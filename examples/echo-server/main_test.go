package main

import (
	"context"
	"testing"
	"time"

	"github.com/ekhodzitsky/kimi-wire"
)

type testHandler struct {
	t       *testing.T
	events  []wire.Event
	approve bool
}

func (h *testHandler) HandleEvent(ctx context.Context, event wire.Event) error {
	h.events = append(h.events, event)
	return nil
}

func (h *testHandler) HandleRequest(ctx context.Context, req wire.Request) (any, error) {
	switch r := req.(type) {
	case wire.ApprovalRequest:
		response := wire.ApprovalResponseKindReject
		if h.approve {
			response = wire.ApprovalResponseKindApprove
		}
		return wire.ApprovalResponse{RequestID: r.ID, Response: response}, nil
	case wire.ToolCallRequest:
		return wire.ToolCallResponse{
			ToolCallID: r.ID,
			ReturnValue: wire.ToolReturnValue{
				IsError: false,
				Output:  wire.ToolOutput{Text: "Hello from tool"},
				Message: "greeting returned",
			},
		}, nil
	default:
		h.t.Fatalf("unexpected request type: %T", req)
		return nil, nil
	}
}

func TestEchoServerPrompt(t *testing.T) {
	cases := []struct {
		name           string
		approve        bool
		wantStatus     wire.PromptStatus
		wantContent    bool
		wantEventTypes []string
	}{
		{
			name:           "approved echoes with greeting",
			approve:        true,
			wantStatus:     wire.PromptStatusFinished,
			wantContent:    true,
			wantEventTypes: []string{"TurnBegin", "ContentPart", "TurnEnd"},
		},
		{
			name:           "rejected cancels turn",
			approve:        false,
			wantStatus:     wire.PromptStatusCancelled,
			wantContent:    false,
			wantEventTypes: []string{"TurnBegin", "TurnEnd"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			serverTr, clientTr := wire.NewChannelTransportPair()
			defer func() {
				if err := serverTr.Close(); err != nil {
					t.Logf("server transport close: %v", err)
				}
			}()
			defer func() {
				if err := clientTr.Close(); err != nil {
					t.Logf("client transport close: %v", err)
				}
			}()

			server := wire.NewServer(serverTr, &echoAgent{}, wire.WithServerInfo("echo-server", "0.1.0"))
			client := wire.NewClient(clientTr)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			go func() {
				_ = server.Serve(ctx)
			}()

			handler := &testHandler{t: t, approve: tc.approve}
			dispatchDone := make(chan struct{})
			go func() {
				defer close(dispatchDone)
				_ = client.Dispatch(ctx, handler)
			}()

			if _, err := client.Initialize(ctx, wire.InitializeParams{
				ProtocolVersion: wire.WireProtocolVersion,
			}); err != nil {
				t.Fatalf("initialize: %v", err)
			}

			result, err := client.Prompt(ctx, wire.UserInput{Text: "world"})
			if err != nil {
				t.Fatalf("prompt: %v", err)
			}
			if result.Status != tc.wantStatus {
				t.Fatalf("status: got %q, want %q", result.Status, tc.wantStatus)
			}

			_ = client.Shutdown(ctx)
			<-dispatchDone

			var gotEventTypes []string
			var gotContent string
			for _, ev := range handler.events {
				gotEventTypes = append(gotEventTypes, wire.TypeName(ev))
				if cp, ok := ev.(wire.ContentPartEvent); ok && cp.Part.Text != nil {
					gotContent = cp.Part.Text.Text
				}
			}

			if len(gotEventTypes) != len(tc.wantEventTypes) {
				t.Fatalf("events: got %v, want %v", gotEventTypes, tc.wantEventTypes)
			}
			for i, want := range tc.wantEventTypes {
				if gotEventTypes[i] != want {
					t.Fatalf("event[%d]: got %q, want %q", i, gotEventTypes[i], want)
				}
			}
			if tc.wantContent && gotContent == "" {
				t.Fatal("expected non-empty content part")
			}
		})
	}
}
