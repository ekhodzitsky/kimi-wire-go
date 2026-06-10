package wire

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Client is a high-level wire protocol client.
// The first fields are 64-bit atomics; keep them at the top of the struct
// for alignment compatibility on 32-bit platforms.
type Client struct {
	requestCounter    uint64
	defaultTimeout    atomic.Int64 // nanoseconds; 0 means none
	transport         Transport
	pending           map[string]chan *RawWireMessage
	oooBuffer         []*RawWireMessage // out-of-order messages
	mu                sync.Mutex
	handshakeDone     atomic.Bool
	readerDone        chan struct{}
	dispatchCh        chan *RawWireMessage
	dispatchCloseOnce sync.Once
	stopCh            chan struct{}
	stopOnce          sync.Once
}

// NewClient creates a new client backed by the given transport.
func NewClient(transport Transport) *Client {
	if transport == nil {
		panic("wire: nil transport")
	}
	c := &Client{
		transport:  transport,
		pending:    make(map[string]chan *RawWireMessage),
		readerDone: make(chan struct{}),
		dispatchCh: make(chan *RawWireMessage, 1024),
		stopCh:     make(chan struct{}),
	}
	go c.readerLoop()
	return c
}

func (c *Client) nextID() string {
	return fmt.Sprintf("req-%d", atomic.AddUint64(&c.requestCounter, 1))
}

func (c *Client) readerLoop() {
	defer close(c.readerDone)
	defer c.dispatchCloseOnce.Do(func() { close(c.dispatchCh) })
	for {
		line, err := c.transport.ReadLine(context.Background())
		if err != nil {
			c.closePending(err)
			return
		}
		raw := new(RawWireMessage)
		if err := json.Unmarshal([]byte(line), raw); err != nil {
			parseErr := &WireError{Kind: ErrJSONParse, Message: err.Error(), Cause: err}
			c.closePending(parseErr)
			return
		}

		if raw.Method != "" {
			switch raw.Method {
			case "event", "request":
				select {
				case c.dispatchCh <- raw:
				case <-c.stopCh:
					return
				}
			default:
				// Unknown method: drop.
			}
			continue
		}

		c.mu.Lock()
		ch, ok := c.pending[raw.ID]
		if ok {
			c.mu.Unlock()
			select {
			case ch <- raw:
			case <-c.stopCh:
				return
			}
		} else {
			c.oooBuffer = append(c.oooBuffer, raw)
			c.mu.Unlock()
		}
	}
}

func (c *Client) closePending(transportErr error) {
	c.mu.Lock()
	pendingCopy := make(map[string]chan *RawWireMessage, len(c.pending))
	for id, ch := range c.pending {
		pendingCopy[id] = ch
		delete(c.pending, id)
	}
	c.mu.Unlock()

	safeMsg := redactString(transportErr.Error())
	for _, ch := range pendingCopy {
		msg := &RawWireMessage{Error: &JSONRPCError{Code: -1, Message: safeMsg}}
		select {
		case ch <- msg:
		default:
		}
	}
}

func (c *Client) writeMessage(ctx context.Context, msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return &WireError{Kind: ErrJSONSerialize, Message: err.Error(), Cause: err}
	}
	return c.transport.WriteLine(ctx, string(data))
}

func (c *Client) readResponse(ctx context.Context, expectedID string, result any) error {
	ch := make(chan *RawWireMessage, 1)

	c.mu.Lock()
	for i := 0; i < len(c.oooBuffer); i++ {
		if c.oooBuffer[i].ID == expectedID {
			msg := c.oooBuffer[i]
			c.oooBuffer = append(c.oooBuffer[:i], c.oooBuffer[i+1:]...)
			c.mu.Unlock()
			return c.decodeResponse(msg, result)
		}
	}
	c.pending[expectedID] = ch
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, expectedID)
		// Drain any late response to prevent goroutine leak.
		select {
		case <-ch:
		default:
		}
		c.mu.Unlock()
	}()

	if d := c.defaultTimeout.Load(); d > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(d))
		defer cancel()
	}

	select {
	case msg := <-ch:
		return c.decodeResponse(msg, result)
	case <-ctx.Done():
		return &WireError{Kind: ErrTimeout, Message: ctx.Err().Error(), Cause: ctx.Err()}
	}
}

func (c *Client) decodeResponse(msg *RawWireMessage, result any) error {
	if msg.Error != nil {
		return &WireError{
			Kind:    ErrRequestFailed,
			Message: redactString(msg.Error.Message),
			Code:    msg.Error.Code,
		}
	}
	if len(msg.Result) == 0 {
		return &WireError{Kind: ErrInternal, Message: "response missing result"}
	}
	if err := json.Unmarshal(msg.Result, result); err != nil {
		return &WireError{Kind: ErrJSONParse, Message: err.Error(), Cause: err}
	}
	return nil
}

// Initialize performs the initialize handshake.
func (c *Client) Initialize(ctx context.Context, params InitializeParams) (InitializeResult, error) {
	id := c.nextID()
	req := JSONRPCRequest[InitializeParams]{
		JSONRPC: "2.0",
		Method:  "initialize",
		ID:      id,
		Params:  params,
	}
	if err := c.writeMessage(ctx, req); err != nil {
		return InitializeResult{}, err
	}

	var result InitializeResult
	if err := c.readResponse(ctx, id, &result); err != nil {
		var werr *WireError
		if errors.As(err, &werr) && werr.Kind == ErrRequestFailed && werr.Code == MethodNotFound {
			c.handshakeDone.Store(true)
			return InitializeResult{
				ProtocolVersion: WireProtocolLegacyVersion,
				Server:          ServerInfo{Name: "unknown", Version: "unknown"},
				SlashCommands:   []SlashCommandInfo{},
			}, nil
		}
		return InitializeResult{}, err
	}
	c.handshakeDone.Store(true)
	return result, nil
}

