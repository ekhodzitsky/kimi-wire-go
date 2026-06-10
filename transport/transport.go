package transport

import "context"

// Transport abstracts reading and writing newline-delimited JSON.
type Transport interface {
	ReadLine(ctx context.Context) (string, error)
	WriteLine(ctx context.Context, line string) error
	Close() error
}
