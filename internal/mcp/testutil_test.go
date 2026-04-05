package mcp_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const defaultTimeout = 30 * time.Second

// mcpProcess manages an uzura mcp subprocess for integration testing.
type mcpProcess struct {
	cmd    *exec.Cmd
	stdin  *json.Encoder
	stdout *bufio.Scanner
	mu     sync.Mutex
	nextID int
}

// buildBinary builds the uzura binary and returns its path.
// The binary is cached per test run using t.TempDir.
func buildBinary(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	bin := filepath.Join(dir, "uzura")

	// Find the module root (where go.mod lives).
	modRoot := findModuleRoot(t)

	cmd := exec.Command("go", "build", "-o", bin, "./cmd/uzura")
	cmd.Dir = modRoot
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
	return bin
}

// findModuleRoot walks up from the current working directory to find go.mod.
func findModuleRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find go.mod in any parent directory")
		}
		dir = parent
	}
}

// startMCP builds and starts an uzura mcp subprocess.
// The process is automatically killed when the test finishes.
func startMCP(t *testing.T) *mcpProcess {
	t.Helper()
	return startMCPWithTimeout(t, defaultTimeout)
}

// startMCPWithTimeout builds and starts an uzura mcp subprocess with a custom timeout.
func startMCPWithTimeout(t *testing.T, timeout time.Duration) *mcpProcess {
	t.Helper()

	bin := buildBinary(t)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	cmd := exec.CommandContext(ctx, bin, "mcp")
	cmd.Stderr = os.Stderr

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		t.Fatalf("stdin pipe: %v", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		t.Fatalf("stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatalf("start uzura mcp: %v", err)
	}

	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

	p := &mcpProcess{
		cmd:    cmd,
		stdin:  json.NewEncoder(stdinPipe),
		stdout: scanner,
		nextID: 1,
	}

	t.Cleanup(func() {
		stdinPipe.Close()
		_ = cmd.Wait()
		cancel()
	})

	return p
}

// initialize performs the MCP initialize handshake.
func (p *mcpProcess) initialize(t *testing.T) {
	t.Helper()

	resp, err := p.sendRequest("initialize", json.RawMessage(`{
		"protocolVersion": "2024-11-05",
		"capabilities": {},
		"clientInfo": {"name": "uzura-test", "version": "1.0"}
	}`))
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("initialize error: %v", resp.Error)
	}

	// Send initialized notification (no response expected).
	p.sendNotification(t, "notifications/initialized")
}

// sendRequest sends a JSON-RPC request and reads the response.
func (p *mcpProcess) sendRequest(method string, params json.RawMessage) (*rpcResponse, error) {
	p.mu.Lock()
	id := p.nextID
	p.nextID++
	p.mu.Unlock()

	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
	}
	if params != nil {
		req["params"] = params
	}

	if err := p.stdin.Encode(req); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}

	if !p.stdout.Scan() {
		if err := p.stdout.Err(); err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}
		return nil, fmt.Errorf("read response: unexpected EOF")
	}

	var resp rpcResponse
	if err := json.Unmarshal(p.stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w (raw: %s)", err, p.stdout.Text())
	}
	return &resp, nil
}

// sendNotification sends a JSON-RPC notification (no response expected).
func (p *mcpProcess) sendNotification(t *testing.T, method string) {
	t.Helper()

	notif := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
	}
	if err := p.stdin.Encode(notif); err != nil {
		t.Fatalf("write notification: %v", err)
	}
}

// callTool sends a tools/call request and returns the result.
func (p *mcpProcess) callTool(t *testing.T, name string, args map[string]any) *toolCallResult {
	t.Helper()

	params := map[string]any{"name": name}
	if args != nil {
		argsJSON, err := json.Marshal(args)
		if err != nil {
			t.Fatalf("marshal args: %v", err)
		}
		params["arguments"] = json.RawMessage(argsJSON)
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}

	resp, err := p.sendRequest("tools/call", json.RawMessage(paramsJSON))
	if err != nil {
		t.Fatalf("callTool %s: %v", name, err)
	}
	if resp.Error != nil {
		t.Fatalf("callTool %s rpc error: %v", name, resp.Error)
	}

	var result toolCallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal tool result: %v (raw: %s)", err, string(resp.Result))
	}
	return &result
}

// rpcResponse is a JSON-RPC 2.0 response for test parsing.
type rpcResponse struct {
	JSONRPC string            `json:"jsonrpc"`
	ID      json.RawMessage   `json:"id,omitempty"`
	Result  json.RawMessage   `json:"result,omitempty"`
	Error   *rpcResponseError `json:"error,omitempty"`
}

// rpcResponseError is a JSON-RPC 2.0 error object for test parsing.
type rpcResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *rpcResponseError) Error() string {
	return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message)
}

// toolCallResult is the result of a tools/call for test parsing.
type toolCallResult struct {
	Content []toolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// Text returns the concatenated text of all content items.
func (r *toolCallResult) Text() string {
	var parts []string
	for _, c := range r.Content {
		if c.Type == "text" {
			parts = append(parts, c.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// toolContent is a content item in a tool call result.
type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
