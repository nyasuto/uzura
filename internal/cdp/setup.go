package cdp

import (
	"encoding/json"

	"github.com/nyasuto/uzura/internal/js"
	"github.com/nyasuto/uzura/internal/network"
	"github.com/nyasuto/uzura/internal/page"
)

// Setup creates a fully wired Page and registers all CDP domains on the server.
// It returns the Page so callers can use it for further operations.
func Setup(s *Server) *page.Page {
	runtimeDomain := NewRuntimeDomain(nil)
	networkDomain := NewNetworkDomain(nil)
	fetchDomain := NewFetchDomain(nil)

	fetcher := network.NewFetcher(nil)
	p := page.New(&page.Options{
		Fetcher:            fetcher,
		VMOptions:          []js.Option{js.WithConsoleCallback(runtimeDomain.ConsoleCallback())},
		NetworkObserver:    networkDomain.Observer(),
		RequestInterceptor: fetchDomain.Interceptor(),
	})

	runtimeDomain.SetPage(p)
	networkDomain.SetPage(p)
	fetchDomain.SetPage(p)

	pageDomain := NewPageDomain(p)
	domDomain := NewDOMDomain(p)

	// Target domain: manages page targets with session multiplexing.
	targetDomain := NewTargetDomain(s, func() (*page.Page, error) {
		newPage := page.New(&page.Options{Fetcher: fetcher})
		return newPage, nil
	})

	// Wire per-page domain setup for session multiplexing.
	targetDomain.SetPageSetup(func(pg *page.Page, sc *handlerScope) {
		rt := NewRuntimeDomain(pg)
		nd := NewNetworkDomain(pg)
		fd := NewFetchDomain(pg)
		pd := NewPageDomain(pg)
		dd := NewDOMDomain(pg)

		pg.SetNetworkObserver(nd.Observer())
		pg.SetRequestInterceptor(fd.Interceptor())

		pd.Register(sc)
		dd.Register(sc)
		rt.Register(sc)
		nd.Register(sc)
		fd.Register(sc)
		registerScopeStubs(sc)
	})

	targetDomain.AddPage(p, "default-context")

	// Register global handlers (used when no sessionId is specified).
	pageDomain.Register(s)
	domDomain.Register(s)
	runtimeDomain.Register(s)
	networkDomain.Register(s)
	fetchDomain.Register(s)
	targetDomain.Register(s)
	registerStubs(s)

	return p
}

// registerScopeStubs adds common stubs to a per-session scope.
func registerScopeStubs(sc *handlerScope) {
	empty := func(_ json.RawMessage) (json.RawMessage, error) {
		return json.Marshal(struct{}{})
	}
	sc.Handle("Page.setLifecycleEventsEnabled", empty)
	sc.Handle("Page.setInterceptFileChooserDialog", empty)
	sc.Handle("Page.getNavigationHistory", handleGetNavigationHistory)
	sc.Handle("Page.createIsolatedWorld", handleCreateIsolatedWorld)
	sc.HandleSession("Page.addScriptToEvaluateOnNewDocument", handleAddScriptToEvaluateOnNewDocument)
	sc.HandleSession("Runtime.runIfWaitingForDebugger", handleRunIfWaitingForDebugger)
	sc.Handle("Emulation.setDeviceMetricsOverride", empty)
	sc.Handle("Emulation.setTouchEmulationEnabled", empty)
	sc.Handle("Emulation.setFocusEmulationEnabled", empty)
	sc.Handle("Emulation.setEmulatedMedia", empty)
	sc.Handle("Log.enable", empty)
	sc.Handle("Performance.enable", empty)
	sc.Handle("Security.enable", empty)
	sc.Handle("Inspector.enable", empty)
	sc.Handle("ServiceWorker.enable", empty)
	sc.Handle("CSS.enable", empty)
	sc.Handle("Overlay.enable", empty)
}
