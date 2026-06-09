package wire

import (
	"encoding/json"
	"fmt"
)

// Event is the interface for all wire events.
type Event interface {
	eventType() string
}

// TurnBeginEvent signals a new turn.
type TurnBeginEvent struct {
	UserInput UserInput `json:"user_input"`
}

func (TurnBeginEvent) eventType() string { return "TurnBegin" }

// TurnEndEvent signals the end of a turn.
type TurnEndEvent struct{}

func (TurnEndEvent) eventType() string { return "TurnEnd" }

// StepBeginEvent signals a new step.
type StepBeginEvent struct {
	N uint32 `json:"n"`
}

func (StepBeginEvent) eventType() string { return "StepBegin" }

// StepInterruptedEvent signals a step interruption.
type StepInterruptedEvent struct{}

func (StepInterruptedEvent) eventType() string { return "StepInterrupted" }

// StepRetryEvent signals a step retry.
type StepRetryEvent struct {
	N           uint32  `json:"n"`
	NextAttempt uint32  `json:"next_attempt"`
	MaxAttempts uint32  `json:"max_attempts"`
	WaitS       uint32  `json:"wait_s"`
	ErrorType   string  `json:"error_type"`
	StatusCode  *uint32 `json:"status_code,omitempty"`
}

func (StepRetryEvent) eventType() string { return "StepRetry" }

// CompactionBeginEvent signals compaction start.
type CompactionBeginEvent struct{}

func (CompactionBeginEvent) eventType() string { return "CompactionBegin" }

// CompactionEndEvent signals compaction end.
type CompactionEndEvent struct{}

func (CompactionEndEvent) eventType() string { return "CompactionEnd" }

// StatusUpdateEvent carries server status.
type StatusUpdateEvent struct {
	ContextUsage     *float64    `json:"context_usage,omitempty"`
	ContextTokens    *uint64     `json:"context_tokens,omitempty"`
	MaxContextTokens *uint64     `json:"max_context_tokens,omitempty"`
	TokenUsage       *TokenUsage `json:"token_usage,omitempty"`
	MessageID        string      `json:"message_id,omitempty"`
	PlanMode         *bool       `json:"plan_mode,omitempty"`
}

func (StatusUpdateEvent) eventType() string { return "StatusUpdate" }

// TokenUsage is a token usage breakdown.
type TokenUsage struct {
	InputOther         uint64 `json:"input_other"`
	Output             uint64 `json:"output"`
	InputCacheRead     uint64 `json:"input_cache_read"`
	InputCacheCreation uint64 `json:"input_cache_creation"`
}

// ContentPartEvent wraps a ContentPart as an event.
// The payload is the ContentPart itself (no extra wrapper).
type ContentPartEvent struct {
	Part ContentPart
}

func (ContentPartEvent) eventType() string { return "ContentPart" }

// ToolCallEvent is a tool call from the model.
type ToolCallEvent struct {
	ID       string          `json:"id"`
	Function ToolCallFunction `json:"function"`
	Extras   json.RawMessage  `json:"extras,omitempty"`
}

func (ToolCallEvent) eventType() string { return "ToolCall" }

// ToolCallFunction is the function part of a tool call.
type ToolCallFunction struct {
	Name      string  `json:"name"`
	Arguments *string `json:"arguments,omitempty"`
}

// ToolCallPartEvent is a partial tool call.
type ToolCallPartEvent struct {
	ArgumentsPart *string `json:"arguments_part,omitempty"`
}

func (ToolCallPartEvent) eventType() string { return "ToolCallPart" }

// ToolResultEvent is the result of a tool execution.
type ToolResultEvent struct {
	ToolCallID  string          `json:"tool_call_id"`
	ReturnValue ToolReturnValue `json:"return_value"`
}

func (ToolResultEvent) eventType() string { return "ToolResult" }

// ApprovalResponseKind is the client's response to an approval request.
type ApprovalResponseKind string

const (
	ApprovalResponseKindApprove           ApprovalResponseKind = "approve"
	ApprovalResponseKindApproveForSession ApprovalResponseKind = "approve_for_session"
	ApprovalResponseKindReject            ApprovalResponseKind = "reject"
)

// ApprovalResponseEvent is a response to an approval request.
type ApprovalResponseEvent struct {
	RequestID string               `json:"request_id"`
	Response  ApprovalResponseKind `json:"response"`
	Feedback  string               `json:"feedback,omitempty"`
}

func (ApprovalResponseEvent) eventType() string { return "ApprovalResponse" }

// SubagentEventPayload is the payload of a SubagentEvent.
type SubagentEventPayload struct {
	TypeName string          `json:"type"`
	Payload  json.RawMessage `json:"payload"`
}

// SubagentEvent is an event from a subagent.
type SubagentEvent struct {
	ParentToolCallID *string              `json:"parent_tool_call_id,omitempty"`
	AgentID          *string              `json:"agent_id,omitempty"`
	SubagentType     *string              `json:"subagent_type,omitempty"`
	Event            SubagentEventPayload `json:"event"`
}

