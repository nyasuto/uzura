package markdown

import (
	"fmt"
	"strings"

	"github.com/nyasuto/uzura/internal/dom"
)

// Convert transforms a DOM node subtree into Markdown text.
func Convert(node dom.Node) string {
	var sb strings.Builder
	c := &converter{buf: &sb}
	c.walk(node)
	return strings.TrimSpace(sb.String()) + "\n"
}

type converter struct {
	buf       *strings.Builder
	listDepth int
	inPre     bool
}

func (c *converter) walk(n dom.Node) {
	switch n.NodeType() {
	case dom.TextNode:
		c.handleText(n)
	case dom.ElementNode:
		el, ok := n.(*dom.Element)
		if ok {
			c.handleElement(el)
		}
	case dom.DocumentNode:
		c.walkChildren(n)
	}
}

func (c *converter) handleText(n dom.Node) {
	text := n.TextContent()
	if c.inPre {
		c.buf.WriteString(text)
		return
	}
	// Collapse whitespace in normal flow
	text = collapseWhitespace(text)
	if text != "" {
		c.buf.WriteString(text)
	}
}

func (c *converter) handleElement(el *dom.Element) {
	tag := el.LocalName()

	switch tag {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		c.handleHeading(el)
	case "p":
		c.handleParagraph(el)
	case "br":
		c.buf.WriteString("\n")
	case "hr":
		c.ensureBlankLine()
		c.buf.WriteString("---\n\n")
	case "strong", "b":
		c.handleInlineWrap(el, "**")
	case "em", "i":
		c.handleInlineWrap(el, "*")
	case "code":
		if !c.inPre {
			c.handleInlineCode(el)
		} else {
			c.walkChildren(el)
		}
	case "pre":
		c.handlePre(el)
	case "a":
		c.handleLink(el)
	case "img":
		c.handleImage(el)
	case "ul":
		c.handleList(el, false)
	case "ol":
		c.handleList(el, true)
	case "li":
		c.walkChildren(el)
	case "blockquote":
		c.handleBlockquote(el)
	case "table":
		c.handleTable(el)
	case "div", "section", "article", "main", "span", "header",
		"footer", "nav", "aside", "figure", "figcaption":
		c.walkChildren(el)
	default:
		c.walkChildren(el)
	}
}

func (c *converter) handleHeading(el *dom.Element) {
	level := int(el.LocalName()[1] - '0')
	c.ensureBlankLine()
	c.buf.WriteString(strings.Repeat("#", level))
	c.buf.WriteString(" ")
	c.walkChildren(el)
	c.buf.WriteString("\n\n")
}

func (c *converter) handleParagraph(el *dom.Element) {
	c.ensureBlankLine()
	c.walkChildren(el)
	c.buf.WriteString("\n\n")
}

func (c *converter) handleInlineWrap(el *dom.Element, mark string) {
	c.buf.WriteString(mark)
	c.walkChildren(el)
	c.buf.WriteString(mark)
}

func (c *converter) handleInlineCode(el *dom.Element) {
	text := el.TextContent()
	if strings.Contains(text, "`") {
		c.buf.WriteString("`` ")
		c.buf.WriteString(text)
		c.buf.WriteString(" ``")
	} else {
		c.buf.WriteString("`")
		c.buf.WriteString(text)
		c.buf.WriteString("`")
	}
}

func (c *converter) handlePre(el *dom.Element) {
	c.ensureBlankLine()
	// Detect language from <code class="language-xxx"> child
	lang := ""
	if child := el.FirstChild(); child != nil {
		if code, ok := child.(*dom.Element); ok && code.LocalName() == "code" {
			cls := code.GetAttribute("class")
			if strings.HasPrefix(cls, "language-") {
				lang = strings.TrimPrefix(cls, "language-")
			}
		}
	}
	c.buf.WriteString("```")
	c.buf.WriteString(lang)
	c.buf.WriteString("\n")
	c.inPre = true
	c.walkChildren(el)
	c.inPre = false
	// Ensure trailing newline before closing fence
	s := c.buf.String()
	if s != "" && s[len(s)-1] != '\n' {
		c.buf.WriteString("\n")
	}
	c.buf.WriteString("```\n\n")
}

func (c *converter) handleLink(el *dom.Element) {
	href := el.GetAttribute("href")
	c.buf.WriteString("[")
	c.walkChildren(el)
	c.buf.WriteString("](")
	c.buf.WriteString(href)
	c.buf.WriteString(")")
}

func (c *converter) handleImage(el *dom.Element) {
	alt := el.GetAttribute("alt")
	src := el.GetAttribute("src")
	c.buf.WriteString("![")
	c.buf.WriteString(alt)
	c.buf.WriteString("](")
	c.buf.WriteString(src)
	c.buf.WriteString(")")
}

