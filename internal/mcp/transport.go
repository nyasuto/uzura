package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Transport handles newline-delimited JSON-RPC communication over stdio.
// Messages are read from stdin and written to stdout, one JSON object per line.
// Logging goes to stderr so it does not interfere with the MCP protocol.
type Transport struct {
	scanner *bufio.Scanner
	writer  *bufio.Writer
	logger  io.Writer
}

// NewTransport creates a Transport that reads from r (stdin), writes to w (stdout),
// and logs to log (stderr).
func NewTransport(r io.Reader, w, log io.Writer) *Transport {
	s := bufio.NewScanner(r)
	// Allow up to 10 MB per line for large JSON-RPC messages.
	s.Buffer(make([]byte, 64*1024), 10*1024*1024)
	return &Transport{
		scanner: s,
		writer:  bufio.NewWriter(w),
		logger:  log,
	}
}

// Read reads the next JSON-RPC message from the input stream.
// Empty lines are skipped. Returns io.EOF at end of input.
func (t *Transport) Read() ([]byte, error) {
	for t.scanner.Scan() {
		line := strings.TrimSpace(t.scanner.Text())
		if line == "" {
			continue
		}
		return []byte(line), nil
	}
	if err := t.scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

// Write marshals v as JSON and writes it as a single line to the output stream.
func (t *Transport) Write(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if _, err := t.writer.Write(data); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := t.writer.WriteByte('\n'); err != nil {
		return fmt.Errorf("write newline: %w", err)
	}
	return t.writer.Flush()
}

// Log writes a formatted log message to the logger (stderr).
func (t *Transport) Log(format string, args ...any) {
	fmt.Fprintf(t.logger, format+"\n", args...)
}

// Serve runs the server loop, reading requests from the transport,
// dispatching them through the server, and writing responses back.
// Returns nil when the input stream reaches EOF.
func (s *Server) Serve(tr *Transport) error {
	for {
		data, err := tr.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		resp := s.HandleMessage(data)
		if resp == nil {
			// Notification — no response needed.
			continue
		}

		// Write the raw JSON response (already marshaled by HandleMessage).
		if _, err := tr.writer.Write(resp); err != nil {
			return fmt.Errorf("write: %w", err)
		}
		if err := tr.writer.WriteByte('\n'); err != nil {
			return fmt.Errorf("write newline: %w", err)
		}
		if err := tr.writer.Flush(); err != nil {
			return fmt.Errorf("flush: %w", err)
		}
	}
}
