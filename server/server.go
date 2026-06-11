package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sync"
	"time"

	"github.com/ekhodzitsky/kimi-wire/internal/redact"
	"github.com/ekhodzitsky/kimi-wire/protocol"
	"github.com/ekhodzitsky/kimi-wire/transport"
)

// Server is a Wire protocol server.
type Server struct {
	transport      transport.Transport
	agent          Agent
	info           protocol.ServerInfo
	slashCmds      []protocol.SlashCommandInfo
	toolValidator  func(protocol.ExternalTool) error
	supportedHooks []string
	defaultTimeout time.Duration
	logf           func(string, ...any)

	mu            sync.Mutex
	handshakeDone bool
	negotiated    string
	clientCaps    protocol.ClientCapabilities
	serverCaps    protocol.ServerCapabilities
	externalTools []protocol.ExternalTool
	hooks         map[string]*hookSubscription
	activeTurn    *turn
	activeOp      activeOperation

	requestCounter uint64
	pending        map[string]chan *protocol.RawWireMessage

	serveCtx    context.Context
	cancelServe context.CancelFunc
	readDone    chan struct{}
	serveDone   chan struct{}
	dispatchCh  chan *protocol.RawWireMessage
	readErr     error
}

type activeOperation struct {
	cancel context.CancelFunc
	done   <-chan struct{}
}

type hookSubscription struct {
	id      string
	event   string
	matcher *regexp.Regexp
	timeout time.Duration
}

// Agent produces prompt results during a turn.
type Agent interface {
	Prompt(ctx context.Context, input protocol.UserInput, turn Turn) (protocol.PromptResult, error)
}

// Steerer is an optional agent capability that receives steering input.
type Steerer interface {
	Steer(ctx context.Context, input protocol.UserInput) error
}

// Replayer is an optional agent capability that replays session events.
type Replayer interface {
	Replay(ctx context.Context, emitter Emitter) (protocol.ReplayResult, error)
}

// PlanModeSwitcher is an optional agent capability that toggles plan mode.
type PlanModeSwitcher interface {
	SetPlanMode(ctx context.Context, enabled bool, emitter Emitter) (protocol.SetPlanModeResult, error)
}

// New creates a Server backed by the given transport and agent.
func New(transport transport.Transport, agent Agent, opts ...Option) *Server {
	if transport == nil {
		panic("server: nil transport")
	}
	if agent == nil {
		panic("server: nil agent")
	}
	s := &Server{
		transport:      transport,
		agent:          agent,
		info:           protocol.ServerInfo{Name: "kimi-wire-server", Version: "0.3.0"},
		slashCmds:      []protocol.SlashCommandInfo{},
		toolValidator:  func(protocol.ExternalTool) error { return nil },
		supportedHooks: []string{},
		defaultTimeout: 0,
		logf:           func(string, ...any) {},
		pending:        make(map[string]chan *protocol.RawWireMessage),
		hooks:          make(map[string]*hookSubscription),
		negotiated:     protocol.WireProtocolLegacyVersion,
		dispatchCh:     make(chan *protocol.RawWireMessage, 1024),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Serve runs the server until the transport closes or ctx is cancelled.
func (s *Server) Serve(ctx context.Context) error {
	s.serveCtx, s.cancelServe = context.WithCancel(ctx)
	defer s.cancelServe()
	s.readDone = make(chan struct{})
	s.serveDone = make(chan struct{})

	go s.readLoop()
	go s.serveLoop()

	<-s.readDone
	s.cancelServe()
	err := s.readErr
	if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, io.EOF) {
		s.closePending(err)
	}
	s.clearActiveTurn()
	<-s.serveDone
	return err
}

// Close closes the underlying transport.
func (s *Server) Close() error {
	return s.transport.Close()
}

func (s *Server) readLoop() {
	var readErr error
	defer func() {
		s.readErr = readErr
		close(s.readDone)
	}()
	for {
		line, err := s.transport.ReadLine(s.serveCtx)
		if err != nil {
			if err != io.EOF && !errors.Is(err, context.Canceled) && s.serveCtx.Err() == nil {
				s.logf("read error: %v", redact.RedactString(err.Error()))
			}
			readErr = err
			return
		}
		var raw protocol.RawWireMessage
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			s.logf("parse error: %v", redact.RedactString(err.Error()))
			_ = s.sendError("", codeParseError, err.Error())
			continue
		}
		if raw.ID != "" && raw.Method == "" {
			s.routeResponse(&raw)
			continue
		}
		select {
		case s.dispatchCh <- &raw:
		case <-s.serveCtx.Done():
			return
		}
	}
}