func (c *converter) handleList(el *dom.Element, ordered bool) {
	c.ensureBlankLine()
	c.listDepth++
	indent := strings.Repeat("  ", c.listDepth-1)
	idx := 1
	for child := el.FirstChild(); child != nil; child = child.NextSibling() {
		li, ok := child.(*dom.Element)
		if !ok || li.LocalName() != "li" {
			continue
		}
		if ordered {
			fmt.Fprintf(c.buf, "%s%d. ", indent, idx)
			idx++
		} else {
			c.buf.WriteString(indent + "- ")
		}
		c.walkLiChildren(li)
		c.buf.WriteString("\n")
	}
	c.listDepth--
	if c.listDepth == 0 {
		c.buf.WriteString("\n")
	}
}

func (c *converter) walkLiChildren(li *dom.Element) {
	for child := li.FirstChild(); child != nil; child = child.NextSibling() {
		if el, ok := child.(*dom.Element); ok {
			tag := el.LocalName()
			if tag == "ul" || tag == "ol" {
				c.buf.WriteString("\n")
				c.handleList(el, tag == "ol")
				continue
			}
		}
		c.walk(child)
	}
}

func (c *converter) handleBlockquote(el *dom.Element) {
	c.ensureBlankLine()
	// Convert children only (not the blockquote itself) to avoid infinite recursion
	var inner strings.Builder
	ic := &converter{buf: &inner}
	ic.walkChildren(el)
	text := strings.TrimSpace(inner.String())
	for _, line := range strings.Split(text, "\n") {
		c.buf.WriteString("> ")
		c.buf.WriteString(line)
		c.buf.WriteString("\n")
	}
	c.buf.WriteString("\n")
}

func (c *converter) handleTable(el *dom.Element) {
	rows := collectTableRows(el)
	if len(rows) == 0 {
		return
	}

	c.ensureBlankLine()

	// Determine column count
	cols := 0
	for _, row := range rows {
		if len(row) > cols {
			cols = len(row)
		}
	}

	// First row is header
	c.writeTableRow(rows[0], cols)
	// Separator
	c.buf.WriteString("|")
	for i := 0; i < cols; i++ {
		c.buf.WriteString(" --- |")
	}
	c.buf.WriteString("\n")

	// Data rows
	for _, row := range rows[1:] {
		c.writeTableRow(row, cols)
	}
	c.buf.WriteString("\n")
}

func (c *converter) writeTableRow(cells []string, cols int) {
	c.buf.WriteString("|")
	for i := 0; i < cols; i++ {
		c.buf.WriteString(" ")
		if i < len(cells) {
			c.buf.WriteString(cells[i])
		}
		c.buf.WriteString(" |")
	}
	c.buf.WriteString("\n")
}

func collectTableRows(table *dom.Element) [][]string {
	var rows [][]string
	// Walk thead/tbody/tfoot/tr
	for child := table.FirstChild(); child != nil; child = child.NextSibling() {
		el, ok := child.(*dom.Element)
		if !ok {
			continue
		}
		switch el.LocalName() {
		case "thead", "tbody", "tfoot":
			rows = append(rows, collectRowsFromSection(el)...)
		case "tr":
			rows = append(rows, collectCells(el))
		}
	}
	return rows
}

func collectRowsFromSection(section *dom.Element) [][]string {
	var rows [][]string
	for child := section.FirstChild(); child != nil; child = child.NextSibling() {
		if el, ok := child.(*dom.Element); ok && el.LocalName() == "tr" {
			rows = append(rows, collectCells(el))
		}
	}
	return rows
}

func collectCells(tr *dom.Element) []string {
	var cells []string
	for child := tr.FirstChild(); child != nil; child = child.NextSibling() {
		if el, ok := child.(*dom.Element); ok {
			tag := el.LocalName()
			if tag == "td" || tag == "th" {
				cells = append(cells, strings.TrimSpace(el.TextContent()))
			}
		}
	}
	return cells
}

func (c *converter) walkChildren(n dom.Node) {
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		c.walk(child)
	}
}

func (c *converter) ensureBlankLine() {
	s := c.buf.String()
	if s == "" {
		return
	}
	if !strings.HasSuffix(s, "\n\n") {
		if strings.HasSuffix(s, "\n") {
			c.buf.WriteString("\n")
		} else {
			c.buf.WriteString("\n\n")
		}
	}
}

func collapseWhitespace(s string) string {
	var sb strings.Builder
	inSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !inSpace {
				sb.WriteRune(' ')
				inSpace = true
			}
		} else {
			sb.WriteRune(r)
			inSpace = false
		}
	}
	return sb.String()
}
