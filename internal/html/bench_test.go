package html

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkParse100KB(b *testing.B) {
	// Generate ~100KB of HTML
	var sb strings.Builder
	sb.WriteString("<html><head><title>Bench</title></head><body>")
	for i := 0; sb.Len() < 100_000; i++ {
		sb.WriteString(fmt.Sprintf(`<div class="item" id="item-%d"><p>Paragraph %d with <a href="/link/%d">a link</a> and some text content.</p></div>`, i, i, i))
	}
	sb.WriteString("</body></html>")
	html := sb.String()

	b.ResetTimer()
	b.SetBytes(int64(len(html)))

	for b.Loop() {
		_, err := Parse(strings.NewReader(html))
		if err != nil {
			b.Fatal(err)
		}
	}
}
