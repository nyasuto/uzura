package dom

import (
	"strings"
)

// voidElements are elements that cannot have children and have no closing tag.
var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true,
	"embed": true, "hr": true, "img": true, "input": true,
	"link": true, "meta": true, "param": true, "source": true,
	"track": true, "wbr": true,
}

// rawTextElements contain raw text that should not be escaped.
var rawTextElements = map[string]bool{
	"script": true, "style": true,
}

// Serialize converts a DOM node subtree to an HTML string.
func Serialize(n Node) string {
	var sb strings.Builder
	serializeNode(&sb, n)
	return sb.String()
}

func serializeNode(sb *strings.Builder, n Node) {
	switch v := n.(type) {
	case *Document:
		serializeChildren(sb, v)
	case *Element:
		serializeElement(sb, v)
	case *Text:
		if p, ok := v.ParentNode().(*Element); ok && rawTextElements[p.LocalName()] {
			sb.WriteString(v.Data)
		} else {
			sb.WriteString(escapeText(v.Data))
		}
	case *Comment:
		sb.WriteString("<!--")
		sb.WriteString(v.Data)
		sb.WriteString("-->")
	}
}

func serializeElement(sb *strings.Builder, e *Element) {
	tag := e.LocalName()
	sb.WriteByte('<')
	sb.WriteString(tag)

	for _, attr := range e.Attributes() {
		sb.WriteByte(' ')
		sb.WriteString(attr.Key)
		sb.WriteString(`="`)
		sb.WriteString(escapeAttr(attr.Val))
		sb.WriteByte('"')
	}
	sb.WriteByte('>')

	if voidElements[tag] {
		return
	}

	serializeChildren(sb, e)

	sb.WriteString("</")
	sb.WriteString(tag)
	sb.WriteByte('>')
}

func serializeChildren(sb *strings.Builder, n Node) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		serializeNode(sb, c)
	}
}

func escapeText(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func escapeAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// InnerHTML returns the serialized HTML of the node's children.
func InnerHTML(n Node) string {
	var sb strings.Builder
	serializeChildren(&sb, n)
	return sb.String()
}

// OuterHTML returns the serialized HTML of the node itself.
func OuterHTML(n Node) string {
	return Serialize(n)
}
