package markdown

import (
	"strings"
	"testing"

	"github.com/nyasuto/uzura/internal/dom"
	htmlparser "github.com/nyasuto/uzura/internal/html"
)

func parseHTML(t *testing.T, input string) *dom.Document {
	t.Helper()
	doc, err := htmlparser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return doc
}

func TestConvert_Headings(t *testing.T) {
	tests := []struct {
		html string
		want string
	}{
		{`<h1>Title</h1>`, "# Title\n"},
		{`<h2>Sub</h2>`, "## Sub\n"},
		{`<h3>H3</h3>`, "### H3\n"},
		{`<h6>H6</h6>`, "###### H6\n"},
	}
	for _, tt := range tests {
		doc := parseHTML(t, tt.html)
		got := Convert(doc)
		if got != tt.want {
			t.Errorf("Convert(%q) = %q, want %q", tt.html, got, tt.want)
		}
	}
}

func TestConvert_Paragraph(t *testing.T) {
	doc := parseHTML(t, `<p>Hello world</p><p>Second paragraph</p>`)
	got := Convert(doc)
	if !strings.Contains(got, "Hello world") {
		t.Errorf("missing first paragraph: %q", got)
	}
	if !strings.Contains(got, "Second paragraph") {
		t.Errorf("missing second paragraph: %q", got)
	}
	// Paragraphs should be separated by blank line
	if !strings.Contains(got, "\n\n") {
		t.Errorf("paragraphs should be separated by blank line: %q", got)
	}
}

func TestConvert_Emphasis(t *testing.T) {
	tests := []struct {
		html string
		want string
	}{
		{`<p><strong>bold</strong></p>`, "**bold**"},
		{`<p><em>italic</em></p>`, "*italic*"},
		{`<p><b>bold</b></p>`, "**bold**"},
		{`<p><i>italic</i></p>`, "*italic*"},
		{`<p><strong><em>both</em></strong></p>`, "***both***"},
	}
	for _, tt := range tests {
		doc := parseHTML(t, tt.html)
		got := Convert(doc)
		if !strings.Contains(got, tt.want) {
			t.Errorf("Convert(%q) = %q, want to contain %q", tt.html, got, tt.want)
		}
	}
}

func TestConvert_Link(t *testing.T) {
	doc := parseHTML(t, `<p><a href="https://example.com">Example</a></p>`)
	got := Convert(doc)
	want := "[Example](https://example.com)"
	if !strings.Contains(got, want) {
		t.Errorf("got %q, want to contain %q", got, want)
	}
}

func TestConvert_Image(t *testing.T) {
	doc := parseHTML(t, `<img src="pic.jpg" alt="A picture">`)
	got := Convert(doc)
	want := "![A picture](pic.jpg)"
	if !strings.Contains(got, want) {
		t.Errorf("got %q, want to contain %q", got, want)
	}
}

func TestConvert_InlineCode(t *testing.T) {
	doc := parseHTML(t, `<p>Use <code>fmt.Println</code> to print</p>`)
	got := Convert(doc)
	if !strings.Contains(got, "`fmt.Println`") {
		t.Errorf("got %q, want inline code", got)
	}
}

func TestConvert_CodeBlock(t *testing.T) {
	doc := parseHTML(t, `<pre><code class="language-go">func main() {}</code></pre>`)
	got := Convert(doc)
	if !strings.Contains(got, "```go\n") {
		t.Errorf("missing language fence: %q", got)
	}
	if !strings.Contains(got, "func main() {}") {
		t.Errorf("missing code content: %q", got)
	}
	if !strings.Contains(got, "\n```") {
		t.Errorf("missing closing fence: %q", got)
	}
}

func TestConvert_UnorderedList(t *testing.T) {
	doc := parseHTML(t, `<ul><li>A</li><li>B</li><li>C</li></ul>`)
	got := Convert(doc)
	if !strings.Contains(got, "- A\n") {
		t.Errorf("missing list item A: %q", got)
	}
	if !strings.Contains(got, "- B\n") {
		t.Errorf("missing list item B: %q", got)
	}
}

func TestConvert_OrderedList(t *testing.T) {
	doc := parseHTML(t, `<ol><li>First</li><li>Second</li></ol>`)
	got := Convert(doc)
	if !strings.Contains(got, "1. First\n") {
		t.Errorf("missing ordered item 1: %q", got)
	}
	if !strings.Contains(got, "2. Second\n") {
		t.Errorf("missing ordered item 2: %q", got)
	}
}

