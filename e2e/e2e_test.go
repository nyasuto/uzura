//go:build e2e

package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/nyasuto/uzura/internal/cdp"
)

const testHTML = `<!DOCTYPE html><html><head><title>Test Page</title></head><body><h1>Hello</h1><div class="content"><p>World</p></div></body></html>`

// startFullServer creates an HTTP test server serving test HTML, and a fully
// wired CDP server. It returns the CDP WebSocket endpoint and the HTML server URL.
func startFullServer(t *testing.T) (wsEndpoint, htmlURL string) {
	t.Helper()

	htmlSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(testHTML))
	}))
	t.Cleanup(htmlSrv.Close)

	s := cdp.NewServer(cdp.WithAddr(":0"), cdp.WithBrowserVersion("Uzura/test"))
	cdp.Setup(s)

	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("start CDP server: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
	})

	return fmt.Sprintf("ws://%s/devtools/page/default", s.Addr()), htmlSrv.URL
}

// e2eDir returns the absolute path to the e2e directory.
func e2eDir(t *testing.T) string {
	t.Helper()
	// This test file lives in e2e/, so use its directory.
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	return dir
}

// runNode runs a Node.js script with the given arguments and returns an error
// if it fails. Stdout and stderr are captured and logged on failure.
func runNode(t *testing.T, script string, args ...string) {
	t.Helper()

	nodeArgs := append([]string{script}, args...)
	cmd := exec.Command("node", nodeArgs...)
	cmd.Dir = e2eDir(t)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd.WaitDelay = time.Second

	if err := cmd.Start(); err != nil {
		t.Fatalf("start node: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		t.Logf("stdout:\n%s", stdout.String())
		if stderr.Len() > 0 {
			t.Logf("stderr:\n%s", stderr.String())
		}
		if err != nil {
			t.Fatalf("node %s failed: %v", filepath.Base(script), err)
		}
	case <-ctx.Done():
		cmd.Process.Kill()
		t.Fatalf("node %s timed out", filepath.Base(script))
	}
}

func TestPuppeteerConnect(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not found, skipping puppeteer test")
	}
	// Check that node_modules exist.
	if _, err := os.Stat(filepath.Join(e2eDir(t), "node_modules")); err != nil {
		t.Skip("node_modules not found, run 'npm install' in e2e/ first")
	}

	wsEndpoint, htmlURL := startFullServer(t)
	t.Logf("CDP: %s, HTML: %s", wsEndpoint, htmlURL)
	runNode(t, "puppeteer_test.mjs", wsEndpoint, htmlURL)
}

func TestPlaywrightConnect(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not found, skipping playwright test")
	}
	if _, err := os.Stat(filepath.Join(e2eDir(t), "node_modules")); err != nil {
		t.Skip("node_modules not found, run 'npm install' in e2e/ first")
	}

	wsEndpoint, htmlURL := startFullServer(t)
	t.Logf("CDP: %s, HTML: %s", wsEndpoint, htmlURL)
	runNode(t, "playwright_test.mjs", wsEndpoint, htmlURL)
}

func TestUnsupportedMethodError(t *testing.T) {
	wsEndpoint, _ := startFullServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsEndpoint, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Send a request for a non-existent method.
	req := map[string]interface{}{
		"id":     1,
		"method": "NonExistent.fakeMethod",
	}
	data, _ := json.Marshal(req)
	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Read response.
	_, respData, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var resp struct {
		ID    int `json:"id"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respData, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("expected error response for unsupported method")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("expected error code -32601, got %d", resp.Error.Code)
	}
	if resp.ID != 1 {
		t.Errorf("expected id 1, got %d", resp.ID)
	}
	t.Logf("Unsupported method error: code=%d message=%q", resp.Error.Code, resp.Error.Message)
}
