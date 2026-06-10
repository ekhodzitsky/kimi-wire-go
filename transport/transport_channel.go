package transport

import (
	"context"
	"fmt"
	"io"
	"sync"
)

// ChannelTransport is an in-memory transport backed by channels.
type ChannelTransport struct {
	rx        <-chan string
	tx        chan<- string
	closeCh   chan struct{}
	closeOnce sync.Once
}

// NewChannelTransportPair creates a connected pair of ChannelTransports.
func NewChannelTransportPair() (a, b *ChannelTransport) {
	ch1 := make(chan string, 64)
	ch2 := make(chan string, 64)
	return &ChannelTransport{rx: ch1, tx: ch2, closeCh: make(chan struct{})},
		&ChannelTransport{rx: ch2, tx: ch1, closeCh: make(chan struct{})}
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
	case <-t.closeCh:
		return "", io.EOF
	}
}

func (t *ChannelTransport) WriteLine(ctx context.Context, line string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if fmt.Sprint(r) == "send on closed channel" {
				err = fmt.Errorf("transport closed")
			} else {
				panic(r)
			}
		}
	}()

	select {
	case <-t.closeCh:
		return fmt.Errorf("transport closed")
	case <-ctx.Done():
		return ctx.Err()
	case t.tx <- line:
		return nil
	}
}

func (t *ChannelTransport) Close() error {
	t.closeOnce.Do(func() {
		close(t.closeCh)
		close(t.tx)
	})
	return nil
}