func TestConvert_NestedList(t *testing.T) {
	html := `<ul><li>A<ul><li>A1</li><li>A2</li></ul></li><li>B</li></ul>`
	doc := parseHTML(t, html)
	got := Convert(doc)
	if !strings.Contains(got, "- A\n") {
		t.Errorf("missing parent item: %q", got)
	}
	if !strings.Contains(got, "  - A1\n") {
		t.Errorf("missing nested item A1: %q", got)
	}
}

func TestConvert_Blockquote(t *testing.T) {
	doc := parseHTML(t, `<blockquote><p>Quoted text</p></blockquote>`)
	got := Convert(doc)
	if !strings.Contains(got, "> Quoted text") {
		t.Errorf("missing blockquote: %q", got)
	}
}

func TestConvert_Table(t *testing.T) {
	html := `<table>
		<thead><tr><th>Name</th><th>Age</th></tr></thead>
		<tbody><tr><td>Alice</td><td>30</td></tr><tr><td>Bob</td><td>25</td></tr></tbody>
	</table>`
	doc := parseHTML(t, html)
	got := Convert(doc)
	if !strings.Contains(got, "| Name | Age |") {
		t.Errorf("missing header row: %q", got)
	}
	if !strings.Contains(got, "| --- | --- |") {
		t.Errorf("missing separator: %q", got)
	}
	if !strings.Contains(got, "| Alice | 30 |") {
		t.Errorf("missing data row: %q", got)
	}
}

func TestConvert_HR(t *testing.T) {
	doc := parseHTML(t, `<p>Above</p><hr><p>Below</p>`)
	got := Convert(doc)
	if !strings.Contains(got, "---") {
		t.Errorf("missing hr: %q", got)
	}
}

func TestConvert_EmptyElements(t *testing.T) {
	doc := parseHTML(t, `<p></p><div></div>`)
	got := Convert(doc)
	// Should not crash and produce minimal output
	if strings.Contains(got, "nil") {
		t.Errorf("unexpected nil in output: %q", got)
	}
}

func TestConvert_WhitespaceCollapse(t *testing.T) {
	doc := parseHTML(t, `<p>  hello    world  </p>`)
	got := Convert(doc)
	// Whitespace should be collapsed
	if strings.Contains(got, "  hello") {
		t.Errorf("whitespace not collapsed: %q", got)
	}
}

func TestConvert_DataURISkipped(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string // should NOT appear
	}{
		{
			name: "svg data URI",
			html: `<p><img src="data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciLz4=" alt="icon"></p>`,
			want: "data:image/svg+xml",
		},
		{
			name: "png data URI",
			html: `<p><img src="data:image/png;base64,iVBORw0KGgo=" alt="pixel"></p>`,
			want: "data:image/png",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := parseHTML(t, tt.html)
			got := Convert(doc)
			if strings.Contains(got, tt.want) {
				t.Errorf("output should not contain data URI %q: %q", tt.want, got)
			}
		})
	}
}

func TestConvert_NormalImageKept(t *testing.T) {
	doc := parseHTML(t, `<img src="https://example.com/pic.jpg" alt="photo">`)
	got := Convert(doc)
	if !strings.Contains(got, "![photo](https://example.com/pic.jpg)") {
		t.Errorf("normal image should be kept: %q", got)
	}
}

func TestConvert_SVGElementSkipped(t *testing.T) {
	doc := parseHTML(t, `<p>Before</p><svg xmlns="http://www.w3.org/2000/svg"><circle cx="50" cy="50" r="40"/></svg><p>After</p>`)
	got := Convert(doc)
	if strings.Contains(got, "circle") {
		t.Errorf("SVG element content should be skipped: %q", got)
	}
	if !strings.Contains(got, "Before") || !strings.Contains(got, "After") {
		t.Errorf("surrounding content should be preserved: %q", got)
	}
}

func TestConvert_ExcessiveBlankLinesNormalized(t *testing.T) {
	// Create HTML that would generate many blank lines
	html := `<p>First</p><p></p><p></p><p></p><p>Second</p>`
	doc := parseHTML(t, html)
	got := Convert(doc)
	// Should never have 3+ consecutive newlines
	if strings.Contains(got, "\n\n\n") {
		t.Errorf("excessive blank lines should be normalized: %q", got)
	}
	if !strings.Contains(got, "First") || !strings.Contains(got, "Second") {
		t.Errorf("content should be preserved: %q", got)
	}
}
