package cdp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/nyasuto/uzura/internal/cdp"
	"github.com/nyasuto/uzura/internal/network"
	"github.com/nyasuto/uzura/internal/page"
)

const testHTML = `<!DOCTYPE html><html><head><title>DOM Test</title></head>` +
	`<body><div id="main" class="container"><p>Hello</p><p class="info">World</p></div></body></html>`

func startDOMServer(t *testing.T) (srv *cdp.Server, htmlSrv *httptest.Server) {
	t.Helper()

	html := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(testHTML))
	}))
	t.Cleanup(html.Close)

	fetcher := network.NewFetcher(nil)
	p := page.New(&page.Options{Fetcher: fetcher})

	s := cdp.NewServer(cdp.WithAddr(":0"))
	pageDomain := cdp.NewPageDomain(p)
	pageDomain.Register(s)
	domDomain := cdp.NewDOMDomain(p)
	domDomain.Register(s)

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
	})

	return s, html
}

// navigateAndDrain navigates and consumes the response + 2 lifecycle events.
func navigateAndDrain(t *testing.T, ctx context.Context, conn *websocket.Conn, url string) {
	t.Helper()
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id": 100, "method": "Page.navigate",
		"params": map[string]string{"url": url},
	})
	readRPC(t, ctx, conn) // response
	readRPC(t, ctx, conn) // domContentEventFired
	readRPC(t, ctx, conn) // loadEventFired
}

