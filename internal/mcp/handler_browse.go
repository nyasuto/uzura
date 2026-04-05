package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nyasuto/uzura/internal/dom"
	"github.com/nyasuto/uzura/internal/markdown"
)

const browseTimeout = 30 * time.Second

// defaultMaxLength is the default maximum output size in bytes (~100KB).
const defaultMaxLength = 100 * 1024

// BrowseParams represents the arguments for the browse tool.
type BrowseParams struct {
	URL       string `json:"url"`
	Format    string `json:"format,omitempty"`
	MaxLength int    `json:"max_length,omitempty"`
}

// RegisterBrowseTool registers the browse tool with its handler on the server.
func RegisterBrowseTool(s *Server) {
	s.Tools.RegisterWithHandler(BrowseTool(), func(arguments json.RawMessage) (*ToolCallResult, error) {
		return handleBrowse(s.Session, arguments)
	})
}

func handleBrowse(session *PageSession, arguments json.RawMessage) (*ToolCallResult, error) {
	var params BrowseParams
	if err := json.Unmarshal(arguments, &params); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}
	if params.URL == "" {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "url is required"}
	}
	if params.Format == "" {
		params.Format = "text"
	}

	ctx, cancel := context.WithTimeout(context.Background(), browseTimeout)
	defer cancel()

	p, err := session.GetOrNavigate(ctx, params.URL)
	if err != nil {
		// On timeout, try returning partial result from cache
		if errors.Is(err, context.DeadlineExceeded) {
			if cached := session.GetCached(params.URL); cached != nil && cached.Document() != nil {
				return &ToolCallResult{
					Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("[partial result - timeout]\n%s",
						dom.CleanTextContent(cached.Document().DocumentElement()))}},
				}, nil
			}
		}
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

	var output string
	switch params.Format {
	case "html":
		output = dom.Serialize(doc)
	case "json":
		output = serializeDocJSON(doc)
	case "markdown":
		output = renderMarkdown(doc, params.URL)
	default: // "text"
		output = dom.CleanTextContent(doc.DocumentElement())
	}

	// Apply default max length if not specified
	maxLen := params.MaxLength
	if maxLen <= 0 {
		maxLen = defaultMaxLength
	}
	if len(output) > maxLen {
		output = output[:maxLen] + "\n\n[truncated]"
	}

	return &ToolCallResult{
		Content: []ToolContent{{Type: "text", Text: output}},
	}, nil
}

func serializeDocJSON(doc *dom.Document) string {
	result := docToMap(doc.DocumentElement())
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error())
	}
	return string(data)
}

func docToMap(n dom.Node) map[string]any {
	m := map[string]any{
		"nodeName": n.NodeName(),
	}

	if el, ok := n.(*dom.Element); ok {
		attrs := el.Attributes()
		if len(attrs) > 0 {
			attrMap := make(map[string]string, len(attrs))
			for _, a := range attrs {
				attrMap[a.Key] = a.Val
			}
			m["attributes"] = attrMap
		}
	}

	if n.NodeType() == dom.TextNode {
		m["text"] = n.TextContent()
		return m
	}

	children := n.ChildNodes()
	if len(children) > 0 {
		kids := make([]map[string]any, 0, len(children))
		for _, child := range children {
			kids = append(kids, docToMap(child))
		}
		m["children"] = kids
	}
	return m
}

func renderMarkdown(doc *dom.Document, pageURL string) string {
	return markdown.RenderWithFallback(doc, pageURL)
}
