package markdown

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nyasuto/uzura/internal/dom"
)

// Metadata holds page metadata extracted from HTML.
type Metadata struct {
	Title       string
	Description string
	Author      string
	URL         string
	OGTitle     string
	OGDesc      string
	OGImage     string
}

// ExtractMetadata extracts metadata from a DOM document's <head>.
func ExtractMetadata(doc *dom.Document, pageURL string) *Metadata {
	m := &Metadata{URL: pageURL}

	head := doc.Head()
	if head == nil {
		m.Title = doc.Title()
		return m
	}

	m.Title = doc.Title()

	for child := head.FirstChild(); child != nil; child = child.NextSibling() {
		el, ok := child.(*dom.Element)
		if !ok {
			continue
		}

		switch el.LocalName() {
		case "meta":
			extractMeta(el, m)
		case "script":
			if el.GetAttribute("type") == "application/ld+json" {
				extractJSONLD(el, m)
			}
		}
	}

	return m
}

func extractMeta(el *dom.Element, m *Metadata) {
	name := strings.ToLower(el.GetAttribute("name"))
	property := strings.ToLower(el.GetAttribute("property"))
	content := el.GetAttribute("content")

	switch name {
	case "description":
		m.Description = content
	case "author":
		m.Author = content
	}

	switch property {
	case "og:title":
		m.OGTitle = content
	case "og:description":
		m.OGDesc = content
	case "og:image":
		m.OGImage = content
	}
}

func extractJSONLD(el *dom.Element, m *Metadata) {
	text := el.TextContent()
	if text == "" {
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		return
	}

	if v, ok := data["headline"].(string); ok && m.Title == "" {
		m.Title = v
	}
	if v, ok := data["description"].(string); ok && m.Description == "" {
		m.Description = v
	}
	if v, ok := data["author"]; ok && m.Author == "" {
		m.Author = extractAuthorFromJSONLD(v)
	}
}

func extractAuthorFromJSONLD(v interface{}) string {
	switch a := v.(type) {
	case string:
		return a
	case map[string]interface{}:
		if name, ok := a["name"].(string); ok {
			return name
		}
	case []interface{}:
		if len(a) > 0 {
			return extractAuthorFromJSONLD(a[0])
		}
	}
	return ""
}

// FormatFrontmatter produces a YAML-like frontmatter block from metadata.
func FormatFrontmatter(m *Metadata) string {
	var sb strings.Builder
	sb.WriteString("---\n")

	title := m.OGTitle
	if title == "" {
		title = m.Title
	}
	if title != "" {
		fmt.Fprintf(&sb, "title: %s\n", title)
	}

	if m.Author != "" {
		fmt.Fprintf(&sb, "author: %s\n", m.Author)
	}

	desc := m.OGDesc
	if desc == "" {
		desc = m.Description
	}
	if desc != "" {
		fmt.Fprintf(&sb, "description: %s\n", desc)
	}

	if m.URL != "" {
		fmt.Fprintf(&sb, "url: %s\n", m.URL)
	}

	if m.OGImage != "" {
		fmt.Fprintf(&sb, "image: %s\n", m.OGImage)
	}

	sb.WriteString("---\n")
	return sb.String()
}
