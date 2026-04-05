package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nyasuto/uzura/internal/markdown"
)

// EvaluateParams represents the arguments for the evaluate tool.
type EvaluateParams struct {
	URL    string `json:"url"`
	Script string `json:"script"`
	Format string `json:"format,omitempty"`
}

// EvaluateTool returns the tool definition for the evaluate tool.
func EvaluateTool() Tool {
	return Tool{
		Name:        "evaluate",
		Description: "ページ上でJavaScriptを実行して結果を返す。format=markdownでJS実行後のDOMからmarkdownを再生成",
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
		},
		"format": {
			"type": "string",
			"enum": ["result", "markdown"],
			"description": "出力形式。result=JS実行結果（デフォルト）、markdown=JS実行後のDOMからmarkdown生成"
		}
	},
	"required": ["url", "script"]
}`),
	}
}

// RegisterEvaluateTool registers the evaluate tool with its handler on the server.
func RegisterEvaluateTool(s *Server) {
	s.Tools.RegisterWithHandler(EvaluateTool(), func(arguments json.RawMessage) (*ToolCallResult, error) {
		return handleEvaluate(s.Session, arguments)
	})
}

func handleEvaluate(session *PageSession, arguments json.RawMessage) (*ToolCallResult, error) {
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

	p, err := session.GetOrNavigate(ctx, params.URL)
	if err != nil {
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("error: %s", err)}},
			IsError: true,
		}, nil
	}

	vm := p.VM()
	val, evalErr := vm.Eval(params.Script)
	if evalErr != nil {
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("js error: %s", evalErr)}},
			IsError: true,
		}, nil
	}

	// If markdown format requested, regenerate markdown from the post-JS DOM
	if params.Format == "markdown" {
		doc := p.Document()
		if doc == nil {
			return &ToolCallResult{
				Content: []ToolContent{{Type: "text", Text: "error: no document after evaluation"}},
				IsError: true,
			}, nil
		}
		md := markdown.RenderWithFallback(doc, params.URL)
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: md}},
		}, nil
	}

	return &ToolCallResult{
		Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("%v", val)}},
	}, nil
}
