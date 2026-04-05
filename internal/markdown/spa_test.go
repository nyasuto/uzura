package markdown

import (
	"strings"
	"testing"
)

func TestDetectSPA(t *testing.T) {
	tests := []struct {
		name string
		html string
		want bool
	}{
		{
			name: "empty body",
			html: `<html><head><title>App</title></head><body></body></html>`,
			want: true,
		},
		{
			name: "whitespace only body",
			html: `<html><body>   </body></html>`,
			want: true,
		},
		{
			name: "loading indicator",
			html: `<html><body><div id="root">Loading...</div></body></html>`,
			want: true,
		},
		{
			name: "please wait indicator",
			html: `<html><body><div>Please wait while the app loads</div></body></html>`,
			want: true,
		},
		{
			name: "javascript required message",
			html: `<html><body><noscript>Please enable JavaScript to use this app</noscript></body></html>`,
			want: true,
		},
		{
			name: "react CSR skeleton",
			html: `<html><head><title>React App</title></head><body><div id="root"></div><script src="/static/js/main.js"></script></body></html>`,
			want: true,
		},
		{
			name: "angular CSR skeleton",
			html: `<html><body><app-root>Loading...</app-root><script src="main.js"></script></body></html>`,
			want: true,
		},
		{
			name: "vue CSR skeleton",
			html: `<html><body><div id="app"></div><script src="/js/app.js"></script></body></html>`,
			want: true,
		},
		{
			name: "normal page with content",
			html: `<html><body><h1>Hello World</h1><p>` + strings.Repeat("This is real content. ", 20) + `</p></body></html>`,
			want: false,
		},
		{
			name: "page with short but real content",
			html: `<html><body><h1>404 Not Found</h1><p>The page you are looking for does not exist. Please check the URL and try again. Contact support if the issue persists.</p></body></html>`,
			want: false,
		},
		{
			name: "page with loading in long content",
			html: `<html><body><h1>Guide</h1><p>` + strings.Repeat("Loading data is important for performance. ", 10) + `</p></body></html>`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := mustParse(t, tt.html)
			got := DetectSPA(doc)
			if got != tt.want {
				t.Errorf("DetectSPA() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectSPAFromContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		quality ContentQuality
		want    bool
	}{
		{"good quality", "Lots of real content here", QualityGood, false},
		{"empty failed", "", QualityFailed, true},
		{"loading partial", "Loading...", QualityPartial, true},
		{"please wait partial", "Please wait", QualityPartial, true},
		{"short real content partial", "Short but real text", QualityPartial, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectSPAFromContent(tt.content, tt.quality)
			if got != tt.want {
				t.Errorf("DetectSPAFromContent(%q, %d) = %v, want %v",
					tt.content, tt.quality, got, tt.want)
			}
		})
	}
}

func TestRenderWithFallback_SPADetection(t *testing.T) {
	tests := []struct {
		name         string
		html         string
		wantSPA      bool
		wantContains string
	}{
		{
			name:         "react CSR app",
			html:         `<html><head><title>React App</title></head><body><div id="root"></div></body></html>`,
			wantSPA:      true,
			wantContains: "spa_detected: true",
		},
		{
			name:         "angular loading",
			html:         `<html><head><title>Angular App</title></head><body><app-root>Loading...</app-root></body></html>`,
			wantSPA:      true,
			wantContains: "spa_detected: true",
		},
		{
			name:    "vue empty mount",
			html:    `<html><head><title>Vue App</title></head><body><div id="app"></div></body></html>`,
			wantSPA: true,
		},
		{
			name: "normal article page",
			html: `<html><head><title>Blog Post</title></head><body>
				<article>` + strings.Repeat("<p>This is paragraph content for the blog post. </p>", 10) + `</article>
			</body></html>`,
			wantSPA: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := mustParse(t, tt.html)
			result := RenderWithFallback(doc, "https://example.com/app")

			hasSPA := strings.Contains(result, "spa_detected: true")
			if hasSPA != tt.wantSPA {
				t.Errorf("spa_detected in output = %v, want %v\noutput:\n%s", hasSPA, tt.wantSPA, result)
			}

			if tt.wantContains != "" && !strings.Contains(result, tt.wantContains) {
				t.Errorf("output missing %q\noutput:\n%s", tt.wantContains, result)
			}
		})
	}
}
