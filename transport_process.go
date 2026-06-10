package wire

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
)

const maxWireLineLength = 16 * 1024 * 1024

// SpawnOptions configures a child-process transport spawn.
type SpawnOptions struct {
	WorkDir *string
	Session *string
	Model   *string
}

// ChildProcessTransport is a transport backed by a child process's stdin/stdout.
type ChildProcessTransport struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    *os.File
	reader    *bufio.Reader
	readMu    sync.Mutex
	closeOnce sync.Once
	mu        sync.Mutex
	closed    bool
}

var (
	pemBeginRe = regexp.MustCompile(`-----BEGIN (?:PGP PRIVATE KEY BLOCK|(?:RSA |EC |OPENSSH |DSA |ENCRYPTED )?PRIVATE KEY)-----`)
	pemEndRe   = regexp.MustCompile(`-----END (?:PGP PRIVATE KEY BLOCK|(?:RSA |EC |OPENSSH |DSA |ENCRYPTED )?PRIVATE KEY)-----`)
)

// SpawnChildProcessTransport spawns a new `kimi` process in wire mode.
func SpawnChildProcessTransport(kimiBinary string, opts SpawnOptions) (*ChildProcessTransport, error) {
	for attempt := 0; attempt < 3; attempt++ {
		cmd := exec.Command(kimiBinary, "--wire")
		if opts.WorkDir != nil {
			cmd.Args = append(cmd.Args, "--work-dir", *opts.WorkDir)
		}
		if opts.Session != nil {
			cmd.Args = append(cmd.Args, "--session", *opts.Session)
		}
		if opts.Model != nil {
			cmd.Args = append(cmd.Args, "--model", *opts.Model)
		}

		stdin, err := cmd.StdinPipe()
		if err != nil {
			return nil, &WireError{Kind: ErrSpawnFailed, Message: err.Error(), Cause: err}
		}

		stdoutRd, stdoutWr, err := os.Pipe()
		if err != nil {
			_ = stdin.Close()
			return nil, &WireError{Kind: ErrSpawnFailed, Message: err.Error(), Cause: err}
		}
		cmd.Stdout = stdoutWr

		stderr, err := cmd.StderrPipe()
		if err != nil {
			_ = stdin.Close()
			_ = stdoutRd.Close()
			_ = stdoutWr.Close()
			return nil, &WireError{Kind: ErrSpawnFailed, Message: err.Error(), Cause: err}
		}

		if err := cmd.Start(); err != nil {
			_ = stdin.Close()
			_ = stdoutRd.Close()
			_ = stdoutWr.Close()
			_ = stderr.Close()
			if attempt < 2 && errors.Is(err, syscall.ETXTBSY) {
				time.Sleep(25 * time.Millisecond)
				continue
			}
			return nil, &WireError{Kind: ErrSpawnFailed, Message: err.Error(), Cause: err}
		}

		// Close our reference to the write side; the child owns it now.
		_ = stdoutWr.Close()

		tr := &ChildProcessTransport{
			cmd:    cmd,
			stdin:  stdin,
			stdout: stdoutRd,
			reader: bufio.NewReader(stdoutRd),
		}

		go tr.logStderr(stderr)

		return tr, nil
	}
	return nil, &WireError{Kind: ErrSpawnFailed, Message: "all spawn attempts failed"}
}

func (t *ChildProcessTransport) logStderr(stderr io.ReadCloser) {
	defer func() { _ = stderr.Close() }()
	reader := bufio.NewReader(stderr)
	var accum []string
	inBlock := false
	for {
		line, err := reader.ReadString('\n')
		trimmed := strings.TrimSuffix(line, "\n")
		if inBlock {
			accum = append(accum, trimmed)
			if pemEndRe.MatchString(trimmed) {
				flushRedacted(accum)
				accum = nil
				inBlock = false
			}
		} else {
			if pemBeginRe.MatchString(trimmed) {
				inBlock = true
				accum = []string{trimmed}
				if pemEndRe.MatchString(trimmed) {
					flushRedacted(accum)
					accum = nil
					inBlock = false
				}
			} else if line != "" {
				log.Printf("[kimi stderr] %s", redactString(trimmed))
			}
		}
		if err != nil {
			if inBlock && len(accum) > 0 {
				flushRedacted(accum)
			}
			return
		}
	}
}

func flushRedacted(lines []string) {
	redacted := redactString(strings.Join(lines, "\n"))
	for _, l := range strings.Split(redacted, "\n") {
		log.Printf("[kimi stderr] %s", l)
	}
}

func (t *ChildProcessTransport) ReadLine(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", &WireError{Kind: ErrTimeout, Message: err.Error(), Cause: err}
	}

	t.readMu.Lock()
	defer t.readMu.Unlock()

	if deadline, ok := ctx.Deadline(); ok {
		_ = t.stdout.SetReadDeadline(deadline)
		defer func() { _ = t.stdout.SetReadDeadline(time.Time{}) }()
	}

	line, err := t.reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) && ctx.Err() != nil {
			return "", &WireError{Kind: ErrTimeout, Message: ctx.Err().Error(), Cause: ctx.Err()}
		}
		if errors.Is(err, io.EOF) {
			if line != "" {
				return strings.TrimSuffix(line, "\n"), nil
			}
			return "", io.EOF
		}
		return "", &WireError{Kind: ErrIO, Message: err.Error(), Cause: err}
	}
	return strings.TrimSuffix(line, "\n"), nil
}

type writeDeadliner interface {
	SetWriteDeadline(time.Time) error
}

func (t *ChildProcessTransport) WriteLine(ctx context.Context, line string) error {
	if err := ctx.Err(); err != nil {
		return &WireError{Kind: ErrTimeout, Message: err.Error(), Cause: err}
	}

	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return &WireError{Kind: ErrStreamClosed, Message: "transport closed"}
	}
	if t.stdin == nil {
		t.mu.Unlock()
		return &WireError{Kind: ErrIO, Message: "stdin not available"}
	}
	stdin := t.stdin
	t.mu.Unlock()

	if deadline, ok := ctx.Deadline(); ok {
		if d, ok := stdin.(writeDeadliner); ok {
			_ = d.SetWriteDeadline(deadline)
			defer func() { _ = d.SetWriteDeadline(time.Time{}) }()
		}
	}

	_, err := fmt.Fprintln(stdin, line)
	if err != nil {
		if ctx.Err() != nil {
			return &WireError{Kind: ErrTimeout, Message: ctx.Err().Error(), Cause: ctx.Err()}
		}
		return &WireError{Kind: ErrIO, Message: err.Error(), Cause: err}
	}
	return nil
}

func (t *ChildProcessTransport) Close() error {
	var closeErr error
	t.closeOnce.Do(func() {
		t.mu.Lock()
		t.closed = true
		if t.stdin != nil {
			closeErr = t.stdin.Close()
			t.stdin = nil
		}
		t.mu.Unlock()

		// Closing stdout unblocks any active reader in ReadLine.
		if t.stdout != nil {
			_ = t.stdout.Close()
		}

		if t.cmd != nil && t.cmd.Process != nil {
			_ = t.cmd.Process.Signal(os.Interrupt)
			done := make(chan error, 1)
			go func() { done <- t.cmd.Wait() }()
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				_ = t.cmd.Process.Kill()
			}
		}
	})
	return closeErr
}
