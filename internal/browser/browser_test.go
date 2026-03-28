package browser_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nyasuto/uzura/internal/browser"
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