func (s *Server) serveLoop() {
	defer close(s.serveDone)
	for {
		select {
		case raw, ok := <-s.dispatchCh:
			if !ok {
				return
			}
			s.dispatch(raw)
		case <-s.serveCtx.Done():
			return
		}
	}
}

func (s *Server) dispatch(raw *protocol.RawWireMessage) {
	switch raw.Method {
	case "initialize":
		s.handleInitialize(raw.ID, raw.Params)
	case "prompt":
		s.handlePrompt(raw.ID, raw.Params)
	case "steer":
		s.handleSteer(raw.ID, raw.Params)
	case "cancel":
		s.handleCancel(raw.ID, raw.Params)
	case "replay":
		s.handleReplay(raw.ID, raw.Params)
	case "set_plan_mode":
		s.handleSetPlanMode(raw.ID, raw.Params)
	default:
		_ = s.sendError(raw.ID, codeMethodNotFound, "method not found")
	}
}

func (s *Server) routeResponse(raw *protocol.RawWireMessage) {
	s.mu.Lock()
	ch, ok := s.pending[raw.ID]
	if ok {
		delete(s.pending, raw.ID)
	}
	s.mu.Unlock()
	if ok {
		select {
		case ch <- raw:
		case <-s.serveCtx.Done():
		}
	}
}

func (s *Server) closePending(err error) {
	s.mu.Lock()
	pendingCopy := make(map[string]chan *protocol.RawWireMessage, len(s.pending))
	for id, ch := range s.pending {
		pendingCopy[id] = ch
		delete(s.pending, id)
	}
	s.mu.Unlock()

	safeMsg := redact.RedactString(err.Error())
	for _, ch := range pendingCopy {
		select {
		case ch <- &protocol.RawWireMessage{Error: &protocol.JSONRPCError{Code: -1, Message: safeMsg}}:
		default:
		}
	}
}

func (s *Server) handleInitialize(id string, params json.RawMessage) {
	var p protocol.InitializeParams
	if err := json.Unmarshal(params, &p); err != nil {
		_ = s.sendError(id, codeInvalidParams, err.Error())
		return
	}

	negotiated := s.negotiateVersion(p.ProtocolVersion)

	var accepted []string
	var rejected []protocol.RejectedExternalTool
	for _, tool := range p.ExternalTools {
		if err := s.toolValidator(tool); err != nil {
			rejected = append(rejected, protocol.RejectedExternalTool{Name: tool.Name, Reason: redact.RedactString(err.Error())})
		} else {
			accepted = append(accepted, tool.Name)
		}
	}
	extResult := &protocol.ExternalToolsResult{Accepted: accepted, Rejected: rejected}

	clientCaps := protocol.ClientCapabilities{}
	serverCaps := protocol.ServerCapabilities{}
	if p.Capabilities != nil {
		clientCaps = *p.Capabilities
		if p.Capabilities.SupportsQuestion != nil && *p.Capabilities.SupportsQuestion {
			b := true
			serverCaps.SupportsQuestion = &b
		}
	}

	hooks := make(map[string]*hookSubscription)
	hooksConfigured := make(map[string]uint32)
	for _, sub := range p.Hooks {
		if !s.isHookSupported(sub.Event) {
			continue
		}
		var matcher *regexp.Regexp
		if sub.Matcher != "" {
			var err error
			matcher, err = regexp.Compile(sub.Matcher)
			if err != nil {
				continue
			}
		}
		timeout := time.Duration(sub.Timeout) * time.Second
		hooks[sub.ID] = &hookSubscription{
			id:      sub.ID,
			event:   sub.Event,
			matcher: matcher,
			timeout: timeout,
		}
		hooksConfigured[sub.Event]++
	}

	s.mu.Lock()
	s.handshakeDone = true
	s.negotiated = negotiated
	s.clientCaps = clientCaps
	s.serverCaps = serverCaps
	s.externalTools = p.ExternalTools
	s.hooks = hooks
	s.mu.Unlock()

	result := protocol.InitializeResult{
		ProtocolVersion: negotiated,
		Server:          s.info,
		SlashCommands:   s.slashCmds,
		Capabilities:    &serverCaps,
	}
	if len(p.ExternalTools) > 0 {
		result.ExternalTools = extResult
	}
	if len(s.supportedHooks) > 0 || len(hooksConfigured) > 0 {
		result.Hooks = &protocol.HooksInfo{
			SupportedEvents: s.supportedHooks,
			Configured:      hooksConfigured,
		}
	}

	if err := s.sendResponse(id, result); err != nil {
		s.logf("failed to send initialize response: %v", redact.RedactString(err.Error()))
	}
}

