package mcp

import (
	"encoding/json"
	"fmt"
	"sync"
)

// Tool represents an MCP tool definition.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ToolsListResult is the response to a tools/list request.
type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

// ToolCallParams represents the parameters of a tools/call request.
type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// ToolCallResult is the response to a tools/call request.
type ToolCallResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ToolContent represents a content item in a tool call result.
type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ToolHandler is a function that executes a tool call.
type ToolHandler func(arguments json.RawMessage) (*ToolCallResult, error)

// ToolRegistry manages tool definitions and their handlers.
type ToolRegistry struct {
	mu       sync.RWMutex
	tools    map[string]Tool
	handlers map[string]ToolHandler
	order    []string // preserves registration order
}

// NewToolRegistry creates a new empty tool registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools:    make(map[string]Tool),
		handlers: make(map[string]ToolHandler),
	}
}

// Register adds or replaces a tool definition.
func (r *ToolRegistry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[tool.Name]; !exists {
		r.order = append(r.order, tool.Name)
	}
	r.tools[tool.Name] = tool
}

// RegisterWithHandler adds a tool definition and its execution handler.
func (r *ToolRegistry) RegisterWithHandler(tool Tool, handler ToolHandler) {
	r.Register(tool)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[tool.Name] = handler
}

// List returns all registered tools in registration order.
func (r *ToolRegistry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Tool, 0, len(r.order))
	for _, name := range r.order {
		result = append(result, r.tools[name])
	}
	return result
}

// Call executes a tool by name with the given arguments.
func (r *ToolRegistry) Call(name string, arguments json.RawMessage) (*ToolCallResult, error) {
	r.mu.RLock()
	handler, ok := r.handlers[name]
	r.mu.RUnlock()

	if !ok {
		return nil, &RPCError{
			Code:    CodeInvalidParams,
			Message: fmt.Sprintf("unknown tool: %s", name),
		}
	}
	return handler(arguments)
}

// BrowseTool returns the tool definition for the browse tool.
func BrowseTool() Tool {
	return Tool{
		Name:        "browse",
		Description: "URLを開いてページのコンテンツを取得する",
		InputSchema: json.RawMessage(`{
	"type": "object",
	"properties": {
		"url": {
			"type": "string",
			"description": "取得するURL"
		},
		"format": {
			"type": "string",
			"enum": ["text", "html", "json", "markdown"],
			"default": "text",
			"description": "出力フォーマット"
		}
	},
	"required": ["url"]
}`),
	}
}
