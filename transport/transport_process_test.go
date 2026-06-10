package transport

import (
	"bytes"
	"context"
	"log"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestChildProcessTransportSmoke(t *testing.T) {
	if _, err := exec.LookPath("kimi"); err != nil {
		t.Skip("kimi binary not found in PATH")
	}
	ctx := context.Background()
	tr, err := SpawnChildProcessTransport("kimi", SpawnOptions{})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	defer tr.Close()

	if err := tr.WriteLine(ctx, `{"jsonrpc":"2.0","method":"initialize","id":"1","params":{"protocol_version":"1.10"}}`); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestChildProcessTransportStderrRedaction(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not found")
	}
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(log.Writer())

	cmd := exec.Command("sh", "-c", "echo 'api_key=supersecret12345678' >&2; sleep 0.1")
	stdin, _ := cmd.StdinPipe()
	_, _ = cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	tr := &ChildProcessTransport{cmd: cmd, stdin: stdin}
	go tr.logStderr(stderr)
	time.Sleep(150 * time.Millisecond)
	_ = tr.Close()

	out := buf.String()
	if strings.Contains(out, "supersecret12345678") {
		t.Fatalf("secret was not redacted in stderr log: %s", out)
	}
	if !strings.Contains(out, "api_key=***") {
		t.Fatalf("expected redacted api_key, got: %s", out)
	}
}
