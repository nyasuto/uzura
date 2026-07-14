package js

import (
	"context"
	"io"
	"testing"
)

func TestResolveURL(t *testing.T) {
	tests := []struct {
		name, base, ref, want string
	}{
		{"absolute untouched", "https://example.com/app/", "https://api.example.com/v1", "https://api.example.com/v1"},
		{"relative path", "https://example.com/app/index.html", "api/items", "https://example.com/app/api/items"},
		{"root relative", "https://example.com/app/index.html", "/v1/items", "https://example.com/v1/items"},
		{"no base", "", "api/items", "api/items"},
		{"scheme relative", "https://example.com/", "//cdn.example.com/a.json", "https://cdn.example.com/a.json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := New(WithWriter(io.Discard))
			if tt.base != "" {
				vm.SetBaseURL(tt.base)
			}
			if got := vm.resolveURL(tt.ref); got != tt.want {
				t.Errorf("resolveURL(%q) = %q, want %q", tt.ref, got, tt.want)
			}
		})
	}
}

func TestHTTPClientDefault(t *testing.T) {
	vm := New(WithWriter(io.Discard))
	_, err := vm.httpClient()(context.Background(), HTTPRequest{Method: "GET", URL: "https://example.com"})
	if err == nil {
		t.Error("expected error from default client, got nil")
	}
}
