package cdp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/coder/websocket"
)

// Server is a CDP WebSocket server.
type Server struct {
	mu              sync.RWMutex
	handlers        map[string]Handler
	sessionHandlers map[string]SessionHandler
	addr            string
	listener        net.Listener
	srv             *http.Server

	// Browser metadata for discovery endpoints.
	browserVersion  string
	protocolVersion string
	userAgent       string
	webSocketURL    string
	debugLog        bool
}

// ServerOption configures a Server.
type ServerOption func(*Server)

// WithAddr sets the listen address (default ":9222").
func WithAddr(addr string) ServerOption {
	return func(s *Server) { s.addr = addr }
}

// WithBrowserVersion sets the browser version string.
func WithBrowserVersion(v string) ServerOption {
	return func(s *Server) { s.browserVersion = v }
}

// WithDebugLog enables debug logging to stderr.
func WithDebugLog(enable bool) ServerOption {
	return func(s *Server) { s.debugLog = enable }
}

// NewServer creates a CDP server with the given options.
func NewServer(opts ...ServerOption) *Server {
	s := &Server{
		handlers:        make(map[string]Handler),
		sessionHandlers: make(map[string]SessionHandler),
		addr:            ":9222",
		browserVersion:  "Uzura/dev",
		protocolVersion: "1.3",
		userAgent:       "Uzura",
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Handle registers a handler for a CDP method (e.g. "Page.navigate").
func (s *Server) Handle(method string, h Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = h
}

// HandleSession registers a session-aware handler for a CDP method.
// Session handlers have access to the client connection for pushing events.
func (s *Server) HandleSession(method string, h SessionHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionHandlers[method] = h
}

// Start begins listening. It returns once the listener is ready.
// Use Shutdown to stop the server.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/json/version", s.handleVersion)
	mux.HandleFunc("/json/list", s.handleList)
	mux.HandleFunc("/json", s.handleList)
	mux.HandleFunc("/json/protocol", s.handleProtocol)
	mux.HandleFunc("/devtools/page/", s.handleWebSocket)

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("cdp listen: %w", err)
	}
	s.listener = ln
	s.webSocketURL = fmt.Sprintf("ws://%s/devtools/page/default", ln.Addr().String())

	s.srv = &http.Server{Handler: mux}
	go func() {
		_ = s.srv.Serve(ln)
	}()
	return nil
}

// Addr returns the listener address, available after Start.
func (s *Server) Addr() string {
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.srv == nil {
		return nil
	}
	return s.srv.Shutdown(ctx)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	ctx := r.Context()
	sess := newSession(ctx, conn)

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}

		var req Request
		if err := json.Unmarshal(data, &req); err != nil {
			resp := Response{
				Error: &RPCError{Code: -32700, Message: "parse error"},
			}
			s.writeJSON(ctx, conn, resp)
			continue
		}

		resp, events := s.dispatch(sess, req)
		resp.SessionID = req.SessionID
		s.writeJSON(ctx, conn, resp)
		for _, evt := range events {
			if evt.SessionID == "" {
				evt.SessionID = req.SessionID
			}
			s.writeJSON(ctx, conn, evt)
		}
	}
}

func (s *Server) dispatch(sess *Session, req Request) (Response, []Event) {
	if s.debugLog {
		log.Printf("[CDP] → %s (id=%d session=%q)", req.Method, req.ID, req.SessionID)
	}
	// Check session-aware handlers first.
	s.mu.RLock()
	sh, shOK := s.sessionHandlers[req.Method]
	h, hOK := s.handlers[req.Method]
	s.mu.RUnlock()

	if shOK {
		result, events, err := sh(sess, req.Params)
		if err != nil {
			return Response{
				ID:    req.ID,
				Error: &RPCError{Code: -32000, Message: err.Error()},
			}, nil
		}
		return Response{ID: req.ID, Result: result}, events
	}

	if hOK {
		result, err := h(req.Params)
		if err != nil {
			return Response{
				ID:    req.ID,
				Error: &RPCError{Code: -32000, Message: err.Error()},
			}, nil
		}
		return Response{ID: req.ID, Result: result}, nil
	}

	return Response{
		ID:    req.ID,
		Error: &RPCError{Code: -32601, Message: fmt.Sprintf("method not found: %s", req.Method)},
	}, nil
}

func (s *Server) writeJSON(ctx context.Context, conn *websocket.Conn, v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	_ = conn.Write(ctx, websocket.MessageText, data)
}
