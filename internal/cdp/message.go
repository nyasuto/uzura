// Package cdp implements a Chrome DevTools Protocol server.
package cdp

import "encoding/json"

// Request represents an incoming CDP JSON-RPC request.
type Request struct {
	ID     int64           `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Response represents an outgoing CDP JSON-RPC response.
type Response struct {
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *RPCError       `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Event represents a CDP event pushed to the client.
type Event struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Handler processes a CDP method call and returns a result or error.
type Handler func(params json.RawMessage) (json.RawMessage, error)
