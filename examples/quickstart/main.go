package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ekhodzitsky/kimi-wire"
)

func main() {
	ctx := context.Background()

	// Spawn a child process running `kimi --wire`
	transport, err := wire.SpawnChildProcessTransport("kimi", wire.SpawnOptions{})
	if err != nil {
		log.Fatal(err)
	}
	defer transport.Close()

	client := wire.NewClient(transport)

	// Perform the initialization handshake
	if _, err := client.Initialize(ctx, wire.InitializeParams{
		ProtocolVersion: wire.WireProtocolVersion,
	}); err != nil {
		log.Fatal(err)
	}

	// Send a prompt
	result, err := client.Prompt(ctx, wire.UserInput{Text: "Hello!"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Status:", result.Status)
}
