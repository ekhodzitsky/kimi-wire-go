package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	wire "github.com/ekhodzitsky/kimi-wire"
)

func TestGoldenFixtures(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			t.Parallel()

			path := filepath.Join("testdata", entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}

			var raw wire.RawWireMessage
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("unmarshal RawWireMessage: %v", err)
			}

			switch raw.Method {
			case "event":
				roundTripEvent(t, raw.Params)
			case "request":
				roundTripRequest(t, raw.Params)
			default:
				roundTripRaw(t, raw)
			}
		})
	}
}

func roundTripEvent(t *testing.T, params json.RawMessage) {
	t.Helper()
	ev, err := wire.ParseEvent(params)
	if err != nil {
		t.Fatalf("parse event: %v", err)
	}

	marshaled, err := wire.MarshalEvent(ev)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	ev2, err := wire.ParseEvent(marshaled)
	if err != nil {
		t.Fatalf("parse marshaled event: %v", err)
	}

	if wire.TypeName(ev) != wire.TypeName(ev2) {
		t.Fatalf("event type changed: %q -> %q", wire.TypeName(ev), wire.TypeName(ev2))
	}
}

func roundTripRequest(t *testing.T, params json.RawMessage) {
	t.Helper()
	req, err := wire.ParseRequest(params)
	if err != nil {
		t.Fatalf("parse request: %v", err)
	}

	marshaled, err := wire.MarshalRequest(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req2, err := wire.ParseRequest(marshaled)
	if err != nil {
		t.Fatalf("parse marshaled request: %v", err)
	}

	if wire.Kind(req) != wire.Kind(req2) {
		t.Fatalf("request kind changed: %q -> %q", wire.Kind(req), wire.Kind(req2))
	}
}

func roundTripRaw(t *testing.T, raw wire.RawWireMessage) {
	t.Helper()
	b, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal raw: %v", err)
	}

	var raw2 wire.RawWireMessage
	if err := json.Unmarshal(b, &raw2); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}

	if raw.JSONRPC != raw2.JSONRPC {
		t.Fatalf("jsonrpc changed: %q -> %q", raw.JSONRPC, raw2.JSONRPC)
	}
	if raw.ID != raw2.ID {
		t.Fatalf("id changed: %q -> %q", raw.ID, raw2.ID)
	}
	if raw.Method != raw2.Method {
		t.Fatalf("method changed: %q -> %q", raw.Method, raw2.Method)
	}
}