func (SubagentEvent) eventType() string { return "SubagentEvent" }

// SteerInputEvent is additional user input steering the current turn.
type SteerInputEvent struct {
	UserInput UserInput `json:"user_input"`
}

func (SteerInputEvent) eventType() string { return "SteerInput" }

// BtwBeginEvent signals a side question has started.
type BtwBeginEvent struct {
	ID       string `json:"id"`
	Question string `json:"question"`
}

func (BtwBeginEvent) eventType() string { return "BtwBegin" }

// BtwEndEvent signals a side question has finished.
type BtwEndEvent struct {
	ID       string  `json:"id"`
	Response *string `json:"response,omitempty"`
	Error    *string `json:"error,omitempty"`
}

func (BtwEndEvent) eventType() string { return "BtwEnd" }

// PlanDisplayEvent carries plan display content.
type PlanDisplayEvent struct {
	Content  string `json:"content"`
	FilePath string `json:"file_path"`
}

func (PlanDisplayEvent) eventType() string { return "PlanDisplay" }

// HookAction is the action taken by a hook.
type HookAction string

const (
	HookActionAllow HookAction = "allow"
	HookActionBlock HookAction = "block"
)

// HookTriggeredEvent signals a hook was triggered.
type HookTriggeredEvent struct {
	Event     string `json:"event"`
	Target    string `json:"target"`
	HookCount uint32 `json:"hook_count"`
}

func (HookTriggeredEvent) eventType() string { return "HookTriggered" }

// HookResolvedEvent signals a hook was resolved.
type HookResolvedEvent struct {
	Event      string     `json:"event"`
	Target     string     `json:"target"`
	Action     HookAction `json:"action"`
	Reason     string     `json:"reason"`
	DurationMs uint64     `json:"duration_ms"`
}

func (HookResolvedEvent) eventType() string { return "HookResolved" }

// MarshalEvent serializes an Event to the wire envelope format.
func MarshalEvent(e Event) ([]byte, error) {
	var payload json.RawMessage
	var err error

	switch v := e.(type) {
	case ToolCallEvent:
		// The payload must include the inner `type: "function"` discriminator.
		payload, err = json.Marshal(struct {
			Type     string           `json:"type"`
			ID       string           `json:"id"`
			Function ToolCallFunction `json:"function"`
			Extras   json.RawMessage  `json:"extras,omitempty"`
		}{
			Type:     "function",
			ID:       v.ID,
			Function: v.Function,
			Extras:   v.Extras,
		})
	default:
		payload, err = json.Marshal(e)
	}
	if err != nil {
		return nil, err
	}

	return json.Marshal(map[string]any{
		"type":    e.eventType(),
		"payload": payload,
	})
}

// ParseEvent deserializes a JSON envelope into a concrete Event.
func ParseEvent(data []byte) (Event, error) {
	var env struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	switch env.Type {
	case "TurnBegin":
		var p TurnBeginEvent
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return nil, err
		}
		return p, nil
	case "TurnEnd":
		return TurnEndEvent{}, nil
	case "StepBegin":
		var p StepBeginEvent
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return nil, err
		}
		return p, nil
	case "StepInterrupted":
		return StepInterruptedEvent{}, nil
	case "StepRetry":
		var p StepRetryEvent
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return nil, err
		}
		return p, nil
	case "CompactionBegin":
		return CompactionBeginEvent{}, nil
	case "CompactionEnd":
		return CompactionEndEvent{}, nil
	case "StatusUpdate":
		var p StatusUpdateEvent
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return nil, err
		}
		return p, nil
	case "ContentPart":
		var part ContentPart
		if err := json.Unmarshal(env.Payload, &part); err != nil {
			return nil, err
		}
		return ContentPartEvent{Part: part}, nil
	case "ToolCall":
		var p ToolCallEvent
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return nil, err
		}
		return p, nil
	case "ToolCallPart":
		var p ToolCallPartEvent
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return nil, err
		}
		return p, nil
	case "ToolResult":
		var p ToolResultEvent
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return nil, err
		}
		return p, nil
	case "ApprovalResponse":
		var p ApprovalResponseEvent
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return nil, err
		}
		return p, nil
	case "SubagentEvent":
		var p SubagentEvent
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return nil, err
		}
		return p, nil
	case "SteerInput":
		var p SteerInputEvent
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return nil, err
		}
		return p, nil
	case "BtwBegin":
		var p BtwBeginEvent
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return nil, err
		}
		return p, nil
	case "BtwEnd":
		var p BtwEndEvent
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return nil, err
		}
		return p, nil
	case "PlanDisplay":
		var p PlanDisplayEvent
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return nil, err
		}
		return p, nil
	case "HookTriggered":
		var p HookTriggeredEvent
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return nil, err
		}
		return p, nil
	case "HookResolved":
		var p HookResolvedEvent
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return nil, err
		}
		return p, nil
	default:
		return nil, fmt.Errorf("unknown event type: %s", env.Type)
	}
}
