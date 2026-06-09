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

const maxPendingMessages = 1024

// Client is a high-level wire protocol client.
type Client struct {
	transport      Transport
	requestCounter uint64
	pending        map[string]chan *RawWireMessage
	oooBuffer      []*RawWireMessage // out-of-order messages
	mu, oooMu      sync.Mutex
	handshakeDone  bool
	defaultTimeout time.Duration
	maxIORetries   uint32
	readerDone     chan struct{}
	dispatchCh     chan *RawWireMessage
}

// NewClient creates a new client backed by the given transport.
func NewClient(transport Transport) *Client {
	c := &Client{
		transport:  transport,
		pending:    make(map[string]chan *RawWireMessage),
		readerDone: make(chan struct{}),
		dispatchCh: make(chan *RawWireMessage, 1024),
	}
	go c.readerLoop()
	return c
}

func (c *Client) nextID() string {
	return fmt.Sprintf("req-%d", atomic.AddUint64(&c.requestCounter, 1))
}

func (c *Client) readerLoop() {
	defer close(c.readerDone)
	for {
		line, err := c.transport.ReadLine(context.Background())
		if err != nil {
			return
		}
		var raw RawWireMessage
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}

		// Events and requests go to the dispatch channel.
		if raw.Method == "event" || raw.Method == "request" {
			select {
			case c.dispatchCh <- &raw:
			default:
				// Drop if dispatch buffer is full.
			}
			continue
		}

		c.mu.Lock()
		ch, ok := c.pending[raw.ID]
		c.mu.Unlock()
		if ok {
			ch <- &raw
		} else {
			c.oooMu.Lock()
			if len(c.oooBuffer) < maxPendingMessages {
				c.oooBuffer = append(c.oooBuffer, &raw)
			}
			c.oooMu.Unlock()
		}
	}
}

func (c *Client) sendRequest(ctx context.Context, req any) error {
	data, err := json.Marshal(req)
	if err != nil {
		return &WireError{Kind: ErrJSONSerialize, Message: err.Error()}
	}
	return c.transport.WriteLine(ctx, string(data))
}

func (c *Client) readResponse(ctx context.Context, expectedID string, result any) error {
	c.oooMu.Lock()
	for i, msg := range c.oooBuffer {
		if msg.ID == expectedID {
			c.oooBuffer = append(c.oooBuffer[:i], c.oooBuffer[i+1:]...)
			c.oooMu.Unlock()
			return c.decodeResponse(msg, result)
		}
	}
	c.oooMu.Unlock()

	ch := make(chan *RawWireMessage, 1)
	c.mu.Lock()
	c.pending[expectedID] = ch
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.pending, expectedID)
		c.mu.Unlock()
	}()

	if c.defaultTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.defaultTimeout)
		defer cancel()
	}

	select {
	case msg := <-ch:
		return c.decodeResponse(msg, result)
	case <-ctx.Done():
		return &WireError{Kind: ErrTimeout, Message: ctx.Err().Error()}
	}
}

func (c *Client) decodeResponse(msg *RawWireMessage, result any) error {
	if msg.Error != nil {
		return &WireError{
			Kind:    ErrRequestFailed,
			Message: msg.Error.Message,
			Code:    msg.Error.Code,
		}
	}
	if len(msg.Result) == 0 {
		return &WireError{Kind: ErrInternal, Message: "response missing result"}
	}
	if err := json.Unmarshal(msg.Result, result); err != nil {
		return &WireError{Kind: ErrJSONParse, Message: err.Error()}
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
	if err := c.sendRequest(ctx, req); err != nil {
		return InitializeResult{}, err
	}

	var result InitializeResult
	if err := c.readResponse(ctx, id, &result); err != nil {
		var werr *WireError
		if errors.As(err, &werr) && werr.Kind == ErrRequestFailed && werr.Code == MethodNotFound {
			c.handshakeDone = true
			return InitializeResult{
				ProtocolVersion: "legacy/no-handshake",
				Server:          ServerInfo{Name: "unknown", Version: "unknown"},
				SlashCommands:   []SlashCommandInfo{},
			}, nil
		}
		return InitializeResult{}, err
	}
	c.handshakeDone = true
	return result, nil
}

// IsHandshakeDone returns true if the initialize handshake has completed.
func (c *Client) IsHandshakeDone() bool { return c.handshakeDone }

// Prompt sends a prompt and waits for the result.
func (c *Client) Prompt(ctx context.Context, userInput UserInput) (PromptResult, error) {
	id := c.nextID()
	req := JSONRPCRequest[PromptParams]{
		JSONRPC: "2.0",
		Method:  "prompt",
		ID:      id,
		Params:  PromptParams{UserInput: userInput},
	}
	if err := c.sendRequest(ctx, req); err != nil {
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
	if err := c.sendRequest(ctx, req); err != nil {
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
	if err := c.sendRequest(ctx, req); err != nil {
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
	if err := c.sendRequest(ctx, req); err != nil {
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
	if err := c.sendRequest(ctx, req); err != nil {
		return err
	}
	var result CancelResult
	return c.readResponse(ctx, id, &result)
}

// Shutdown gracefully shuts down the client.
func (c *Client) Shutdown(ctx context.Context) error {
	err := c.transport.Close()
	<-c.readerDone
	close(c.dispatchCh)
	return err
}

// WithDefaultTimeout sets a default timeout for readResponse calls.
func (c *Client) WithDefaultTimeout(d time.Duration) *Client {
	c.defaultTimeout = d
	return c
}

// WithMaxIORetries sets the maximum number of retries for transient I/O errors.
func (c *Client) WithMaxIORetries(n uint32) *Client {
	if n > 5 {
		n = 5
	}
	c.maxIORetries = n
	return c
}

// SendResponse sends a JSON-RPC success response.
func (c *Client) SendResponse(ctx context.Context, id string, result any) error {
	resp := JSONRPCSuccessResponse[any]{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	return c.sendRequest(ctx, resp)
}

// SendError sends a JSON-RPC error response.
func (c *Client) SendError(ctx context.Context, id string, code int, message string) error {
	resp := JSONRPCErrorResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &JSONRPCError{Code: code, Message: message},
	}
	return c.sendRequest(ctx, resp)
}
