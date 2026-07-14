package page

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestNavigate_ExecutesScriptsWithFetch models a CSR page: the initial HTML
// contains only an empty #root, and a script fetches JSON and builds the
// DOM from it. This verifies Navigate now executes scripts and drives the
// event loop until the fetch resolves.
func TestNavigate_ExecutesScriptsWithFetch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/post", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"title":"CSR Title","body":"loaded via fetch"}`)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html><html><body><div id="root">Loading...</div>
<script>
fetch("/api/post").then(function(r){ return r.json(); }).then(function(data){
  var h1 = document.createElement("h1");
  h1.textContent = data.title;
  document.getElementById("root").textContent = "";
  document.getElementById("root").appendChild(h1);
  var p = document.createElement("p");
  p.textContent = data.body;
  document.getElementById("root").appendChild(p);
});
</script></body></html>`)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	p := New(nil)
	defer p.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := p.Navigate(ctx, ts.URL); err != nil {
		t.Fatal(err)
	}

	h1, err := p.Document().QuerySelector("h1")
	if err != nil {
		t.Fatal(err)
	}
	if h1 == nil {
		t.Fatal("h1 not found: scripts did not run or fetch did not complete")
	}
	if h1.TextContent() != "CSR Title" {
		t.Errorf("h1 = %q, want %q", h1.TextContent(), "CSR Title")
	}
	if strings.Contains(p.Document().Body().TextContent(), "Loading...") {
		t.Error("Loading placeholder still present")
	}
}

// TestNavigate_ExecutesScriptsWithXHR is the XHR-based equivalent (axios-like
// path) of TestNavigate_ExecutesScriptsWithFetch, using the same server.
func TestNavigate_ExecutesScriptsWithXHR(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/post", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"title":"CSR Title","body":"loaded via fetch"}`)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html><html><body><div id="root">Loading...</div>
<script>
var xhr = new XMLHttpRequest();
xhr.responseType = "json";
xhr.onload = function() {
  var h1 = document.createElement("h1");
  h1.textContent = xhr.response.title;
  document.getElementById("root").textContent = "";
  document.getElementById("root").appendChild(h1);
};
xhr.open("GET", "/api/post");
xhr.send();
</script></body></html>`)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	p := New(nil)
	defer p.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := p.Navigate(ctx, ts.URL); err != nil {
		t.Fatal(err)
	}

	h1, err := p.Document().QuerySelector("h1")
	if err != nil {
		t.Fatal(err)
	}
	if h1 == nil {
		t.Fatal("h1 not found: scripts did not run or XHR did not complete")
	}
	if h1.TextContent() != "CSR Title" {
		t.Errorf("h1 = %q, want %q", h1.TextContent(), "CSR Title")
	}
}

// TestNavigate_WebAPIsAvailable is a smoke test confirming localStorage,
// history/location, and fetch are all visible to page scripts after
// Navigate.
func TestNavigate_WebAPIsAvailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html><html><body><div id="out"></div>
<script>
localStorage.setItem("k", "v1");
history.pushState({}, "", "/pushed");
document.getElementById("out").textContent =
  [localStorage.getItem("k"), location.pathname, typeof fetch].join("|");
</script></body></html>`)
	}))
	defer srv.Close()

	p := New(nil)
	defer p.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := p.Navigate(ctx, srv.URL); err != nil {
		t.Fatal(err)
	}

	out := p.Document().GetElementById("out")
	if out == nil {
		t.Fatal("#out not found")
	}
	want := "v1|/pushed|function"
	if got := out.TextContent(); got != want {
		t.Errorf("#out textContent = %q, want %q", got, want)
	}
}

// TestNavigate_TopLevelFetchRespectsPageDeadline guards against Finding 2 of
// the js-web-apis event-loop review: a fetch() called directly from a
// script's top-level body (as opposed to from inside a setTimeout/task
// callback that only runs once the event loop is already pumping) must
// inherit the page's navigation deadline, not context.Background().
//
// Before the fix, runScripts executed the document's scripts before priming
// the event loop's context (RunEventLoopContext only sets it afterward), so
// this fetch's derived request context had no deadline: /slow's handler
// below would never observe cancellation, and the request goroutine (plus
// its context.WithCancel) would leak past Navigate's return, only settling
// whenever the hung request happened to end on its own (never, here).
func TestNavigate_TopLevelFetchRespectsPageDeadline(t *testing.T) {
	serverCanceled := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("/slow", func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		close(serverCanceled)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html><html><body>
<script>
fetch("/slow");
</script></body></html>`)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	p := New(nil)
	defer p.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	// Partial-result policy: the page deadline firing mid-fetch must never
	// fail Navigate itself.
	if err := p.Navigate(ctx, ts.URL); err != nil {
		t.Fatalf("Navigate: %v", err)
	}

	select {
	case <-serverCanceled:
		// Good: the top-level fetch's request context was derived from the
		// page deadline and got canceled by it, instead of leaking on
		// context.Background().
	case <-time.After(5 * time.Second):
		t.Fatal("server handler never observed cancellation: top-level fetch escaped the page deadline (leaked context/goroutine)")
	}
}

// TestNavigate_HandleFulfillExecutesScripts confirms that a mocked response
// (via OnRequest + Request.Fulfill, the same path CDP's Fetch domain drives)
// still executes inline scripts and builds the DOM, exactly like a real
// network response would. This exercises handleFulfill's runScripts call.
func TestNavigate_HandleFulfillExecutesScripts(t *testing.T) {
	p := New(nil)
	defer p.Close()

	p.OnRequest(func(req *Request) {
		req.Fulfill(FulfillOption{
			Status:  200,
			Headers: map[string]string{"Content-Type": "text/html"},
			Body: []byte(`<!DOCTYPE html><html><body><div id="root"></div>
<script>
var h1 = document.createElement("h1");
h1.textContent = "mocked title";
document.getElementById("root").appendChild(h1);
</script></body></html>`),
		})
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Fulfilled responses never hit the network, so any URL works here.
	if err := p.Navigate(ctx, "http://mocked.invalid/page"); err != nil {
		t.Fatal(err)
	}

	h1, err := p.Document().QuerySelector("h1")
	if err != nil {
		t.Fatal(err)
	}
	if h1 == nil {
		t.Fatal("h1 not found: handleFulfill did not execute the mocked response's script")
	}
	if h1.TextContent() != "mocked title" {
		t.Errorf("h1 = %q, want %q", h1.TextContent(), "mocked title")
	}
}
