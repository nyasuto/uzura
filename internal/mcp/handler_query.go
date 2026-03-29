package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nyasuto/uzura/internal/dom"
)

// QueryParams represents the arguments for the query tool.
type QueryParams struct {
	URL       string `json:"url"`
	Selector  string `json:"selector"`
	Attribute string `json:"attribute,omitempty"`
}

// QueryResult represents a single matched element.
type QueryResult struct {
	Text      string `json:"text"`
	Value     string `json:"value,omitempty"`
	OuterHTML string `json:"outerHTML"`
}

// QueryTool returns the tool definition for the query tool.
func QueryTool() Tool {
	return Tool{
		Name:        "query",
		Description: "CSSセレクターで要素を検索し、テキストや属性を返す",
		InputSchema: json.RawMessage(`{
	"type": "object",
	"properties": {
		"url": {
			"type": "string",
			"description": "対象ページのURL"
		},
		"selector": {
			"type": "string",
			"description": "CSSセレクター"
		},
		"attribute": {
			"type": "string",
			"description": "取得する属性名（省略時はtextContent）"
		}
	},
	"required": ["url", "selector"]
}`),
	}
}

// RegisterQueryTool registers the query tool with its handler on the server.
func RegisterQueryTool(s *Server) {
	s.Tools.RegisterWithHandler(QueryTool(), func(arguments json.RawMessage) (*ToolCallResult, error) {
		return handleQuery(s.Session, arguments)
	})
}

func handleQuery(session *PageSession, arguments json.RawMessage) (*ToolCallResult, error) {
	var params QueryParams
	if err := json.Unmarshal(arguments, &params); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}
	if params.URL == "" {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "url is required"}
	}
	if params.Selector == "" {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "selector is required"}
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

	doc := p.Document()
	if doc == nil {
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: "error: no document loaded"}},
			IsError: true,
		}, nil
	}

	elements, err := doc.DocumentElement().QuerySelectorAll(params.Selector)
	if err != nil {
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("selector error: %s", err)}},
			IsError: true,
		}, nil
	}

	results := make([]QueryResult, 0, len(elements))
	for _, el := range elements {
		qr := QueryResult{
			Text:      el.TextContent(),
			OuterHTML: dom.OuterHTML(el),
		}
		if params.Attribute != "" {
			qr.Value = el.GetAttribute(params.Attribute)
		}
		results = append(results, qr)
	}

	data, err := json.Marshal(results)
	if err != nil {
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("marshal error: %s", err)}},
			IsError: true,
		}, nil
	}

	return &ToolCallResult{
		Content: []ToolContent{{Type: "text", Text: string(data)}},
	}, nil
}
