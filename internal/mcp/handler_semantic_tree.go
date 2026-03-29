package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nyasuto/uzura/internal/semantic"
)

// SemanticTreeParams represents arguments for the semantic_tree tool.
type SemanticTreeParams struct {
	URL      string `json:"url"`
	MaxDepth int    `json:"max_depth,omitempty"`
}

// SemanticTreeTool returns the tool definition for semantic_tree.
func SemanticTreeTool() Tool {
	return Tool{
		Name:        "semantic_tree",
		Description: "ページの論理構造を操作可能な要素付きで返す",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"url": {
					"type": "string",
					"description": "取得するURL"
				},
				"max_depth": {
					"type": "integer",
					"default": 10,
					"description": "ツリーの最大深さ"
				}
			},
			"required": ["url"]
		}`),
	}
}

// RegisterSemanticTreeTool registers the semantic_tree tool with its handler.
func RegisterSemanticTreeTool(s *Server) {
	s.Tools.RegisterWithHandler(SemanticTreeTool(), func(arguments json.RawMessage) (*ToolCallResult, error) {
		return handleSemanticTree(s.Session, arguments)
	})
}

func handleSemanticTree(session *PageSession, arguments json.RawMessage) (*ToolCallResult, error) {
	var params SemanticTreeParams
	if err := json.Unmarshal(arguments, &params); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}

	if params.URL == "" {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "url is required"}
	}
	if params.MaxDepth <= 0 {
		params.MaxDepth = semantic.DefaultMaxDepth
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

	builder := semantic.NewBuilder()
	nodes := builder.Build(doc)
	nodes = semantic.CompressTree(builder, nodes, params.MaxDepth)

	// Store NodeMap in session for interact tool to use
	session.SetNodeMap(builder.NodeMap)

	var sb strings.Builder
	for _, n := range nodes {
		sb.WriteString(n.Format(0))
	}

	output := sb.String()
	if output == "" {
		output = "(no semantic structure found)"
	}

	return &ToolCallResult{
		Content: []ToolContent{{Type: "text", Text: output}},
	}, nil
}
