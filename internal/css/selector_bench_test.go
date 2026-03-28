package css

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nyasuto/uzura/internal/dom"
	htmlparser "github.com/nyasuto/uzura/internal/html"
)

// buildLargeHTML generates an HTML document with many nested elements.
func buildLargeHTML(numItems int) string {
	var sb strings.Builder
	sb.WriteString("<html><body>")
	for i := range numItems {
		cls := "item"
		if i%3 == 0 {
			cls += " featured"
		}
		fmt.Fprintf(&sb, `<div class="container c%d"><div class="%s" id="item-%d">`, i, cls, i)
		fmt.Fprintf(&sb, `<span class="title">Title %d</span>`, i)
		fmt.Fprintf(&sb, `<p class="desc">Description for item %d</p>`, i)
		sb.WriteString(`<a href="/link" class="btn">Click</a>`)
		sb.WriteString("</div></div>")
	}
	sb.WriteString("</body></html>")
	return sb.String()
}

func benchDoc(b *testing.B, numItems int) *dom.Document {
	b.Helper()
	htmlStr := buildLargeHTML(numItems)
	doc, err := htmlparser.Parse(strings.NewReader(htmlStr))
	if err != nil {
		b.Fatal(err)
	}
	doc.SetQueryEngine(NewEngine())
	return doc
}

func BenchmarkQuerySelectorAll_1000(b *testing.B) {
	doc := benchDoc(b, 1000)

	selectors := []string{
		".item",
		".featured",
		"div > .item > span.title",
		"a[href]",
		"#item-500",
		".container .item .desc",
	}

	for _, sel := range selectors {
		b.Run(sel, func(b *testing.B) {
			for b.Loop() {
				_, err := QuerySelectorAll(doc, sel)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkQuerySelector_1000(b *testing.B) {
	doc := benchDoc(b, 1000)

	for b.Loop() {
		_, err := QuerySelector(doc, "#item-999")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompiledSelector_1000(b *testing.B) {
	doc := benchDoc(b, 1000)
	sel, err := Compile(".item")
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		sel.QueryAll(doc)
	}
}
