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
				// Malformed dispatch message: drop and continue.
				continue
			}
			switch m := msg.(type) {
			case EventMessage:
				if err := handler.HandleEvent(ctx, m.Event); err != nil {
					return err
				}
			case RequestMessage:
				resp, err := handler.HandleRequest(ctx, m.Request)
				if err != nil {
					if sendErr := c.SendError(ctx, m.ID, -32603, redactString(err.Error())); sendErr != nil {
						return sendErr
					}
				} else {
					if sendErr := c.SendResponse(ctx, m.ID, resp); sendErr != nil {
						return sendErr
					}
				}
			}
		}
	}
}