func (s *Server) negotiateVersion(requested string) string {
	if requested == "" {
		return protocol.WireProtocolLegacyVersion
	}
	if cmp := semverCompare(requested, "1.1"); cmp < 0 {
		return protocol.WireProtocolLegacyVersion
	}
	if cmp := semverCompare(requested, protocol.WireProtocolVersion); cmp > 0 {
		return protocol.WireProtocolVersion
	}
	return requested
}

// semverCompare compares two dot-separated version strings.
// It returns -1 if a < b, 0 if equal, 1 if a > b.
func semverCompare(a, b string) int {
	partsA := splitVersion(a)
	partsB := splitVersion(b)
	for i := 0; i < len(partsA) || i < len(partsB); i++ {
		var va, vb int
		if i < len(partsA) {
			va = partsA[i]
		}
		if i < len(partsB) {
			vb = partsB[i]
		}
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
	}
	return 0
}

func splitVersion(v string) []int {
	var out []int
	start := 0
	for i := 0; i < len(v); i++ {
		if v[i] == '.' {
			out = append(out, atoi(v[start:i]))
			start = i + 1
		}
	}
	out = append(out, atoi(v[start:]))
	return out
}

func atoi(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			break
		}
		n = n*10 + int(s[i]-'0')
	}
	return n
}

func (s *Server) isHookSupported(event string) bool {
	for _, e := range s.supportedHooks {
		if e == event {
			return true
		}
	}
	return false
}

func (s *Server) handlePrompt(id string, params json.RawMessage) {
	var p protocol.PromptParams
	if err := json.Unmarshal(params, &p); err != nil {
		_ = s.sendError(id, codeInvalidParams, err.Error())
		return
	}

	s.mu.Lock()
	if s.activeTurn != nil {
		s.mu.Unlock()
		_ = s.sendError(id, codeTurnInProgress, "A turn is already in progress")
		return
	}
	t := newTurn(s, p.UserInput)
	s.activeTurn = t
	s.activeOp = activeOperation{cancel: t.cancel, done: t.done}
	s.mu.Unlock()

	if err := s.emitEvent(s.serveCtx, protocol.TurnBeginEvent(p)); err != nil {
		s.endTurn(id, t, protocol.PromptResult{}, err)
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logf("panic in Agent.Prompt: %v", redact.RedactString(fmt.Sprintf("%v", r)))
				s.endTurn(id, t, protocol.PromptResult{}, &codedError{code: codeInternalError, msg: "internal error"})
			}
		}()
		result, err := s.agent.Prompt(t.ctx, p.UserInput, t)
		s.endTurn(id, t, result, err)
	}()
}

func (s *Server) endTurn(id string, t *turn, result protocol.PromptResult, err error) {
	_ = s.emitEvent(s.serveCtx, protocol.TurnEndEvent{})
	t.close()
	s.mu.Lock()
	if s.activeTurn == t {
		s.activeTurn = nil
		s.activeOp = activeOperation{}
	}
	s.mu.Unlock()
	if err != nil {
		code := codeInternalError
		msg := "internal error"
		var ce CodedError
		if errors.As(err, &ce) && ce.Code() != 0 {
			code = ce.Code()
			msg = err.Error()
		}
		_ = s.sendError(id, code, msg)
		return
	}
	_ = s.sendResponse(id, result)
}

