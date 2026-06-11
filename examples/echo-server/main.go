package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/ekhodzitsky/kimi-wire"
)

// stdioTransport reads NDJSON from stdin and writes to stdout.
type stdioTransport struct {
	reader *bufio.Reader
}

func newStdioTransport() *stdioTransport {
	return &stdioTransport{reader: bufio.NewReader(os.Stdin)}
}

func (t *stdioTransport) ReadLine(ctx context.Context) (string, error) {
	line, err := t.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return line[:len(line)-1], nil
}

func (t *stdioTransport) WriteLine(ctx context.Context, line string) error {
	_, err := fmt.Fprintln(os.Stdout, line)
	return err
}

func (t *stdioTransport) Close() error {
	_ = os.Stdout.Close()
	return nil
}

type echoAgent struct{}

func (a *echoAgent) Prompt(ctx context.Context, input wire.UserInput, turn wire.Turn) (wire.PromptResult, error) {
	// Demonstrate RequestApproval before echoing.
	approval, err := turn.RequestApproval(ctx, wire.ApprovalRequest{
		ID:          "approve-echo",
		ToolCallID:  "echo-1",
		Sender:      "echo-server",
		Action:      "echo",
		Description: "Echo user input back",
	})
	if err != nil {
		return wire.PromptResult{Status: wire.PromptStatusCancelled}, err
	}
	if approval.Response != wire.ApprovalResponseKindApprove {
		return wire.PromptResult{Status: wire.PromptStatusCancelled}, nil
	}

	// Demonstrate CallExternalTool to obtain a greeting.
	greeting, err := turn.CallExternalTool(ctx, wire.ToolCallRequest{
		ID:   "greet-tool",
		Name: "greeting",
	})
	if err != nil {
		return wire.PromptResult{Status: wire.PromptStatusCancelled}, err
	}

	text := input.Text
	if text == "" {
		text = "hello"
	}
	message := fmt.Sprintf("%s Echo: %s", greeting.ReturnValue.Output.Text, text)
	_ = turn.Emit(ctx, wire.ContentPartEvent{Part: wire.ContentPart{
		Type: wire.ContentPartTypeText,
		Text: &wire.TextPart{Text: message},
	}})

	return wire.PromptResult{Status: wire.PromptStatusFinished}, nil
}

func main() {
	var (
		wireMode = flag.Bool("wire", false, "run in wire mode")
		info     = flag.Bool("info", false, "print server info")
		jsonInfo = flag.Bool("json", false, "print info as JSON")
	)
	flag.Parse()

	if *info && *jsonInfo {
		fmt.Println(`{"wire_protocol_version":"` + wire.WireProtocolVersion + `"}`)
		return
	}
	if !*wireMode {
		log.Println("run with --wire")
		os.Exit(1)
	}

	server := wire.NewServer(newStdioTransport(), &echoAgent{}, wire.WithServerInfo("echo-server", "0.1.0"))
	if err := server.Serve(context.Background()); err != nil && err != io.EOF {
		log.Fatal(err)
	}
}
