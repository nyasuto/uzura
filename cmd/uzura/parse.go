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
)

func runParse() error {
	fs := flag.NewFlagSet("parse", flag.ExitOnError)
	format := fs.String("format", "text", "output format: text, json, html")
	if err := fs.Parse(os.Args[2:]); err != nil {
		return err
	}

	var r io.Reader
	if fs.NArg() > 0 {
		f, err := os.Open(fs.Arg(0))
		if err != nil {
			return err
		}
		defer f.Close()
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
		fmt.Fprint(os.Stdout, dom.Serialize(doc))
	default:
		return fmt.Errorf("unknown format: %s", *format)
	}

	return nil
}

func printTree(w io.Writer, n dom.Node, depth int) {
	indent := strings.Repeat("  ", depth)
	switch v := n.(type) {
	case *dom.Document:
		fmt.Fprintf(w, "%s#document\n", indent)
	case *dom.Element:
		attrs := v.Attributes()
		if len(attrs) > 0 {
			var parts []string
			for _, a := range attrs {
				parts = append(parts, fmt.Sprintf(`%s="%s"`, a.Key, a.Val))
			}
			fmt.Fprintf(w, "%s<%s %s>\n", indent, v.LocalName(), strings.Join(parts, " "))
		} else {
			fmt.Fprintf(w, "%s<%s>\n", indent, v.LocalName())
		}
	case *dom.Text:
		text := strings.TrimSpace(v.Data)
		if text != "" {
			fmt.Fprintf(w, "%s\"%s\"\n", indent, text)
		}
	case *dom.Comment:
		fmt.Fprintf(w, "%s<!-- %s -->\n", indent, strings.TrimSpace(v.Data))
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
