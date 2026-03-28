package network

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCookieJarPersistence(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/set":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc123", Path: "/"})
			w.WriteHeader(http.StatusOK)
		case "/get":
			c, err := r.Cookie("session")
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("no cookie"))
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(c.Value))
		}
	}))
	defer srv.Close()

	f := NewFetcher(&FetcherOptions{EnableCookies: true})

	// Set cookie
	resp, err := f.Fetch(srv.URL + "/set")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// Cookie should be sent back
	resp, err = f.Fetch(srv.URL + "/get")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200 (cookie not sent back)", resp.StatusCode)
	}
}

func TestCookieJarAcrossRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "token", Value: "xyz", Path: "/"})
			http.Redirect(w, r, "/dashboard", http.StatusFound)
		case "/dashboard":
			c, err := r.Cookie("token")
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("welcome:" + c.Value))
		}
	}))
	defer srv.Close()

	f := NewFetcher(&FetcherOptions{EnableCookies: true})
	resp, err := f.Fetch(srv.URL + "/login")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestNoCookieJarByDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/set":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc", Path: "/"})
			w.WriteHeader(http.StatusOK)
		case "/get":
			_, err := r.Cookie("session")
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	f := NewFetcher(nil) // no cookies
	resp, err := f.Fetch(srv.URL + "/set")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	resp, err = f.Fetch(srv.URL + "/get")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400 (cookie should not be sent without jar)", resp.StatusCode)
	}
}
