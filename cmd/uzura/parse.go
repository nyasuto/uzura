package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nyasuto/uzura/internal/dom"
	htmlparser "github.com/nyasuto/uzura/internal/html"
	"github.com/nyasuto/uzura/internal/markdown"
	"github.com/nyasuto/uzura/internal/semantic"
)

func runParse() error {
	fs := flag.NewFlagSet("parse", flag.ExitOnError)
	format := fs.String("format", "text", "output format: text, json, html, markdown, semantic")
	verbose := fs.Bool("verbose", false, "show token estimate on stderr (markdown only)")
	semanticDepth := fs.Int("semantic-depth", semantic.DefaultMaxDepth, "max depth for semantic tree")
	if err := fs.Parse(os.Args[2:]); err != nil {
		return err
	}

	var r io.Reader
	if fs.NArg() > 0 {
		f, err := os.Open(fs.Arg(0))
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		r = f
	} else {
		r = os.Stdin
	}

	doc, err := htmlparser.Parse(r)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	switch *format {
	case "text":
		printTree(os.Stdout, doc, 0)
	case "json":
		obj := nodeToMap(doc)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(obj)
	case "html":
		_, _ = fmt.Fprint(os.Stdout, dom.Serialize(doc))
	case "markdown":
		md := renderDocMarkdown(doc, "")
		_, _ = fmt.Fprint(os.Stdout, md)
		if *verbose {
			fmt.Fprintf(os.Stderr, "estimated tokens: ~%d\n", estimateTokens(md))
		}
	case "semantic":
		output := renderSemantic(doc, *semanticDepth)
		_, _ = fmt.Fprint(os.Stdout, output)
	default:
		return fmt.Errorf("unknown format: %s", *format)
	}

	return nil
}

// estimateTokens provides a rough token count estimate (~4 chars per token for English).
func estimateTokens(s string) int {
	return (len(s) + 3) / 4
}

func renderDocMarkdown(doc *dom.Document, pageURL string) string {
	return markdown.RenderWithFallback(doc, pageURL)
}

func printTree(w io.Writer, n dom.Node, depth int) {
	indent := strings.Repeat("  ", depth)
	switch v := n.(type) {
	case *dom.Document:
		_, _ = fmt.Fprintf(w, "%s#document\n", indent)
	case *dom.Element:
		attrs := v.Attributes()
		if len(attrs) > 0 {
			var parts []string
			for _, a := range attrs {
				parts = append(parts, fmt.Sprintf("%s=%q", a.Key, a.Val))
			}
			_, _ = fmt.Fprintf(w, "%s<%s %s>\n", indent, v.LocalName(), strings.Join(parts, " "))
		} else {
			_, _ = fmt.Fprintf(w, "%s<%s>\n", indent, v.LocalName())
		}
	case *dom.Text:
		text := strings.TrimSpace(v.Data)
		if text != "" {
			_, _ = fmt.Fprintf(w, "%s\"%s\"\n", indent, text)
		}
	case *dom.Comment:
		_, _ = fmt.Fprintf(w, "%s<!-- %s -->\n", indent, strings.TrimSpace(v.Data))
	}

	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		printTree(w, c, depth+1)
	}
}

func nodeToMap(n dom.Node) map[string]interface{} {
	m := map[string]interface{}{
		"type": nodeTypeName(n),
		"name": n.NodeName(),
	}

	if e, ok := n.(*dom.Element); ok {
		attrs := e.Attributes()
		if len(attrs) > 0 {
			attrMap := make(map[string]string)
			for _, a := range attrs {
				attrMap[a.Key] = a.Val
			}
			m["attributes"] = attrMap
		}
	}

	if t, ok := n.(*dom.Text); ok {
		m["data"] = t.Data
	}
	if c, ok := n.(*dom.Comment); ok {
		m["data"] = c.Data
	}

	children := n.ChildNodes()
	if len(children) > 0 {
		var childList []map[string]interface{}
		for _, c := range children {
			childList = append(childList, nodeToMap(c))
		}
		m["children"] = childList
	}

	return m
}

func renderSemantic(doc *dom.Document, maxDepth int) string {
	b := semantic.NewBuilder()
	nodes := b.Build(doc)
	nodes = semantic.CompressTree(b, nodes, maxDepth)

	var sb strings.Builder

	// Count interactive elements for summary
	interactCount := 0
	countInteractive(nodes, &interactCount)

	for _, n := range nodes {
		sb.WriteString(n.Format(0))
	}

	if sb.Len() == 0 {
		sb.WriteString("(no semantic structure found)\n")
	}

	fmt.Fprintf(&sb, "\n--- %d interactive element(s) ---\n", interactCount)
	return sb.String()
}

func countInteractive(nodes []*semantic.SemanticNode, count *int) {
	for _, n := range nodes {
		switch n.Role {
		case "link", "button", "textbox", "checkbox", "radio", "combobox":
			*count++
		}
		countInteractive(n.Children, count)
	}
}

func nodeTypeName(n dom.Node) string {
	switch n.NodeType() {
	case dom.ElementNode:
		return "element"
	case dom.TextNode:
		return "text"
	case dom.CommentNode:
		return "comment"
	case dom.DocumentNode:
		return "document"
	default:
		return "unknown"
	}
}
