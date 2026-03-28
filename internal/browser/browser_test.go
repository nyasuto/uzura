package browser_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nyasuto/uzura/internal/browser"
	"github.com/nyasuto/uzura/internal/page"
)

func TestNewBrowser(t *testing.T) {
	b := browser.New()
	defer b.Close()

	if b == nil {
		t.Fatal("New() returned nil")
	}
}

func TestDefaultContext(t *testing.T) {
	b := browser.New()
	defer b.Close()

	ctx := b.DefaultContext()
	if ctx == nil {
		t.Fatal("DefaultContext() returned nil")
	}

	// Calling DefaultContext again returns the same instance.
	ctx2 := b.DefaultContext()
	if ctx != ctx2 {
		t.Error("DefaultContext() should return the same instance")
	}
}

func TestNewContext(t *testing.T) {
	b := browser.New()
	defer b.Close()

	ctx1 := b.NewContext()
	ctx2 := b.NewContext()

	if ctx1 == ctx2 {
		t.Error("NewContext() should return distinct instances")
	}

	contexts := b.Contexts()
	// Should include default + ctx1 + ctx2 = 3
	if len(contexts) != 3 {
		t.Errorf("Contexts() = %d, want 3", len(contexts))
	}
}

func TestContextClose(t *testing.T) {
	b := browser.New()
	defer b.Close()

	ctx := b.NewContext()
	before := len(b.Contexts())

	if err := ctx.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	after := len(b.Contexts())
	if after != before-1 {
		t.Errorf("after Close(): Contexts() = %d, want %d", after, before-1)
	}
}

func TestCloseDefaultContextError(t *testing.T) {
	b := browser.New()
	defer b.Close()

	err := b.DefaultContext().Close()
	if err == nil {
		t.Error("closing default context should return an error")
	}
}

func TestBrowserClose(t *testing.T) {
	b := browser.New()
	_ = b.NewContext()
	_ = b.NewContext()

	if err := b.Close(); err != nil {
		t.Fatalf("Browser.Close() error: %v", err)
	}

	if len(b.Contexts()) != 0 {
		t.Error("after Browser.Close(), Contexts() should be empty")
	}
}

func TestCookieIsolation(t *testing.T) {
	// Set up two servers: one sets a cookie, one reads it.
	setCookie := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:  "session",
			Value: "abc123",
			Path:  "/",
		})
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body>cookie set</body></html>")
	}))
	defer setCookie.Close()

	readCookie := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session")
		w.Header().Set("Content-Type", "text/html")
		if err != nil {
			fmt.Fprint(w, "<html><body>no cookie</body></html>")
			return
		}
		fmt.Fprintf(w, "<html><body>%s</body></html>", c.Value)
	}))
	defer readCookie.Close()

	b := browser.New()
	defer b.Close()

	ctx1 := b.NewContext()
	ctx2 := b.NewContext()

	bgCtx := context.Background()

	// ctx1: navigate to set cookie
	p1, err := ctx1.NewPage()
	if err != nil {
		t.Fatalf("ctx1.NewPage: %v", err)
	}
	err = p1.Navigate(bgCtx, setCookie.URL)
	if err != nil {
		t.Fatalf("ctx1 navigate set: %v", err)
	}

	// ctx1: navigate to read cookie — should see the cookie
	err = p1.Navigate(bgCtx, readCookie.URL)
	if err != nil {
		t.Fatalf("ctx1 navigate read: %v", err)
	}
	body1 := p1.Document().Body().TextContent()
	if body1 != "abc123" {
		t.Errorf("ctx1 cookie = %q, want %q", body1, "abc123")
	}

	// ctx2: navigate to read cookie — should NOT see ctx1's cookie
	p2, err := ctx2.NewPage()
	if err != nil {
		t.Fatalf("ctx2.NewPage: %v", err)
	}
	err = p2.Navigate(bgCtx, readCookie.URL)
	if err != nil {
		t.Fatalf("ctx2 navigate read: %v", err)
	}
	body2 := p2.Document().Body().TextContent()
	if body2 != "no cookie" {
		t.Errorf("ctx2 cookie = %q, want %q (cookies should be isolated)", body2, "no cookie")
	}
}

func TestContextNewPage(t *testing.T) {
	b := browser.New()
	defer b.Close()

	ctx := b.DefaultContext()
	p, err := ctx.NewPage()
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	if p == nil {
		t.Fatal("NewPage returned nil")
	}

	pages := ctx.Pages()
	if len(pages) != 1 {
		t.Errorf("Pages() = %d, want 1", len(pages))
	}
}

func TestPageClose(t *testing.T) {
	b := browser.New()
	defer b.Close()

	ctx := b.DefaultContext()
	p, err := ctx.NewPage()
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}

	if p.IsClosed() {
		t.Error("new page should not be closed")
	}

	if err := p.Close(); err != nil {
		t.Fatalf("Page.Close: %v", err)
	}

	if !p.IsClosed() {
		t.Error("after Close(), IsClosed should be true")
	}

	// Close is idempotent.
	if err := p.Close(); err != nil {
		t.Errorf("second Close() should not error: %v", err)
	}

	// Page should be removed from context.
	if len(ctx.Pages()) != 0 {
		t.Errorf("Pages() = %d, want 0 after close", len(ctx.Pages()))
	}
}

