package markdown

import (
	"strings"
	"testing"

	htmlparser "github.com/nyasuto/uzura/internal/html"
)

func TestExtractNoscriptContent(t *testing.T) {
	tests := []struct {
		name      string
		html      string
		wantCount int
		wantTexts []string
	}{
		{
			name:      "no noscript elements",
			html:      `<html><body><p>Hello</p></body></html>`,
			wantCount: 0,
		},
		{
			name:      "single noscript with useful content",
			html:      `<html><body><noscript><div><p>Enable JavaScript for a better experience.</p><p>Here is the main content without JS.</p></div></noscript></body></html>`,
			wantCount: 1,
			wantTexts: []string{"main content without JS"},
		},
		{
			name: "multiple noscript elements",
			html: `<html><body>
				<noscript><p>First noscript block with content.</p></noscript>
				<noscript><p>Second noscript block with more content here.</p></noscript>
			</body></html>`,
			wantCount: 2,
		},
		{
			name:      "noscript with only short message",
			html:      `<html><body><noscript>Enable JS</noscript></body></html>`,
			wantCount: 1,
			wantTexts: []string{"Enable JS"},
		},
		{
			name:      "noscript in head is skipped",
			html:      `<html><head><noscript><meta http-equiv="refresh" content="0;url=nojs.html"></noscript></head><body><noscript><p>Useful body noscript content here.</p></noscript></body></html>`,
			wantCount: 1,
			wantTexts: []string{"Useful body noscript"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := htmlparser.Parse(strings.NewReader(tt.html))
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			contents := ExtractNoscriptContent(doc)
			if len(contents) != tt.wantCount {
				t.Errorf("got %d noscript contents, want %d", len(contents), tt.wantCount)
			}

			for _, want := range tt.wantTexts {
				found := false
				for _, c := range contents {
					if strings.Contains(c, want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected text %q not found in noscript contents: %v", want, contents)
				}
			}
		})
	}
}

func TestPickBestNoscript(t *testing.T) {
	tests := []struct {
		name     string
		contents []string
		wantBest string
	}{
		{
			name:     "empty list",
			contents: nil,
			wantBest: "",
		},
		{
			name:     "single short entry",
			contents: []string{"Enable JS"},
			wantBest: "",
		},
		{
			name: "picks longest useful content",
			contents: []string{
				"Enable JavaScript",
				"This is the main article content that was placed inside noscript for users without JavaScript support.",
			},
			wantBest: "main article content",
		},
		{
			name: "filters out non-useful content",
			contents: []string{
				"Please enable JavaScript to view this page.",
				strings.Repeat("Real content paragraph. ", 10),
			},
			wantBest: "Real content paragraph",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PickBestNoscript(tt.contents)
			if tt.wantBest == "" {
				if got != "" {
					t.Errorf("expected empty, got %q", got)
				}
				return
			}
			if !strings.Contains(got, tt.wantBest) {
				t.Errorf("expected result to contain %q, got %q", tt.wantBest, got)
			}
		})
	}
}

func TestNoscriptPriorityDecision(t *testing.T) {
	tests := []struct {
		name            string
		normalQuality   ContentQuality
		noscriptContent string
		wantUseNoscript bool
	}{
		{
			name:            "good normal content — no noscript needed",
			normalQuality:   QualityGood,
			noscriptContent: strings.Repeat("Noscript content. ", 20),
			wantUseNoscript: false,
		},
		{
			name:            "failed normal, good noscript — use noscript",
			normalQuality:   QualityFailed,
			noscriptContent: strings.Repeat("Noscript article content. ", 20),
			wantUseNoscript: true,
		},
		{
			name:            "partial normal, good noscript — use noscript",
			normalQuality:   QualityPartial,
			noscriptContent: strings.Repeat("Noscript article content. ", 20),
			wantUseNoscript: true,
		},
		{
			name:            "failed normal, empty noscript — no noscript",
			normalQuality:   QualityFailed,
			noscriptContent: "",
			wantUseNoscript: false,
		},
		{
			name:            "partial normal, short noscript — no noscript",
			normalQuality:   QualityPartial,
			noscriptContent: "Enable JS",
			wantUseNoscript: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldUseNoscript(tt.normalQuality, tt.noscriptContent)
			if got != tt.wantUseNoscript {
				t.Errorf("ShouldUseNoscript(%d, %q) = %v, want %v",
					tt.normalQuality, truncate(tt.noscriptContent, 40), got, tt.wantUseNoscript)
			}
		})
	}
}

func TestRenderWithFallback_NoscriptFallback(t *testing.T) {
	// A page where normal content is just "Loading..." but noscript has real content
	html := `<html><head><title>SPA Page</title></head><body>
		<div id="app">Loading...</div>
		<noscript>
			<h1>Welcome to our site</h1>
			<p>This is the noscript version of our single page application with substantial content for users who do not have JavaScript enabled in their browsers.</p>
		</noscript>
	</body></html>`

	doc := mustParse(t, html)
	result := RenderWithFallback(doc, "https://example.com/spa")

	if !strings.Contains(result, "Welcome to our site") {
		t.Error("expected noscript content in output")
	}
	if !strings.Contains(result, "noscript version") {
		t.Error("expected noscript paragraph content in output")
	}
}

func TestRenderWithFallback_NoscriptNotUsedWhenGoodContent(t *testing.T) {
	// A well-structured page where readability succeeds — noscript should be ignored
	html := `<html><head><title>Good Article</title></head><body>
		<article>` + strings.Repeat("<p>This is a great paragraph of real article content. </p>\n", 10) + `</article>
		<noscript><p>You need JavaScript to view this site.</p></noscript>
	</body></html>`

	doc := mustParse(t, html)
	result := RenderWithFallback(doc, "https://example.com/article")

	if strings.Contains(result, "You need JavaScript") {
		t.Error("noscript content should NOT appear when readability succeeds")
	}
	if !strings.Contains(result, "real article content") {
		t.Error("expected real article content in output")
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