func (s *Server) clearActiveTurn() {
	s.mu.Lock()
	op := s.activeOp
	t := s.activeTurn
	s.activeTurn = nil
	s.activeOp = activeOperation{}
	s.mu.Unlock()
	if t != nil {
		t.cancel()
		<-t.done
	} else if op.cancel != nil {
		op.cancel()
		<-op.done
	}
}

func (s *Server) handleSteer(id string, params json.RawMessage) {
	steerer, ok := s.agent.(Steerer)
	if !ok {
		_ = s.sendError(id, codeMethodNotFound, "method not found")
		return
	}
	var p protocol.SteerParams
	if err := json.Unmarshal(params, &p); err != nil {
		_ = s.sendError(id, codeInvalidParams, err.Error())
		return
	}
	s.mu.Lock()
	t := s.activeTurn
	s.mu.Unlock()
	if t == nil {
		_ = s.sendError(id, codeTurnInProgress, "No agent turn is in progress")
		return
	}
	select {
	case t.steerCh <- p.UserInput:
	case <-t.ctx.Done():
		_ = s.sendError(id, codeTurnInProgress, "No agent turn is in progress")
		return
	}
	if err := steerer.Steer(t.ctx, p.UserInput); err != nil {
		_ = s.sendError(id, codeInternalError, err.Error())
		return
	}
	_ = s.sendResponse(id, protocol.SteerResult{Status: protocol.SteerStatusSteered})
}

func (s *Server) handleCancel(id string, params json.RawMessage) {
	s.mu.Lock()
	op := s.activeOp
	s.mu.Unlock()
	if op.cancel == nil {
		_ = s.sendError(id, codeTurnInProgress, "No agent turn is in progress")
		return
	}
	op.cancel()
	<-op.done
	_ = s.sendResponse(id, protocol.CancelResult{})
}

func (s *Server) handleReplay(id string, params json.RawMessage) {
	replayer, ok := s.agent.(Replayer)
	if !ok {
		_ = s.sendError(id, codeMethodNotFound, "method not found")
		return
	}

	replayCtx, cancel := context.WithCancel(s.serveCtx)
	done := make(chan struct{})

	s.mu.Lock()
	if s.activeTurn != nil || s.activeOp.cancel != nil {
		s.mu.Unlock()
		cancel()
		_ = s.sendError(id, codeTurnInProgress, "A turn is already in progress")
		return
	}
	s.activeOp = activeOperation{cancel: cancel, done: done}
	s.mu.Unlock()

	go func() {
		defer close(done)
		defer func() {
			s.mu.Lock()
			if s.activeOp.done == done {
				s.activeOp = activeOperation{}
			}
			s.mu.Unlock()
		}()
		result, err := replayer.Replay(replayCtx, s)
		if err != nil {
			_ = s.sendError(id, codeInternalError, err.Error())
			return
		}
		_ = s.sendResponse(id, result)
	}()
}

func (s *Server) handleSetPlanMode(id string, params json.RawMessage) {
	s.mu.Lock()
	supports := s.clientCaps.SupportsPlanMode != nil && *s.clientCaps.SupportsPlanMode
	s.mu.Unlock()
	if !supports {
		_ = s.sendError(id, codePlanModeNotSupported, "Plan mode is not supported")
		return
	}
	switcher, ok := s.agent.(PlanModeSwitcher)
	if !ok {
		_ = s.sendError(id, codeMethodNotFound, "method not found")
		return
	}
	var p protocol.SetPlanModeParams
	if err := json.Unmarshal(params, &p); err != nil {
		_ = s.sendError(id, codeInvalidParams, err.Error())
		return
	}
	result, err := switcher.SetPlanMode(s.serveCtx, p.Enabled, s)
	if err != nil {
		_ = s.sendError(id, codeInternalError, err.Error())
		return
	}
	_ = s.sendResponse(id, result)
}

func (s *Server) sendResponse(id string, result any) error {
	err := s.writeMessage(s.serveCtx, protocol.JSONRPCSuccessResponse[any]{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
	if err != nil {
		s.logf("failed to write response: %v", redact.RedactString(err.Error()))
	}
	return err
}

// Emit emits an event to the client. It satisfies the Emitter interface for
// optional capabilities such as Replayer and PlanModeSwitcher.
func (s *Server) Emit(ctx context.Context, event protocol.Event) error {
	return s.emitEvent(ctx, event)
}
