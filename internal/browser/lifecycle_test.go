package browser_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nyasuto/uzura/internal/browser"
)

// --- Phase 8.3: Page goroutine management with context cancellation ---

func TestPageCancelOnClose(t *testing.T) {
	// A slow server that blocks until the request context is canceled.
	started := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done() // block until client cancels
	}))
	defer ts.Close()

	b := browser.New()
	defer b.Close()

	p, err := b.DefaultContext().NewPage()
	if err != nil {
		t.Fatal(err)
	}

	// Start navigation in background goroutine.
	navErr := make(chan error, 1)
	go func() {
		navErr <- p.Navigate(context.Background(), ts.URL)
	}()

	// Wait for server to receive the request.
	<-started

	// Close the page — should cancel the in-flight navigation.
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-navErr:
		if err == nil {
			t.Error("Navigate should have returned an error after page close")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Navigate did not return after page close (context not canceled)")
	}
}

func TestBrowserCloseCancel(t *testing.T) {
	started := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	defer ts.Close()

	b := browser.New()
	p, err := b.DefaultContext().NewPage()
	if err != nil {
		t.Fatal(err)
	}

	navErr := make(chan error, 1)
	go func() {
		navErr <- p.Navigate(context.Background(), ts.URL)
	}()

	<-started

	// Close browser — should cascade cancel to all pages.
	_ = b.Close()

	select {
	case err := <-navErr:
		if err == nil {
			t.Error("Navigate should have returned an error after browser close")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Navigate did not return after browser close")
	}
}

// --- Phase 8.3: Graceful shutdown order ---

func TestGracefulShutdownOrder(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body>ok</body></html>")
	}))
	defer ts.Close()

	b := browser.New()
	ctx1 := b.NewContext()
	ctx2 := b.NewContext()

	p1, _ := ctx1.NewPage()
	p2, _ := ctx1.NewPage()
	p3, _ := ctx2.NewPage()

	bgCtx := context.Background()
	_ = p1.Navigate(bgCtx, ts.URL)
	_ = p2.Navigate(bgCtx, ts.URL)
	_ = p3.Navigate(bgCtx, ts.URL)

	// Close browser.
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}

	// All pages closed.
	for i, p := range []*interface{ IsClosed() bool }{} {
		_ = i
		_ = p
	}
	if !p1.IsClosed() {
		t.Error("p1 should be closed")
	}
	if !p2.IsClosed() {
		t.Error("p2 should be closed")
	}
	if !p3.IsClosed() {
		t.Error("p3 should be closed")
	}

	// Resources released.
	if p1.Document() != nil {
		t.Error("p1 document should be nil")
	}
	if p2.URL() != "" {
		t.Error("p2 URL should be empty")
	}

	// Contexts empty.
	if len(b.Contexts()) != 0 {
		t.Error("all contexts should be removed")
	}
}

// --- Phase 8.3: Memory leak prevention ---

func TestPageCloseReleasesVM(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body><script>var x = 1;</script></body></html>")
	}))
	defer ts.Close()

	b := browser.New()
	defer b.Close()

	p, _ := b.DefaultContext().NewPage()
	_ = p.Navigate(context.Background(), ts.URL)

	// Access VM to ensure it's created.
	vm := p.VM()
	if vm == nil {
		t.Fatal("VM should not be nil before close")
	}

	_ = p.Close()

	// After close, Document and URL should be nil/empty.
	if p.Document() != nil {
		t.Error("Document should be nil after close")
	}
	if p.URL() != "" {
		t.Error("URL should be empty after close")
	}
}

func TestContextCloseReleasesAllPages(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body>ok</body></html>")
	}))
	defer ts.Close()

	b := browser.New()
	defer b.Close()

	ctx := b.NewContext()
	const n = 10
	for i := 0; i < n; i++ {
		p, err := ctx.NewPage()
		if err != nil {
			t.Fatal(err)
		}
		_ = p.Navigate(context.Background(), ts.URL)
	}

	if len(ctx.Pages()) != n {
		t.Fatalf("expected %d pages, got %d", n, len(ctx.Pages()))
	}

	_ = ctx.Close()

	if len(ctx.Pages()) != 0 {
		t.Errorf("expected 0 pages after close, got %d", len(ctx.Pages()))
	}
}

