// Package mcp implements the Model Context Protocol (MCP) over JSON-RPC 2.0.
package mcp

import (
	"encoding/json"
	"fmt"
)

const jsonRPCVersion = "2.0"

// JSON-RPC 2.0 error codes.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

// Request represents an incoming JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"` // nil for notifications
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params,omitempty"`
}

// IsNotification returns true if the request has no ID (notification).
func (r *Request) IsNotification() bool {
	return r.ID == nil
}

// Response represents an outgoing JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Error   *RPCError        `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message)
}

// Notification represents an outgoing JSON-RPC 2.0 notification (no ID).
type Notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// ParseRequest parses a raw JSON message into a Request.
func ParseRequest(data []byte) (*Request, error) {
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, &RPCError{Code: CodeParseError, Message: "parse error: " + err.Error()}
	}
	if req.JSONRPC != jsonRPCVersion {
		return nil, &RPCError{Code: CodeInvalidRequest, Message: "invalid jsonrpc version"}
	}
	if req.Method == "" {
		return nil, &RPCError{Code: CodeInvalidRequest, Message: "missing method"}
	}
	return &req, nil
}

// NewResponse creates a successful response for the given request ID.
func NewResponse(id *json.RawMessage, result any) (*Response, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	return &Response{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Result:  data,
	}, nil
}

// NewErrorResponse creates an error response for the given request ID.
func NewErrorResponse(id *json.RawMessage, rpcErr *RPCError) *Response {
	return &Response{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Error:   rpcErr,
	}
}

// NewNotification creates an outgoing notification.
func NewNotification(method string, params any) (*Notification, error) {
	var data json.RawMessage
	if params != nil {
		var err error
		data, err = json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("marshal params: %w", err)
		}
	}
	return &Notification{
		JSONRPC: jsonRPCVersion,
		Method:  method,
		Params:  data,
	}, nil
}
