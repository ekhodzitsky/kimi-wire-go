package wire

import "github.com/ekhodzitsky/kimi-wire/protocol"

// Content types
type UserInput = protocol.UserInput
type ContentPart = protocol.ContentPart
type TextPart = protocol.TextPart
type ThinkPart = protocol.ThinkPart
type ImageURLPart = protocol.ImageURLPart
type AudioURLPart = protocol.AudioURLPart
type VideoURLPart = protocol.VideoURLPart
type MediaURL = protocol.MediaURL
type DisplayBlockType = protocol.DisplayBlockType
type DisplayBlock = protocol.DisplayBlock
type TodoDisplayItem = protocol.TodoDisplayItem
type TodoStatus = protocol.TodoStatus
type ToolReturnValue = protocol.ToolReturnValue
type ToolOutput = protocol.ToolOutput

const ContentPartTypeText = protocol.ContentPartTypeText
const ContentPartTypeThink = protocol.ContentPartTypeThink
const ContentPartTypeImageURL = protocol.ContentPartTypeImageURL
const ContentPartTypeAudioURL = protocol.ContentPartTypeAudioURL
const ContentPartTypeVideoURL = protocol.ContentPartTypeVideoURL

const DisplayBlockTypeBrief = protocol.DisplayBlockTypeBrief
const DisplayBlockTypeDiff = protocol.DisplayBlockTypeDiff
const DisplayBlockTypeTodo = protocol.DisplayBlockTypeTodo
const DisplayBlockTypeShell = protocol.DisplayBlockTypeShell
const DisplayBlockTypeUnknown = protocol.DisplayBlockTypeUnknown

const TodoStatusPending = protocol.TodoStatusPending
const TodoStatusInProgress = protocol.TodoStatusInProgress
const TodoStatusDone = protocol.TodoStatusDone

var NewDisplayBlockBrief = protocol.NewDisplayBlockBrief
var NewDisplayBlockDiff = protocol.NewDisplayBlockDiff
var NewDisplayBlockTodo = protocol.NewDisplayBlockTodo
var NewDisplayBlockShell = protocol.NewDisplayBlockShell

// Event types
type Event = protocol.Event
type TurnBeginEvent = protocol.TurnBeginEvent
type TurnEndEvent = protocol.TurnEndEvent
type StepBeginEvent = protocol.StepBeginEvent
type StepInterruptedEvent = protocol.StepInterruptedEvent
type StepRetryEvent = protocol.StepRetryEvent
type CompactionBeginEvent = protocol.CompactionBeginEvent
type CompactionEndEvent = protocol.CompactionEndEvent
type StatusUpdateEvent = protocol.StatusUpdateEvent
type TokenUsage = protocol.TokenUsage
type ContentPartEvent = protocol.ContentPartEvent
type ToolCallEvent = protocol.ToolCallEvent
type ToolCallFunction = protocol.ToolCallFunction
type ToolCallPartEvent = protocol.ToolCallPartEvent
type ToolResultEvent = protocol.ToolResultEvent
type ApprovalResponseKind = protocol.ApprovalResponseKind
type ApprovalResponseEvent = protocol.ApprovalResponseEvent
type SubagentEventPayload = protocol.SubagentEventPayload
type SubagentEvent = protocol.SubagentEvent
type SteerInputEvent = protocol.SteerInputEvent
type BtwBeginEvent = protocol.BtwBeginEvent
type BtwEndEvent = protocol.BtwEndEvent
type PlanDisplayEvent = protocol.PlanDisplayEvent
type HookAction = protocol.HookAction
type HookTriggeredEvent = protocol.HookTriggeredEvent
type HookResolvedEvent = protocol.HookResolvedEvent

const ApprovalResponseKindApprove = protocol.ApprovalResponseKindApprove
const ApprovalResponseKindApproveForSession = protocol.ApprovalResponseKindApproveForSession
const ApprovalResponseKindReject = protocol.ApprovalResponseKindReject

