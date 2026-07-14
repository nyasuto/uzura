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