func TestPageCloseReleasesResources(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body><p>hello</p></body></html>")
	}))
	defer ts.Close()

	b := browser.New()
	defer b.Close()

	p, err := b.DefaultContext().NewPage()
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	if err := p.Navigate(context.Background(), ts.URL); err != nil {
		t.Fatalf("Navigate: %v", err)
	}

	// Verify page has content.
	if p.Document() == nil {
		t.Fatal("expected document before close")
	}

	if err := p.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// After close, document and URL should be nil/"".
	if p.Document() != nil {
		t.Error("Document should be nil after close")
	}
	if p.URL() != "" {
		t.Error("URL should be empty after close")
	}
}

func TestPageNavigateAfterClose(t *testing.T) {
	b := browser.New()
	defer b.Close()

	p, err := b.DefaultContext().NewPage()
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	_ = p.Close()

	err = p.Navigate(context.Background(), "http://example.com")
	if err == nil {
		t.Fatal("Navigate on closed page should error")
	}
}

func TestMultiplePages(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<html><body>%s</body></html>", r.URL.Path)
	}))
	defer ts.Close()

	b := browser.New()
	defer b.Close()

	ctx := b.DefaultContext()

	p1, _ := ctx.NewPage()
	p2, _ := ctx.NewPage()
	p3, _ := ctx.NewPage()

	if len(ctx.Pages()) != 3 {
		t.Fatalf("Pages() = %d, want 3", len(ctx.Pages()))
	}

	// Each page has its own ID.
	ids := map[string]bool{p1.ID(): true, p2.ID(): true, p3.ID(): true}
	if len(ids) != 3 {
		t.Error("page IDs should be unique")
	}

	// Close middle page.
	_ = p2.Close()
	pages := ctx.Pages()
	if len(pages) != 2 {
		t.Fatalf("Pages() = %d, want 2 after closing one", len(pages))
	}

	// Remaining pages should still work.
	bgCtx := context.Background()
	if err := p1.Navigate(bgCtx, ts.URL+"/a"); err != nil {
		t.Fatalf("p1 Navigate: %v", err)
	}
	if err := p3.Navigate(bgCtx, ts.URL+"/c"); err != nil {
		t.Fatalf("p3 Navigate: %v", err)
	}

	if p1.Document().Body().TextContent() != "/a" {
		t.Errorf("p1 body = %q, want /a", p1.Document().Body().TextContent())
	}
	if p3.Document().Body().TextContent() != "/c" {
		t.Errorf("p3 body = %q, want /c", p3.Document().Body().TextContent())
	}
}

func TestPageResourceIndependence(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<html><body><div id='page'>%s</div></body></html>", r.URL.Path)
	}))
	defer ts.Close()

	b := browser.New()
	defer b.Close()

	ctx := b.DefaultContext()
	p1, _ := ctx.NewPage()
	p2, _ := ctx.NewPage()

	bgCtx := context.Background()
	_ = p1.Navigate(bgCtx, ts.URL+"/first")
	_ = p2.Navigate(bgCtx, ts.URL+"/second")

	// Each page has independent DOM.
	t1 := p1.Document().Body().TextContent()
	t2 := p2.Document().Body().TextContent()
	if t1 == t2 {
		t.Errorf("pages should have independent DOM: p1=%q p2=%q", t1, t2)
	}

	// Closing p1 does not affect p2.
	_ = p1.Close()
	if p2.IsClosed() {
		t.Error("closing p1 should not close p2")
	}
	if p2.Document() == nil {
		t.Error("p2 document should still exist after p1 close")
	}
}

func TestContextCloseClosesPages(t *testing.T) {
	b := browser.New()
	defer b.Close()

	ctx := b.NewContext()
	p1, _ := ctx.NewPage()
	p2, _ := ctx.NewPage()

	_ = ctx.Close()

	if !p1.IsClosed() {
		t.Error("p1 should be closed after context close")
	}
	if !p2.IsClosed() {
		t.Error("p2 should be closed after context close")
	}
}

func TestConcurrentPageNavigation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<html><body>%s</body></html>", r.URL.Path)
	}))
	defer ts.Close()

	b := browser.New()
	defer b.Close()

	ctx := b.DefaultContext()
	const n = 5
	pages := make([]*page.Page, n)
	for i := range pages {
		p, err := ctx.NewPage()
		if err != nil {
			t.Fatalf("NewPage %d: %v", i, err)
		}
		pages[i] = p
	}

	// Navigate all pages concurrently.
	errs := make(chan error, n)
	for i, p := range pages {
		go func(p *page.Page, path string) {
			errs <- p.Navigate(context.Background(), ts.URL+path)
		}(p, fmt.Sprintf("/%d", i))
	}

	for i := 0; i < n; i++ {
		if err := <-errs; err != nil {
			t.Errorf("concurrent navigate error: %v", err)
		}
	}

	// Verify each page got its own content.
	for i, p := range pages {
		expected := fmt.Sprintf("/%d", i)
		got := p.Document().Body().TextContent()
		if got != expected {
			t.Errorf("page %d: body = %q, want %q", i, got, expected)
		}
	}
}

func TestBrowserOptions(t *testing.T) {
	b := browser.New(browser.WithUserAgent("TestAgent/1.0"))
	defer b.Close()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<html><body>%s</body></html>", r.UserAgent())
	}))
	defer ts.Close()

	p, err := b.DefaultContext().NewPage()
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	if err := p.Navigate(context.Background(), ts.URL); err != nil {
		t.Fatalf("Navigate: %v", err)
	}
	body := p.Document().Body().TextContent()
	if body != "TestAgent/1.0" {
		t.Errorf("user agent = %q, want %q", body, "TestAgent/1.0")
	}
}
