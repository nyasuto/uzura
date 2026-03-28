package css

import (
	"strings"
	"testing"

	"github.com/nyasuto/uzura/internal/dom"
	htmlparser "github.com/nyasuto/uzura/internal/html"
)

func parseHTML(t *testing.T, s string) *dom.Document {
	t.Helper()
	doc, err := htmlparser.Parse(strings.NewReader(s))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return doc
}

func TestCompile(t *testing.T) {
	tests := []struct {
		name    string
		sel     string
		wantErr bool
	}{
		{"tag", "div", false},
		{"class", ".foo", false},
		{"id", "#bar", false},
		{"attribute", "[href]", false},
		{"complex", "div.foo > span.bar", false},
		{"invalid", "!!!", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Compile(tt.sel)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compile(%q) error = %v, wantErr %v", tt.sel, err, tt.wantErr)
			}
		})
	}
}

func TestQuerySelectorAll(t *testing.T) {
	doc := parseHTML(t, `<html><body>
		<div id="a" class="foo">
			<span class="bar">hello</span>
			<span class="baz">world</span>
		</div>
		<div id="b" class="foo">
			<p>text</p>
		</div>
	</body></html>`)

	tests := []struct {
		name     string
		sel      string
		wantLen  int
		wantTags []string
	}{
		{"by tag", "span", 2, []string{"SPAN", "SPAN"}},
		{"by class", ".foo", 2, []string{"DIV", "DIV"}},
		{"by id", "#a", 1, []string{"DIV"}},
		{"descendant", "div span", 2, []string{"SPAN", "SPAN"}},
		{"child combinator", "div > p", 1, []string{"P"}},
		{"attribute", "[id]", 2, []string{"DIV", "DIV"}},
		{"compound", "div.foo", 2, []string{"DIV", "DIV"}},
		{"no match", ".nonexistent", 0, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := QuerySelectorAll(doc, tt.sel)
			if err != nil {
				t.Fatalf("QuerySelectorAll error: %v", err)
			}
			if len(results) != tt.wantLen {
				t.Errorf("got %d results, want %d", len(results), tt.wantLen)
			}
			for i, r := range results {
				if i < len(tt.wantTags) && r.TagName() != tt.wantTags[i] {
					t.Errorf("result[%d].TagName() = %q, want %q", i, r.TagName(), tt.wantTags[i])
				}
			}
		})
	}
}

func TestQuerySelector(t *testing.T) {
	doc := parseHTML(t, `<html><body>
		<div id="first" class="item">one</div>
		<div id="second" class="item">two</div>
	</body></html>`)

	tests := []struct {
		name   string
		sel    string
		wantID string
	}{
		{"first match", ".item", "first"},
		{"by id", "#second", "second"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := QuerySelector(doc, tt.sel)
			if err != nil {
				t.Fatalf("QuerySelector error: %v", err)
			}
			if result == nil {
				t.Fatal("expected a result, got nil")
			}
			if result.Id() != tt.wantID {
				t.Errorf("got id=%q, want %q", result.Id(), tt.wantID)
			}
		})
	}

	// No match
	result, err := QuerySelector(doc, ".nonexistent")
	if err != nil {
		t.Fatalf("QuerySelector error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestQueryFromElement(t *testing.T) {
	doc := parseHTML(t, `<html><body>
		<div id="container">
			<span class="inner">inside</span>
		</div>
		<span class="outer">outside</span>
	</body></html>`)

	container := doc.GetElementById("container")
	if container == nil {
		t.Fatal("container not found")
	}

	// QuerySelectorAll from element should only find descendants
	results, err := QuerySelectorAll(container, "span")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results from container, want 1", len(results))
	}
	if len(results) > 0 && results[0].ClassName() != "inner" {
		t.Errorf("got class=%q, want %q", results[0].ClassName(), "inner")
	}
}
