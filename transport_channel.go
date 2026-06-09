package wire

import (
	"context"
	"io"
)

// ChannelTransport is an in-memory transport backed by channels.
type ChannelTransport struct {
	rx <-chan string
	tx chan<- string
}

// NewChannelTransportPair creates a connected pair of ChannelTransports.
func NewChannelTransportPair() (a, b *ChannelTransport) {
	ch1 := make(chan string, 64)
	ch2 := make(chan string, 64)
	return &ChannelTransport{rx: ch1, tx: ch2}, &ChannelTransport{rx: ch2, tx: ch1}
}

func (t *ChannelTransport) ReadLine(ctx context.Context) (string, error) {
	select {
	case line, ok := <-t.rx:
		if !ok {
			return "", io.EOF
		}
		return line, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (t *ChannelTransport) WriteLine(ctx context.Context, line string) error {
	select {
	case t.tx <- line:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *ChannelTransport) Close() error {
	close(t.tx)
	return nil
}
