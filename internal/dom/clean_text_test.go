package dom

import "testing"

func TestCleanTextContent(t *testing.T) {
	tests := []struct {
		name string
		html func() *Element // build DOM tree
		want string
	}{
		{
			name: "plain text only",
			html: func() *Element {
				doc := NewDocument()
				div := doc.CreateElement("div")
				div.AppendChild(doc.CreateTextNode("hello world"))
				return div
			},
			want: "hello world",
		},
		{
			name: "skips script content",
			html: func() *Element {
				doc := NewDocument()
				div := doc.CreateElement("div")
				div.AppendChild(doc.CreateTextNode("before "))
				script := doc.CreateElement("script")
				script.AppendChild(doc.CreateTextNode("var x = 1;"))
				div.AppendChild(script)
				div.AppendChild(doc.CreateTextNode(" after"))
				return div
			},
			want: "before  after",
		},
		{
			name: "skips style content",
			html: func() *Element {
				doc := NewDocument()
				div := doc.CreateElement("div")
				div.AppendChild(doc.CreateTextNode("visible "))
				style := doc.CreateElement("style")
				style.AppendChild(doc.CreateTextNode("body { color: red; }"))
				div.AppendChild(style)
				div.AppendChild(doc.CreateTextNode(" text"))
				return div
			},
			want: "visible  text",
		},
		{
			name: "skips noscript content",
			html: func() *Element {
				doc := NewDocument()
				div := doc.CreateElement("div")
				div.AppendChild(doc.CreateTextNode("main "))
				noscript := doc.CreateElement("noscript")
				noscript.AppendChild(doc.CreateTextNode("enable JS"))
				div.AppendChild(noscript)
				div.AppendChild(doc.CreateTextNode(" content"))
				return div
			},
			want: "main  content",
		},
		{
			name: "nested script inside div",
			html: func() *Element {
				doc := NewDocument()
				div := doc.CreateElement("div")
				inner := doc.CreateElement("p")
				inner.AppendChild(doc.CreateTextNode("para "))
				script := doc.CreateElement("script")
				script.AppendChild(doc.CreateTextNode("alert(1)"))
				inner.AppendChild(script)
				inner.AppendChild(doc.CreateTextNode(" text"))
				div.AppendChild(inner)
				return div
			},
			want: "para  text",
		},
		{
			name: "multiple scripts and styles",
			html: func() *Element {
				doc := NewDocument()
				div := doc.CreateElement("div")
				s1 := doc.CreateElement("script")
				s1.AppendChild(doc.CreateTextNode("js1"))
				div.AppendChild(s1)
				div.AppendChild(doc.CreateTextNode("real"))
				s2 := doc.CreateElement("style")
				s2.AppendChild(doc.CreateTextNode("css"))
				div.AppendChild(s2)
				s3 := doc.CreateElement("script")
				s3.AppendChild(doc.CreateTextNode("js2"))
				div.AppendChild(s3)
				return div
			},
			want: "real",
		},
		{
			name: "deeply nested structure",
			html: func() *Element {
				doc := NewDocument()
				div := doc.CreateElement("div")
				body := doc.CreateElement("body")
				p := doc.CreateElement("p")
				p.AppendChild(doc.CreateTextNode("hello"))
				body.AppendChild(p)
				script := doc.CreateElement("script")
				script.AppendChild(doc.CreateTextNode("console.log('x')"))
				body.AppendChild(script)
				div.AppendChild(body)
				return div
			},
			want: "hello",
		},
		{
			name: "empty element",
			html: func() *Element {
				doc := NewDocument()
				return doc.CreateElement("div")
			},
			want: "",
		},
		{
			name: "skips hidden attribute",
			html: func() *Element {
				doc := NewDocument()
				div := doc.CreateElement("div")
				div.AppendChild(doc.CreateTextNode("visible"))
				hidden := doc.CreateElement("span")
				hidden.SetAttribute("hidden", "")
				hidden.AppendChild(doc.CreateTextNode("hidden text"))
				div.AppendChild(hidden)
				return div
			},
			want: "visible",
		},
		{
			name: "skips aria-hidden=true",
			html: func() *Element {
				doc := NewDocument()
				div := doc.CreateElement("div")
				div.AppendChild(doc.CreateTextNode("see me"))
				aria := doc.CreateElement("span")
				aria.SetAttribute("aria-hidden", "true")
				aria.AppendChild(doc.CreateTextNode("icon"))
				div.AppendChild(aria)
				return div
			},
			want: "see me",
		},
		{
			name: "keeps aria-hidden=false",
			html: func() *Element {
				doc := NewDocument()
				div := doc.CreateElement("div")
				aria := doc.CreateElement("span")
				aria.SetAttribute("aria-hidden", "false")
				aria.AppendChild(doc.CreateTextNode("shown"))
				div.AppendChild(aria)
				return div
			},
			want: "shown",
		},
		{
			name: "skips display:none",
			html: func() *Element {
				doc := NewDocument()
				div := doc.CreateElement("div")
				div.AppendChild(doc.CreateTextNode("visible"))
				none := doc.CreateElement("div")
				none.SetAttribute("style", "display:none")
				none.AppendChild(doc.CreateTextNode("invisible"))
				div.AppendChild(none)
				return div
			},
			want: "visible",
		},
		{
			name: "skips display: none with space",
			html: func() *Element {
				doc := NewDocument()
				div := doc.CreateElement("div")
				div.AppendChild(doc.CreateTextNode("ok"))
				none := doc.CreateElement("span")
				none.SetAttribute("style", "color: red; Display: None; font-size: 12px")
				none.AppendChild(doc.CreateTextNode("hidden"))
				div.AppendChild(none)
				return div
			},
			want: "ok",
		},
		{
			name: "skips template element",
			html: func() *Element {
				doc := NewDocument()
				div := doc.CreateElement("div")
				div.AppendChild(doc.CreateTextNode("content"))
				tmpl := doc.CreateElement("template")
				tmpl.AppendChild(doc.CreateTextNode("template text"))
				div.AppendChild(tmpl)
				return div
			},
			want: "content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			el := tt.html()
			got := CleanTextContent(el)
			if got != tt.want {
				t.Errorf("CleanTextContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func BenchmarkCleanTextContent(b *testing.B) {
	doc := NewDocument()
	root := doc.CreateElement("div")
	// Build a tree with mixed content
	for i := 0; i < 100; i++ {
		p := doc.CreateElement("p")
		p.AppendChild(doc.CreateTextNode("paragraph text content "))
		root.AppendChild(p)
		if i%5 == 0 {
			script := doc.CreateElement("script")
			script.AppendChild(doc.CreateTextNode("var x = " + string(rune('0'+i%10)) + ";"))
			root.AppendChild(script)
		}
		if i%7 == 0 {
			style := doc.CreateElement("style")
			style.AppendChild(doc.CreateTextNode(".cls { color: red; }"))
			root.AppendChild(style)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CleanTextContent(root)
	}
}
