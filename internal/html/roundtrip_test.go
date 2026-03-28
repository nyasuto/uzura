package html

import (
	"strings"
	"testing"

	"github.com/nyasuto/uzura/internal/dom"
)

func TestRoundTrip(t *testing.T) {
	input := `<html><head><title>Test</title></head><body><div id="main"><p>Hello</p></div></body></html>`

	// Parse
	doc1, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("first parse error: %v", err)
	}

	// Serialize
	html1 := dom.Serialize(doc1)

	// Parse again
	doc2, err := Parse(strings.NewReader(html1))
	if err != nil {
		t.Fatalf("second parse error: %v", err)
	}

	// Serialize again
	html2 := dom.Serialize(doc2)

	// The two serializations should be identical
	if html1 != html2 {
		t.Errorf("roundtrip mismatch:\n  first:  %q\n  second: %q", html1, html2)
	}
}
