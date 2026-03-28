package cdp

import (
	"github.com/nyasuto/uzura/internal/js"
	"github.com/nyasuto/uzura/internal/network"
	"github.com/nyasuto/uzura/internal/page"
)

// Setup creates a fully wired Page and registers all CDP domains on the server.
// It returns the Page so callers can use it for further operations.
func Setup(s *Server) *page.Page {
	runtimeDomain := NewRuntimeDomain(nil)
	networkDomain := NewNetworkDomain(nil)

	fetcher := network.NewFetcher(nil)
	p := page.New(&page.Options{
		Fetcher:         fetcher,
		VMOptions:       []js.Option{js.WithConsoleCallback(runtimeDomain.ConsoleCallback())},
		NetworkObserver: networkDomain.Observer(),
	})

	runtimeDomain.SetPage(p)
	networkDomain.SetPage(p)

	pageDomain := NewPageDomain(p)
	domDomain := NewDOMDomain(p)

	pageDomain.Register(s)
	domDomain.Register(s)
	runtimeDomain.Register(s)
	networkDomain.Register(s)
	registerStubs(s)

	return p
}
