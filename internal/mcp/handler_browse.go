package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nyasuto/uzura/internal/dom"
	htmlparser "github.com/nyasuto/uzura/internal/html"
	"github.com/nyasuto/uzura/internal/markdown"
)

const browseTimeout = 30 * time.Second

// BrowseParams represents the arguments for the browse tool.
type BrowseParams struct {
	URL    string `json:"url"`
	Format string `json:"format,omitempty"`
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
		output = doc.DocumentElement().TextContent()
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
	meta := markdown.ExtractMetadata(doc, pageURL)

	// Try readability extraction first
	result, err := markdown.Extract(doc, pageURL)
	if err == nil && result.Content != "" {
		// Use readability-cleaned content: parse extracted HTML, convert to markdown
		extractedDoc, parseErr := parseExtractedHTML(result.Content)
		if parseErr == nil {
			markdown.Clean(extractedDoc, true)
			body := markdown.Convert(extractedDoc)
			var sb strings.Builder
			sb.WriteString(markdown.FormatFrontmatter(meta))
			sb.WriteString("\n")
			sb.WriteString(body)
			return sb.String()
		}
	}

	// Fallback: clean and convert the full page
	cloned, ok := doc.CloneNode(true).(*dom.Document)
	if !ok {
		return markdown.FormatFrontmatter(meta) + "\n" + doc.DocumentElement().TextContent()
	}
	markdown.Clean(cloned, true)
	body := markdown.Convert(cloned)

	var sb strings.Builder
	sb.WriteString(markdown.FormatFrontmatter(meta))
	sb.WriteString("\n")
	sb.WriteString(body)
	return sb.String()
}

func parseExtractedHTML(content string) (*dom.Document, error) {
	r := strings.NewReader("<html><body>" + content + "</body></html>")
	htmlParser := htmlparser.Parse
	return htmlParser(r)
}
