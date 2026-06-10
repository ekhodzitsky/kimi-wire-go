package transport

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
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
			return nil, fmt.Errorf("spawn failed: %w", err)
		}

		stdoutRd, stdoutWr, err := os.Pipe()
		if err != nil {
			_ = stdin.Close()
			return nil, fmt.Errorf("spawn failed: %w", err)
		}
		cmd.Stdout = stdoutWr

		stderr, err := cmd.StderrPipe()
		if err != nil {
			_ = stdin.Close()
			_ = stdoutRd.Close()
			_ = stdoutWr.Close()
			return nil, fmt.Errorf("spawn failed: %w", err)
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
			return nil, fmt.Errorf("spawn failed: %w", err)
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
	return nil, fmt.Errorf("all spawn attempts failed")
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
		return "", fmt.Errorf("timeout: %w", err)
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
			return "", fmt.Errorf("timeout: %w", ctx.Err())
		}
		if errors.Is(err, io.EOF) {
			if line != "" {
				return strings.TrimSuffix(line, "\n"), nil
			}
			return "", io.EOF
		}
		return "", fmt.Errorf("io error: %w", err)
	}
	return strings.TrimSuffix(line, "\n"), nil
}

type writeDeadliner interface {
	SetWriteDeadline(time.Time) error
}

func (t *ChildProcessTransport) WriteLine(ctx context.Context, line string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("timeout: %w", err)
	}

	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return fmt.Errorf("transport closed")
	}
	if t.stdin == nil {
		t.mu.Unlock()
		return fmt.Errorf("stdin not available")
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
			return fmt.Errorf("timeout: %w", ctx.Err())
		}
		return fmt.Errorf("io error: %w", err)
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

// redactString scrubs secrets from a string.
func redactString(s string) string {
	result := s
	for i, re := range secretPatterns {
		result = re.ReplaceAllString(result, secretReplacements[i])
	}
	return result
}

var (
	secretKeyPattern = regexp.MustCompile(`(?i)(?:^|[^a-z0-9])(api[_-]?key|token|secret|password|auth|access[_-]?key|private[_-]?key|session[_-]?token|bearer|authorization)(?:$|[^a-z0-9])`)

	secretPatterns = []*regexp.Regexp{
		// key = value / key: value / "key": "value" style assignments.
		regexp.MustCompile(`(?i)((?:api[_-]?key|token|secret|password|auth|access[_-]?key|private[_-]?key|session[_-]?token|bearer|authorization)\s*[:=]\s*)["']?[^"'\s]{8,}["']?`),
		// Authorization: Bearer/Basic/Token/API-Key <value>
		regexp.MustCompile(`(?i)(authorization\s*[:=]\s*(?:bearer|basic|token|api-key)\s+)[^\s]+`),
		// AWS Access Key ID.
		regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
		// GitHub personal access tokens.
		regexp.MustCompile(`\b(ghp_|github_pat_|gho_|ghu_|ghs_|ghr_)[a-zA-Z0-9_\-]+\b`),
		// JWT (three base64url segments, including padding).
		regexp.MustCompile(`\beyJ[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+=*`),
		// URL with embedded credentials.
		regexp.MustCompile(`(?i)(\bhttps?://[^:]+:)[^@]+(@[^\s]+)`),
		// PEM private keys.
		regexp.MustCompile(`(?s)-----BEGIN (?:RSA |EC |OPENSSH |DSA |ENCRYPTED )?PRIVATE KEY-----.*?-----END (?:RSA |EC |OPENSSH |DSA |ENCRYPTED )?PRIVATE KEY-----`),
		// PGP private key block.
		regexp.MustCompile(`(?s)-----BEGIN PGP PRIVATE KEY BLOCK-----.*?-----END PGP PRIVATE KEY BLOCK-----`),
	}

	secretReplacements = []string{
		"${1}***",
		"${1}***",
		"AKIA...REDACTED",
		"${1}***",
		"eyJ...REDACTED",
		"${1}***${2}",
		"[PEM_PRIVATE_KEY_REDACTED]",
		"[PGP_PRIVATE_KEY_BLOCK_REDACTED]",
	}
)

// RedactSecrets recursively scrubs secrets from a JSON-like value.
func RedactSecrets(v any) any {
	switch val := v.(type) {
	case string:
		return redactString(val)
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, v2 := range val {
			if secretKeyPattern.MatchString(k) {
				out[k] = "***"
			} else {
				out[k] = RedactSecrets(v2)
			}
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, v2 := range val {
			out[i] = RedactSecrets(v2)
		}
		return out
	case json.RawMessage:
		dec := json.NewDecoder(bytes.NewReader(val))
		dec.UseNumber()
		var inner any
		if err := dec.Decode(&inner); err != nil {
			return json.RawMessage(redactString(string(val)))
		}
		redacted := RedactSecrets(inner)
		out, err := json.Marshal(redacted)
		if err != nil {
			return json.RawMessage(redactString(string(val)))
		}
		return json.RawMessage(out)
	default:
		return v
	}
}