// --- Phase 8.3: MaxPages limit ---

func TestMaxPages(t *testing.T) {
	b := browser.New(browser.WithMaxPages(3))
	defer b.Close()

	ctx := b.DefaultContext()

	// Create up to the limit.
	for i := 0; i < 3; i++ {
		_, err := ctx.NewPage()
		if err != nil {
			t.Fatalf("NewPage %d: unexpected error: %v", i, err)
		}
	}

	// Next creation should fail.
	_, err := ctx.NewPage()
	if err == nil {
		t.Fatal("expected error when exceeding MaxPages")
	}

	// After closing one page, we can create again.
	pages := ctx.Pages()
	_ = pages[0].Close()

	_, err = ctx.NewPage()
	if err != nil {
		t.Fatalf("NewPage after closing one should succeed: %v", err)
	}
}

func TestMaxPagesAcrossContexts(t *testing.T) {
	b := browser.New(browser.WithMaxPages(4))
	defer b.Close()

	ctx1 := b.DefaultContext()
	ctx2 := b.NewContext()

	// Create 2 pages in each context (total 4 = max).
	for i := 0; i < 2; i++ {
		if _, err := ctx1.NewPage(); err != nil {
			t.Fatal(err)
		}
		if _, err := ctx2.NewPage(); err != nil {
			t.Fatal(err)
		}
	}

	// Both contexts should fail to create more.
	if _, err := ctx1.NewPage(); err == nil {
		t.Error("ctx1 should fail: max pages reached")
	}
	if _, err := ctx2.NewPage(); err == nil {
		t.Error("ctx2 should fail: max pages reached")
	}
}

func TestMaxPagesZeroMeansUnlimited(t *testing.T) {
	b := browser.New() // no MaxPages set
	defer b.Close()

	ctx := b.DefaultContext()
	for i := 0; i < 20; i++ {
		if _, err := ctx.NewPage(); err != nil {
			t.Fatalf("NewPage %d: %v", i, err)
		}
	}
}

// --- Phase 8.3: Race condition stress test ---

func TestMassPageCreateClose(t *testing.T) {
	b := browser.New()
	defer b.Close()

	ctx := b.DefaultContext()
	const n = 50
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, err := ctx.NewPage()
			if err != nil {
				return
			}
			_ = p.Close()
		}()
	}

	wg.Wait()

	// All pages should be cleaned up.
	if len(ctx.Pages()) != 0 {
		t.Errorf("expected 0 pages after mass close, got %d", len(ctx.Pages()))
	}
}

func TestConcurrentBrowserClose(t *testing.T) {
	b := browser.New()

	ctx := b.DefaultContext()
	for i := 0; i < 10; i++ {
		_, _ = ctx.NewPage()
	}

	// Close browser concurrently from multiple goroutines.
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = b.Close()
		}()
	}
	wg.Wait()
}

func TestConcurrentCreateAndBrowserClose(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body>ok</body></html>")
	}))
	defer ts.Close()

	b := browser.New()
	ctx := b.DefaultContext()

	var wg sync.WaitGroup
	var created atomic.Int32

	// Spawn goroutines creating pages and navigating.
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, err := ctx.NewPage()
			if err != nil {
				return // browser/context already closed
			}
			created.Add(1)
			_ = p.Navigate(context.Background(), ts.URL)
		}()
	}

	// Give some goroutines time to start.
	time.Sleep(10 * time.Millisecond)

	// Close browser while pages are being created.
	_ = b.Close()

	wg.Wait()
	// No panic or deadlock means success.
}

func TestNewPageAfterBrowserClose(t *testing.T) {
	b := browser.New()
	ctx := b.DefaultContext()
	_ = b.Close()

	_, err := ctx.NewPage()
	if err == nil {
		t.Error("NewPage after browser close should return error")
	}
}

func TestNewContextAfterBrowserClose(t *testing.T) {
	b := browser.New()
	_ = b.Close()

	ctx := b.NewContext()
	if ctx != nil {
		t.Error("NewContext after browser close should return nil")
	}
}
