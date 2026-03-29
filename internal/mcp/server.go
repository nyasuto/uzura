package mcp

import (
	"encoding/json"
	"fmt"
	"sync"
)

const (
	protocolVersion = "2024-11-05"
	serverName      = "uzura"
	serverVersion   = "0.1.0"
)

// Handler is a function that handles an MCP method call.
type Handler func(params json.RawMessage) (any, error)

// Server handles MCP JSON-RPC requests.
type Server struct {
	mu          sync.RWMutex
	handlers    map[string]Handler
	initialized bool
	Tools       *ToolRegistry
	Session     *PageSession
}

// NewServer creates a new MCP server with built-in handlers.
func NewServer() *Server {
	s := &Server{
		handlers: make(map[string]Handler),
		Tools:    NewToolRegistry(),
		Session:  NewPageSession(),
	}
	s.registerBuiltins()
	return s
}

// Handle registers a handler for the given method.
func (s *Server) Handle(method string, h Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = h
}

// HandleMessage processes a raw JSON-RPC message and returns a response.
// Returns nil for notifications that need no response.
func (s *Server) HandleMessage(data []byte) []byte {
	req, err := ParseRequest(data)
	if err != nil {
		rpcErr, ok := err.(*RPCError)
		if !ok {
			rpcErr = &RPCError{Code: CodeInternalError, Message: err.Error()}
		}
		resp := NewErrorResponse(nil, rpcErr)
		out, _ := json.Marshal(resp)
		return out
	}

	// Notifications that require no response.
	if req.IsNotification() {
		s.handleNotification(req)
		return nil
	}

	resp := s.dispatch(req)
	out, _ := json.Marshal(resp)
	return out
}

func (s *Server) handleNotification(req *Request) {
	if req.Method == "notifications/initialized" {
		s.mu.Lock()
		s.initialized = true
		s.mu.Unlock()
	}
}

func (s *Server) dispatch(req *Request) *Response {
	s.mu.RLock()
	h, ok := s.handlers[req.Method]
	s.mu.RUnlock()

	if !ok {
		return NewErrorResponse(req.ID, &RPCError{
			Code:    CodeMethodNotFound,
			Message: fmt.Sprintf("method not found: %s", req.Method),
		})
	}

	result, err := h(req.Params)
	if err != nil {
		rpcErr, ok := err.(*RPCError)
		if !ok {
			rpcErr = &RPCError{Code: CodeInternalError, Message: err.Error()}
		}
		return NewErrorResponse(req.ID, rpcErr)
	}

	resp, marshalErr := NewResponse(req.ID, result)
	if marshalErr != nil {
		return NewErrorResponse(req.ID, &RPCError{
			Code:    CodeInternalError,
			Message: marshalErr.Error(),
		})
	}
	return resp
}

func (s *Server) registerBuiltins() {
	s.Handle("initialize", s.handleInitialize)
	s.Handle("ping", s.handlePing)
	s.Handle("tools/list", s.handleToolsList)
	s.Handle("tools/call", s.handleToolsCall)
}

// InitializeParams represents the client's initialize request parameters.
type InitializeParams struct {
	ProtocolVersion string     `json:"protocolVersion"`
	Capabilities    ClientCaps `json:"capabilities"`
	ClientInfo      AppInfo    `json:"clientInfo"`
}

// ClientCaps represents client capabilities.
type ClientCaps struct {
	Roots    *RootsCap    `json:"roots,omitempty"`
	Sampling *SamplingCap `json:"sampling,omitempty"`
}

// RootsCap represents client root capabilities.
type RootsCap struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// SamplingCap represents client sampling capabilities.
type SamplingCap struct{}

// AppInfo identifies an application by name and version.
type AppInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ServerCaps represents server capabilities.
type ServerCaps struct {
	Tools *ToolsCap `json:"tools,omitempty"`
}

// ToolsCap represents server tool capabilities.
type ToolsCap struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// InitializeResult is the response to an initialize request.
type InitializeResult struct {
	ProtocolVersion string     `json:"protocolVersion"`
	Capabilities    ServerCaps `json:"capabilities"`
	ServerInfo      AppInfo    `json:"serverInfo"`
}

func (s *Server) handleInitialize(params json.RawMessage) (any, error) {
	var p InitializeParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
	}

	return &InitializeResult{
		ProtocolVersion: protocolVersion,
		Capabilities: ServerCaps{
			Tools: &ToolsCap{},
		},
		ServerInfo: AppInfo{
			Name:    serverName,
			Version: serverVersion,
		},
	}, nil
}

func (s *Server) handlePing(_ json.RawMessage) (any, error) {
	return struct{}{}, nil
}

func (s *Server) handleToolsList(_ json.RawMessage) (any, error) {
	return &ToolsListResult{Tools: s.Tools.List()}, nil
}

func (s *Server) handleToolsCall(params json.RawMessage) (any, error) {
	var p ToolCallParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
	}
	return s.Tools.Call(p.Name, p.Arguments)
}
