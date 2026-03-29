package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nyasuto/uzura/internal/page"
)

// EvaluateParams represents the arguments for the evaluate tool.
type EvaluateParams struct {
	URL    string `json:"url"`
	Script string `json:"script"`
}

// EvaluateTool returns the tool definition for the evaluate tool.
func EvaluateTool() Tool {
	return Tool{
		Name:        "evaluate",
		Description: "ページ上でJavaScriptを実行して結果を返す",
		InputSchema: json.RawMessage(`{
	"type": "object",
	"properties": {
		"url": {
			"type": "string",
			"description": "対象ページのURL"
		},
		"script": {
			"type": "string",
			"description": "実行するJavaScript式"
		}
	},
	"required": ["url", "script"]
}`),
	}
}

// RegisterEvaluateTool registers the evaluate tool with its handler on the server.
func RegisterEvaluateTool(s *Server) {
	s.Tools.RegisterWithHandler(EvaluateTool(), handleEvaluate)
}

func handleEvaluate(arguments json.RawMessage) (*ToolCallResult, error) {
	var params EvaluateParams
	if err := json.Unmarshal(arguments, &params); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}
	if params.URL == "" {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "url is required"}
	}
	if params.Script == "" {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "script is required"}
	}

	ctx, cancel := context.WithTimeout(context.Background(), browseTimeout)
	defer cancel()

	p := page.New(nil)
	defer p.Close()

	if err := p.Navigate(ctx, params.URL); err != nil {
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("error: %s", err)}},
			IsError: true,
		}, nil
	}

	vm := p.VM()
	val, err := vm.Eval(params.Script)
	if err != nil {
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("js error: %s", err)}},
			IsError: true,
		}, nil
	}

	return &ToolCallResult{
		Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("%v", val)}},
	}, nil
}
