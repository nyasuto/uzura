package dom

import "testing"

func TestSerializeElement(t *testing.T) {
	tests := []struct {
		name string
		build func() Node
		want  string
	}{
		{
			name: "simple element",
			build: func() Node {
				e := NewElement("div")
				e.AppendChild(NewText("hello"))
				return e
			},
			want: "<div>hello</div>",
		},
		{
			name: "element with attributes",
			build: func() Node {
				e := NewElement("a")
				e.SetAttribute("href", "https://example.com")
				e.SetAttribute("class", "link")
				e.AppendChild(NewText("click"))
				return e
			},
			want: `<a href="https://example.com" class="link">click</a>`,
		},
		{
			name: "void element",
			build: func() Node {
				e := NewElement("br")
				return e
			},
			want: "<br>",
		},
		{
			name: "void element img with attrs",
			build: func() Node {
				e := NewElement("img")
				e.SetAttribute("src", "pic.png")
				e.SetAttribute("alt", "A picture")
				return e
			},
			want: `<img src="pic.png" alt="A picture">`,
		},
		{
			name: "nested elements",
			build: func() Node {
				div := NewElement("div")
				p := NewElement("p")
				p.AppendChild(NewText("paragraph"))
				div.AppendChild(p)
				return div
			},
			want: "<div><p>paragraph</p></div>",
		},
		{
			name: "comment node",
			build: func() Node {
				div := NewElement("div")
				div.AppendChild(NewComment(" test "))
				return div
			},
			want: "<div><!-- test --></div>",
		},
		{
			name: "text escaping",
			build: func() Node {
				e := NewElement("p")
				e.AppendChild(NewText("a < b & c > d"))
				return e
			},
			want: "<p>a &lt; b &amp; c &gt; d</p>",
		},
		{
			name: "attribute escaping",
			build: func() Node {
				e := NewElement("div")
				e.SetAttribute("data-val", `he said "hello"`)
				return e
			},
			want: `<div data-val="he said &quot;hello&quot;"></div>`,
		},
		{
			name: "raw text element script",
			build: func() Node {
				e := NewElement("script")
				e.AppendChild(NewText("var x = 1 < 2 && true;"))
				return e
			},
			want: "<script>var x = 1 < 2 && true;</script>",
		},
		{
			name: "raw text element style",
			build: func() Node {
				e := NewElement("style")
				e.AppendChild(NewText("body { color: red; }"))
				return e
			},
			want: "<style>body { color: red; }</style>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := tt.build()
			got := Serialize(node)
			if got != tt.want {
				t.Errorf("Serialize() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInnerHTML(t *testing.T) {
	div := NewElement("div")
	div.AppendChild(NewText("hello "))
	span := NewElement("span")
	span.AppendChild(NewText("world"))
	div.AppendChild(span)

	got := InnerHTML(div)
	want := "hello <span>world</span>"
	if got != want {
		t.Errorf("InnerHTML() = %q, want %q", got, want)
	}
}

func TestOuterHTML(t *testing.T) {
	div := NewElement("div")
	div.SetAttribute("id", "test")
	div.AppendChild(NewText("content"))

	got := OuterHTML(div)
	want := `<div id="test">content</div>`
	if got != want {
		t.Errorf("OuterHTML() = %q, want %q", got, want)
	}
}
