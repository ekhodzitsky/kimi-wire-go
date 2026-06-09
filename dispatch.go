package wire

import "context"

// Handler handles incoming events and requests during a dispatch loop.
type Handler interface {
	HandleEvent(ctx context.Context, event Event) error
	HandleRequest(ctx context.Context, req Request) (any, error)
}

// Dispatch runs a loop reading and dispatching incoming wire messages.
// It should be run in its own goroutine alongside synchronous method calls.
func (c *Client) Dispatch(ctx context.Context, handler Handler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case raw, ok := <-c.dispatchCh:
			if !ok {
				return nil
			}
			msg, err := ParseWireMessage(*raw)
			if err != nil {
				continue
			}
			switch m := msg.(type) {
			case EventMessage:
				_ = handler.HandleEvent(ctx, m.Event)
			case RequestMessage:
				resp, err := handler.HandleRequest(ctx, m.Request)
				if err != nil {
					_ = c.SendError(ctx, m.ID, -32603, err.Error())
				} else {
					_ = c.SendResponse(ctx, m.ID, resp)
				}
			}
		}
	}
}