const HookActionAllow = protocol.HookActionAllow
const HookActionBlock = protocol.HookActionBlock

var TypeName = protocol.TypeName
var MarshalEvent = protocol.MarshalEvent
var ParseEvent = protocol.ParseEvent

// JSON-RPC types
type RawWireMessage = protocol.RawWireMessage
// JSONRPCRequest is a typed JSON-RPC 2.0 request.
type JSONRPCRequest[T any] struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	ID      string `json:"id"`
	Params  T      `json:"params"`
}

// JSONRPCSuccessResponse is a typed JSON-RPC 2.0 success response.
type JSONRPCSuccessResponse[T any] struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id"`
	Result  T      `json:"result"`
}
type JSONRPCErrorResponse = protocol.JSONRPCErrorResponse
type JSONRPCError = protocol.JSONRPCError

const MethodNotFound = protocol.MethodNotFound

// Method / wire protocol types
const WireProtocolVersion = protocol.WireProtocolVersion
const WireProtocolLegacyVersion = protocol.WireProtocolLegacyVersion

type InitializeParams = protocol.InitializeParams
type ClientInfo = protocol.ClientInfo
type ClientCapabilities = protocol.ClientCapabilities
type WireHookSubscription = protocol.WireHookSubscription
type ExternalTool = protocol.ExternalTool
type InitializeResult = protocol.InitializeResult
type ServerInfo = protocol.ServerInfo
type SlashCommandInfo = protocol.SlashCommandInfo
type ExternalToolsResult = protocol.ExternalToolsResult
type RejectedExternalTool = protocol.RejectedExternalTool
type ServerCapabilities = protocol.ServerCapabilities
type HooksInfo = protocol.HooksInfo
type PromptParams = protocol.PromptParams
type PromptResult = protocol.PromptResult
type PromptStatus = protocol.PromptStatus
type ReplayParams = protocol.ReplayParams
type ReplayResult = protocol.ReplayResult
type ReplayStatus = protocol.ReplayStatus
type SteerParams = protocol.SteerParams
type SteerResult = protocol.SteerResult
type SteerStatus = protocol.SteerStatus
type SetPlanModeParams = protocol.SetPlanModeParams
type SetPlanModeResult = protocol.SetPlanModeResult
type SetPlanModeStatus = protocol.SetPlanModeStatus
type CancelParams = protocol.CancelParams
type CancelResult = protocol.CancelResult

const PromptStatusFinished = protocol.PromptStatusFinished
const PromptStatusCancelled = protocol.PromptStatusCancelled
const PromptStatusMaxStepsReached = protocol.PromptStatusMaxStepsReached
const PromptStatusPending = protocol.PromptStatusPending
const PromptStatusUnexpectedEof = protocol.PromptStatusUnexpectedEof

const ReplayStatusFinished = protocol.ReplayStatusFinished
const ReplayStatusCancelled = protocol.ReplayStatusCancelled

const SteerStatusSteered = protocol.SteerStatusSteered

const SetPlanModeStatusOk = protocol.SetPlanModeStatusOk

// Request types
type Request = protocol.Request
type ApprovalRequest = protocol.ApprovalRequest
type SourceKind = protocol.SourceKind
type ToolCallRequest = protocol.ToolCallRequest
type QuestionRequest = protocol.QuestionRequest
type QuestionItem = protocol.QuestionItem
type QuestionOption = protocol.QuestionOption
type HookRequest = protocol.HookRequest
type ApprovalResponse = protocol.ApprovalResponse
type ToolCallResponse = protocol.ToolCallResponse
type QuestionResponse = protocol.QuestionResponse
type HookResponse = protocol.HookResponse

const SourceKindForegroundTurn = protocol.SourceKindForegroundTurn
const SourceKindBackgroundAgent = protocol.SourceKindBackgroundAgent

var Kind = protocol.Kind
var MarshalRequest = protocol.MarshalRequest
var ParseRequest = protocol.ParseRequest