// IsHandshakeDone returns true if the initialize handshake has completed.
func (c *Client) IsHandshakeDone() bool { return c.handshakeDone.Load() }

// Prompt sends a prompt and waits for the result.
func (c *Client) Prompt(ctx context.Context, userInput UserInput) (PromptResult, error) {
	id := c.nextID()
	req := JSONRPCRequest[PromptParams]{
		JSONRPC: "2.0",
		Method:  "prompt",
		ID:      id,
		Params:  PromptParams{UserInput: userInput},
	}
	if err := c.writeMessage(ctx, req); err != nil {
		return PromptResult{}, err
	}
	var result PromptResult
	if err := c.readResponse(ctx, id, &result); err != nil {
		return PromptResult{}, err
	}
	return result, nil
}

// Replay replays events and requests from the current session.
func (c *Client) Replay(ctx context.Context) (ReplayResult, error) {
	id := c.nextID()
	req := JSONRPCRequest[ReplayParams]{
		JSONRPC: "2.0",
		Method:  "replay",
		ID:      id,
		Params:  ReplayParams{},
	}
	if err := c.writeMessage(ctx, req); err != nil {
		return ReplayResult{}, err
	}
	var result ReplayResult
	if err := c.readResponse(ctx, id, &result); err != nil {
		return ReplayResult{}, err
	}
	return result, nil
}

// Steer steers the current turn with additional user input.
func (c *Client) Steer(ctx context.Context, userInput UserInput) (SteerResult, error) {
	id := c.nextID()
	req := JSONRPCRequest[SteerParams]{
		JSONRPC: "2.0",
		Method:  "steer",
		ID:      id,
		Params:  SteerParams{UserInput: userInput},
	}
	if err := c.writeMessage(ctx, req); err != nil {
		return SteerResult{}, err
	}
	var result SteerResult
	if err := c.readResponse(ctx, id, &result); err != nil {
		return SteerResult{}, err
	}
	return result, nil
}

// SetPlanMode enables or disables plan mode.
func (c *Client) SetPlanMode(ctx context.Context, enabled bool) (SetPlanModeResult, error) {
	id := c.nextID()
	req := JSONRPCRequest[SetPlanModeParams]{
		JSONRPC: "2.0",
		Method:  "set_plan_mode",
		ID:      id,
		Params:  SetPlanModeParams{Enabled: enabled},
	}
	if err := c.writeMessage(ctx, req); err != nil {
		return SetPlanModeResult{}, err
	}
	var result SetPlanModeResult
	if err := c.readResponse(ctx, id, &result); err != nil {
		return SetPlanModeResult{}, err
	}
	return result, nil
}

// Cancel cancels the current turn.
func (c *Client) Cancel(ctx context.Context) error {
	id := c.nextID()
	req := JSONRPCRequest[CancelParams]{
		JSONRPC: "2.0",
		Method:  "cancel",
		ID:      id,
		Params:  CancelParams{},
	}
	if err := c.writeMessage(ctx, req); err != nil {
		return err
	}
	var result CancelResult
	return c.readResponse(ctx, id, &result)
}

// Shutdown gracefully shuts down the client.
func (c *Client) Shutdown(ctx context.Context) error {
	c.stopOnce.Do(func() { close(c.stopCh) })
	closeErr := c.transport.Close()
	select {
	case <-c.readerDone:
	case <-ctx.Done():
		if closeErr != nil {
			return &WireError{Kind: ErrTimeout, Message: fmt.Sprintf("shutdown: %v (context: %v)", closeErr, ctx.Err()), Cause: ctx.Err()}
		}
		return ctx.Err()
	}
	return closeErr
}

// WithDefaultTimeout sets a default timeout for readResponse calls.
func (c *Client) WithDefaultTimeout(d time.Duration) *Client {
	c.defaultTimeout.Store(int64(d))
	return c
}

// SendRaw sends a pre-built raw wire message.
func (c *Client) SendRaw(ctx context.Context, raw *RawWireMessage) error {
	if raw == nil {
		return &WireError{Kind: ErrInternal, Message: "nil raw message"}
	}
	return c.writeMessage(ctx, raw)
}

// ReadRawMessage reads the next raw wire message from the transport.
// This is a low-level primitive; most callers should use Prompt/Steer/etc.
func (c *Client) ReadRawMessage(ctx context.Context) (*RawWireMessage, error) {
	line, err := c.transport.ReadLine(ctx)
	if err != nil {
		return nil, err
	}
	raw := new(RawWireMessage)
	if err := json.Unmarshal([]byte(line), raw); err != nil {
		return nil, &WireError{Kind: ErrJSONParse, Message: err.Error(), Cause: err}
	}
	return raw, nil
}

// SendResponse sends a JSON-RPC success response.
func (c *Client) SendResponse(ctx context.Context, id string, result any) error {
	resp := JSONRPCSuccessResponse[any]{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	return c.writeMessage(ctx, resp)
}

// SendError sends a JSON-RPC error response.
func (c *Client) SendError(ctx context.Context, id string, code int, message string) error {
	resp := JSONRPCErrorResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &JSONRPCError{Code: code, Message: redactString(message)},
	}
	return c.writeMessage(ctx, resp)
}