type rpcResponse struct {
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func callRPC(t *testing.T, ctx context.Context, conn *websocket.Conn, id int, method string, params interface{}) rpcResponse {
	t.Helper()
	msg := map[string]interface{}{"id": id, "method": method}
	if params != nil {
		msg["params"] = params
	}
	sendRPC(t, ctx, conn, msg)
	data := readRPC(t, ctx, conn)
	var resp rpcResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	return resp
}

func TestDOMEnable(t *testing.T) {
	s, _ := startDOMServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	resp := callRPC(t, ctx, conn, 1, "DOM.enable", nil)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestDOMGetDocument(t *testing.T) {
	s, html := startDOMServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	navigateAndDrain(t, ctx, conn, html.URL)

	resp := callRPC(t, ctx, conn, 1, "DOM.getDocument", nil)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	var result struct {
		Root struct {
			NodeID         int    `json:"nodeId"`
			NodeType       int    `json:"nodeType"`
			NodeName       string `json:"nodeName"`
			ChildNodeCount int    `json:"childNodeCount"`
			Children       []struct {
				NodeID   int    `json:"nodeId"`
				NodeName string `json:"nodeName"`
			} `json:"children"`
		} `json:"root"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result.Root.NodeType != 9 {
		t.Errorf("root nodeType = %d, want 9 (Document)", result.Root.NodeType)
	}
	if result.Root.NodeName != "#document" {
		t.Errorf("root nodeName = %q, want #document", result.Root.NodeName)
	}
	if result.Root.NodeID == 0 {
		t.Error("root nodeId should not be 0")
	}
	if result.Root.ChildNodeCount == 0 {
		t.Error("root should have children")
	}
}

func TestDOMGetDocumentNoDocument(t *testing.T) {
	s, _ := startDOMServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	resp := callRPC(t, ctx, conn, 1, "DOM.getDocument", nil)
	if resp.Error == nil {
		t.Fatal("expected error when no document loaded")
	}
}

func TestDOMQuerySelector(t *testing.T) {
	s, html := startDOMServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	navigateAndDrain(t, ctx, conn, html.URL)

	// Get document to get root nodeId.
	docResp := callRPC(t, ctx, conn, 1, "DOM.getDocument", map[string]int{"depth": 0})
	var doc struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	json.Unmarshal(docResp.Result, &doc)

	// querySelector for #main
	resp := callRPC(t, ctx, conn, 2, "DOM.querySelector", map[string]interface{}{
		"nodeId":   doc.Root.NodeID,
		"selector": "#main",
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	var qr struct {
		NodeID int `json:"nodeId"`
	}
	json.Unmarshal(resp.Result, &qr)
	if qr.NodeID == 0 {
		t.Error("expected non-zero nodeId for #main")
	}

	// Verify we can get outerHTML of the found node.
	htmlResp := callRPC(t, ctx, conn, 3, "DOM.getOuterHTML", map[string]int{"nodeId": qr.NodeID})
	if htmlResp.Error != nil {
		t.Fatalf("getOuterHTML error: %v", htmlResp.Error)
	}
	var hr struct {
		OuterHTML string `json:"outerHTML"`
	}
	json.Unmarshal(htmlResp.Result, &hr)
	if hr.OuterHTML == "" {
		t.Error("outerHTML should not be empty")
	}
}

func TestDOMQuerySelectorNotFound(t *testing.T) {
	s, html := startDOMServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	navigateAndDrain(t, ctx, conn, html.URL)

	docResp := callRPC(t, ctx, conn, 1, "DOM.getDocument", map[string]int{"depth": 0})
	var doc struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	json.Unmarshal(docResp.Result, &doc)

	resp := callRPC(t, ctx, conn, 2, "DOM.querySelector", map[string]interface{}{
		"nodeId":   doc.Root.NodeID,
		"selector": "#nonexistent",
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	var qr struct {
		NodeID int `json:"nodeId"`
	}
	json.Unmarshal(resp.Result, &qr)
	if qr.NodeID != 0 {
		t.Errorf("expected nodeId 0 for missing element, got %d", qr.NodeID)
	}
}

func TestDOMQuerySelectorAll(t *testing.T) {
	s, html := startDOMServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	navigateAndDrain(t, ctx, conn, html.URL)

	docResp := callRPC(t, ctx, conn, 1, "DOM.getDocument", map[string]int{"depth": 0})
	var doc struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	json.Unmarshal(docResp.Result, &doc)

	resp := callRPC(t, ctx, conn, 2, "DOM.querySelectorAll", map[string]interface{}{
		"nodeId":   doc.Root.NodeID,
		"selector": "p",
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	var qr struct {
		NodeIDs []int `json:"nodeIds"`
	}
	json.Unmarshal(resp.Result, &qr)
	if len(qr.NodeIDs) != 2 {
		t.Fatalf("expected 2 <p> elements, got %d", len(qr.NodeIDs))
	}
	for _, id := range qr.NodeIDs {
		if id == 0 {
			t.Error("nodeId should not be 0")
		}
	}
}

func TestDOMGetOuterHTML(t *testing.T) {
	s, html := startDOMServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	navigateAndDrain(t, ctx, conn, html.URL)

	docResp := callRPC(t, ctx, conn, 1, "DOM.getDocument", map[string]int{"depth": 0})
	var doc struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	json.Unmarshal(docResp.Result, &doc)

	// Find #main and get its outerHTML.
	qResp := callRPC(t, ctx, conn, 2, "DOM.querySelector", map[string]interface{}{
		"nodeId":   doc.Root.NodeID,
		"selector": "#main",
	})
	var qr struct {
		NodeID int `json:"nodeId"`
	}
	json.Unmarshal(qResp.Result, &qr)

	resp := callRPC(t, ctx, conn, 3, "DOM.getOuterHTML", map[string]int{"nodeId": qr.NodeID})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	var hr struct {
		OuterHTML string `json:"outerHTML"`
	}
	json.Unmarshal(resp.Result, &hr)

	// Should contain the div with its children.
	if hr.OuterHTML == "" {
		t.Fatal("outerHTML should not be empty")
	}
	// The outerHTML should contain "container" class and "Hello" text.
	if !contains(hr.OuterHTML, "container") {
		t.Errorf("outerHTML should contain 'container', got %q", hr.OuterHTML)
	}
	if !contains(hr.OuterHTML, "Hello") {
		t.Errorf("outerHTML should contain 'Hello', got %q", hr.OuterHTML)
	}
}

func TestDOMGetAttributes(t *testing.T) {
	s, html := startDOMServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	navigateAndDrain(t, ctx, conn, html.URL)

	docResp := callRPC(t, ctx, conn, 1, "DOM.getDocument", map[string]int{"depth": 0})
	var doc struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	json.Unmarshal(docResp.Result, &doc)

	qResp := callRPC(t, ctx, conn, 2, "DOM.querySelector", map[string]interface{}{
		"nodeId":   doc.Root.NodeID,
		"selector": "#main",
	})
	var qr struct {
		NodeID int `json:"nodeId"`
	}
	json.Unmarshal(qResp.Result, &qr)

	resp := callRPC(t, ctx, conn, 3, "DOM.getAttributes", map[string]int{"nodeId": qr.NodeID})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	var ar struct {
		Attributes []string `json:"attributes"`
	}
	json.Unmarshal(resp.Result, &ar)

	// Expect flat [name, value, name, value, ...] — should have id=main, class=container.
	attrMap := make(map[string]string)
	for i := 0; i+1 < len(ar.Attributes); i += 2 {
		attrMap[ar.Attributes[i]] = ar.Attributes[i+1]
	}
	if attrMap["id"] != "main" {
		t.Errorf("id = %q, want %q", attrMap["id"], "main")
	}
	if attrMap["class"] != "container" {
		t.Errorf("class = %q, want %q", attrMap["class"], "container")
	}
}

func TestDOMSetAttributeValue(t *testing.T) {
	s, html := startDOMServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	navigateAndDrain(t, ctx, conn, html.URL)

	docResp := callRPC(t, ctx, conn, 1, "DOM.getDocument", map[string]int{"depth": 0})
	var doc struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	json.Unmarshal(docResp.Result, &doc)

	qResp := callRPC(t, ctx, conn, 2, "DOM.querySelector", map[string]interface{}{
		"nodeId":   doc.Root.NodeID,
		"selector": "#main",
	})
	var qr struct {
		NodeID int `json:"nodeId"`
	}
	json.Unmarshal(qResp.Result, &qr)

	// Set a new attribute.
	resp := callRPC(t, ctx, conn, 3, "DOM.setAttributeValue", map[string]interface{}{
		"nodeId": qr.NodeID,
		"name":   "data-test",
		"value":  "hello",
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	// Verify via getAttributes.
	attrResp := callRPC(t, ctx, conn, 4, "DOM.getAttributes", map[string]int{"nodeId": qr.NodeID})
	var ar struct {
		Attributes []string `json:"attributes"`
	}
	json.Unmarshal(attrResp.Result, &ar)

	attrMap := make(map[string]string)
	for i := 0; i+1 < len(ar.Attributes); i += 2 {
		attrMap[ar.Attributes[i]] = ar.Attributes[i+1]
	}
	if attrMap["data-test"] != "hello" {
		t.Errorf("data-test = %q, want %q", attrMap["data-test"], "hello")
	}
}

func TestDOMRemoveAttribute(t *testing.T) {
	s, html := startDOMServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	navigateAndDrain(t, ctx, conn, html.URL)

	docResp := callRPC(t, ctx, conn, 1, "DOM.getDocument", map[string]int{"depth": 0})
	var doc struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	json.Unmarshal(docResp.Result, &doc)

	qResp := callRPC(t, ctx, conn, 2, "DOM.querySelector", map[string]interface{}{
		"nodeId":   doc.Root.NodeID,
		"selector": "#main",
	})
	var qr struct {
		NodeID int `json:"nodeId"`
	}
	json.Unmarshal(qResp.Result, &qr)

	// Remove the class attribute.
	resp := callRPC(t, ctx, conn, 3, "DOM.removeAttribute", map[string]interface{}{
		"nodeId": qr.NodeID,
		"name":   "class",
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	// Verify class is gone.
	attrResp := callRPC(t, ctx, conn, 4, "DOM.getAttributes", map[string]int{"nodeId": qr.NodeID})
	var ar struct {
		Attributes []string `json:"attributes"`
	}
	json.Unmarshal(attrResp.Result, &ar)

	for i := 0; i+1 < len(ar.Attributes); i += 2 {
		if ar.Attributes[i] == "class" {
			t.Error("class attribute should have been removed")
		}
	}
}

func TestDOMRequestChildNodes(t *testing.T) {
	s, html := startDOMServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	navigateAndDrain(t, ctx, conn, html.URL)

	// Get document with depth 0 (no children inline).
	docResp := callRPC(t, ctx, conn, 1, "DOM.getDocument", map[string]int{"depth": 0})
	var doc struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	json.Unmarshal(docResp.Result, &doc)

	// Request children — response comes first, then DOM.setChildNodes event.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     2,
		"method": "DOM.requestChildNodes",
		"params": map[string]int{"nodeId": doc.Root.NodeID},
	})

	// Read RPC response.
	respData := readRPC(t, ctx, conn)
	var resp rpcResponse
	json.Unmarshal(respData, &resp)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	// Read DOM.setChildNodes event.
	evtData := readRPC(t, ctx, conn)
	var evt struct {
		Method string `json:"method"`
		Params struct {
			ParentID int `json:"parentId"`
			Nodes    []struct {
				NodeID   int    `json:"nodeId"`
				NodeName string `json:"nodeName"`
			} `json:"nodes"`
		} `json:"params"`
	}
	if err := json.Unmarshal(evtData, &evt); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	if evt.Method != "DOM.setChildNodes" {
		t.Errorf("event method = %q, want DOM.setChildNodes", evt.Method)
	}
	if evt.Params.ParentID != doc.Root.NodeID {
		t.Errorf("parentId = %d, want %d", evt.Params.ParentID, doc.Root.NodeID)
	}
	if len(evt.Params.Nodes) == 0 {
		t.Error("expected child nodes")
	}
}

func TestDOMNodeNotFound(t *testing.T) {
	s, html := startDOMServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	navigateAndDrain(t, ctx, conn, html.URL)

	// Use invalid nodeId.
	resp := callRPC(t, ctx, conn, 1, "DOM.getOuterHTML", map[string]int{"nodeId": 9999})
	if resp.Error == nil {
		t.Fatal("expected error for invalid nodeId")
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
