package mcp

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestTransport_ReadWrite(t *testing.T) {
	// Simulate a pipe: write JSON-RPC messages, read them back.
	in := new(bytes.Buffer)
	out := new(bytes.Buffer)
	log := new(bytes.Buffer)

	tr := NewTransport(in, out, log)

	// Write two messages to the "stdin" buffer.
	msg1 := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	msg2 := `{"jsonrpc":"2.0","id":2,"method":"ping"}`
	in.WriteString(msg1 + "\n")
	in.WriteString(msg2 + "\n")

	// Read first message.
	got1, err := tr.Read()
	if err != nil {
		t.Fatalf("Read 1: %v", err)
	}
	if string(got1) != msg1 {
		t.Errorf("msg1 = %q, want %q", got1, msg1)
	}

	// Read second message.
	got2, err := tr.Read()
	if err != nil {
		t.Fatalf("Read 2: %v", err)
	}
	if string(got2) != msg2 {
		t.Errorf("msg2 = %q, want %q", got2, msg2)
	}

	// Read at EOF.
	_, err = tr.Read()
	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
}

func TestTransport_Write(t *testing.T) {
	out := new(bytes.Buffer)
	tr := NewTransport(strings.NewReader(""), out, new(bytes.Buffer))

	resp := &Response{
		JSONRPC: "2.0",
		ID:      rawID(1),
		Result:  json.RawMessage(`{}`),
	}

	if err := tr.Write(resp); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Output should be a single JSON line followed by newline.
	line := out.String()
	if !strings.HasSuffix(line, "\n") {
		t.Error("output should end with newline")
	}

	trimmed := strings.TrimSpace(line)
	var got Response
	if err := json.Unmarshal([]byte(trimmed), &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if got.JSONRPC != "2.0" {
		t.Errorf("jsonrpc = %q, want %q", got.JSONRPC, "2.0")
	}
}

func TestTransport_WriteMultiple(t *testing.T) {
	out := new(bytes.Buffer)
	tr := NewTransport(strings.NewReader(""), out, new(bytes.Buffer))

	for i := 0; i < 3; i++ {
		resp := &Response{
			JSONRPC: "2.0",
			ID:      rawID(i),
			Result:  json.RawMessage(`{}`),
		}
		if err := tr.Write(resp); err != nil {
			t.Fatalf("Write %d: %v", i, err)
		}
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestTransport_Log(t *testing.T) {
	log := new(bytes.Buffer)
	tr := NewTransport(strings.NewReader(""), new(bytes.Buffer), log)

	tr.Log("test message: %s", "hello")

	got := log.String()
	if !strings.Contains(got, "test message: hello") {
		t.Errorf("log output = %q, want to contain %q", got, "test message: hello")
	}
}

func TestTransport_EmptyLines(t *testing.T) {
	// Empty lines between messages should be skipped.
	input := "\n\n" + `{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n\n"
	in := strings.NewReader(input)
	tr := NewTransport(in, new(bytes.Buffer), new(bytes.Buffer))

	msg, err := tr.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	var req Request
	if err := json.Unmarshal(msg, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.Method != "ping" {
		t.Errorf("method = %q, want %q", req.Method, "ping")
	}
}

func TestTransport_ServerIntegration(t *testing.T) {
	// Full round-trip: write requests to stdin pipe, run server loop, read responses from stdout pipe.
	requests := []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"ping"}`,
	}

	in := new(bytes.Buffer)
	for _, r := range requests {
		in.WriteString(r + "\n")
	}

	out := new(bytes.Buffer)
	log := new(bytes.Buffer)

	srv := NewServer()
	tr := NewTransport(in, out, log)

	// Run the serve loop.
	err := srv.Serve(tr)
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// Parse responses from stdout.
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	// We expect 2 responses: initialize (id:1) and ping (id:2).
	// The notification should NOT produce a response.
	if len(lines) != 2 {
		t.Fatalf("expected 2 response lines, got %d: %v", len(lines), lines)
	}

	// Verify initialize response.
	var resp1 Response
	if err := json.Unmarshal([]byte(lines[0]), &resp1); err != nil {
		t.Fatalf("unmarshal resp1: %v", err)
	}
	if resp1.Error != nil {
		t.Fatalf("initialize error: %v", resp1.Error)
	}

	// Verify ping response.
	var resp2 Response
	if err := json.Unmarshal([]byte(lines[1]), &resp2); err != nil {
		t.Fatalf("unmarshal resp2: %v", err)
	}
	if resp2.Error != nil {
		t.Fatalf("ping error: %v", resp2.Error)
	}
	if string(resp2.Result) != "{}" {
		t.Errorf("ping result = %s, want {}", resp2.Result)
	}
}

// rawID is a helper to create a json.RawMessage ID.
func rawID(id int) *json.RawMessage {
	data := json.RawMessage([]byte(strings.TrimSpace(
		func() string { b, _ := json.Marshal(id); return string(b) }(),
	)))
	return &data
}
