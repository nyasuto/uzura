package cdp

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/nyasuto/uzura/internal/page"
)

var sessionIDCounter atomic.Int64

// TargetInfo holds metadata for a CDP target.
type TargetInfo struct {
	TargetID         string `json:"targetId"`
	Type             string `json:"type"`
	Title            string `json:"title"`
	URL              string `json:"url"`
	Attached         bool   `json:"attached"`
	BrowserContextID string `json:"browserContextId"`
}

// PageSetupFunc sets up per-page CDP domain handlers on a scope.
type PageSetupFunc func(p *page.Page, sc *handlerScope)

// TargetDomain manages CDP targets (pages) and emits lifecycle events.
type TargetDomain struct {
	mu       sync.Mutex
	targets  map[string]*TargetInfo // targetID → info
	pages    map[string]*page.Page  // targetID → page
	sessions map[string]string      // sessionID → targetID
	server   *Server

	// createPage is called when Target.createTarget is received.
	createPage func() (*page.Page, error)

	// pageSetup wires per-page domain handlers into a session scope.
	pageSetup PageSetupFunc
}

// NewTargetDomain creates a TargetDomain.
func NewTargetDomain(server *Server, createPage func() (*page.Page, error)) *TargetDomain {
	return &TargetDomain{
		targets:    make(map[string]*TargetInfo),
		pages:      make(map[string]*page.Page),
		sessions:   make(map[string]string),
		server:     server,
		createPage: createPage,
	}
}

// SetPageSetup sets the callback that wires per-page domain handlers.
func (td *TargetDomain) SetPageSetup(fn PageSetupFunc) {
	td.mu.Lock()
	defer td.mu.Unlock()
	td.pageSetup = fn
}

// AddPage registers an existing page as a target and broadcasts targetCreated.
func (td *TargetDomain) AddPage(p *page.Page, contextID string) {
	info := &TargetInfo{
		TargetID:         p.ID(),
		Type:             "page",
		Title:            "",
		URL:              "about:blank",
		Attached:         false,
		BrowserContextID: contextID,
	}

	td.mu.Lock()
	td.targets[p.ID()] = info
	td.pages[p.ID()] = p
	td.mu.Unlock()

	if td.server != nil {
		td.server.Broadcast("Target.targetCreated", map[string]interface{}{
			"targetInfo": info,
		})
	}
}

// RemovePage unregisters a page and broadcasts targetDestroyed.
func (td *TargetDomain) RemovePage(targetID string) {
	td.mu.Lock()
	_, ok := td.targets[targetID]
	if ok {
		delete(td.targets, targetID)
		delete(td.pages, targetID)
	}
	// Clean up any sessions pointing to this target.
	for sid, tid := range td.sessions {
		if tid == targetID {
			delete(td.sessions, sid)
			if td.server != nil {
				td.server.RemoveScope(sid)
			}
		}
	}
	td.mu.Unlock()

	if ok && td.server != nil {
		td.server.Broadcast("Target.targetDestroyed", map[string]interface{}{
			"targetId": targetID,
		})
	}
}

// Page returns the page for the given target ID.
func (td *TargetDomain) Page(targetID string) *page.Page {
	td.mu.Lock()
	defer td.mu.Unlock()
	return td.pages[targetID]
}

// Targets returns all target infos.
func (td *TargetDomain) Targets() []*TargetInfo {
	td.mu.Lock()
	defer td.mu.Unlock()
	result := make([]*TargetInfo, 0, len(td.targets))
	for _, t := range td.targets {
		result = append(result, t)
	}
	return result
}

// Register registers Target domain handlers on the server.
func (td *TargetDomain) Register(s *Server) {
	s.HandleSession("Target.setDiscoverTargets", td.handleSetDiscoverTargets)
	s.HandleSession("Target.setAutoAttach", td.handleSetAutoAttach)
	s.HandleSession("Target.createTarget", td.handleCreateTarget)
	s.Handle("Target.closeTarget", td.handleCloseTarget)
	s.Handle("Target.getTargetInfo", td.handleGetTargetInfo)
	s.Handle("Target.getTargets", td.handleGetTargets)
	s.HandleSession("Target.attachToTarget", td.handleAttachToTarget)
	s.Handle("Target.detachFromTarget", td.handleDetachFromTarget)
	s.Handle("Target.getBrowserContexts", handleGetBrowserContexts)
}

// setupScope creates a handler scope for a session and wires per-page handlers.
func (td *TargetDomain) setupScope(sessionID string, p *page.Page, setupFn PageSetupFunc) {
	if td.server == nil {
		return
	}
	sc := td.server.RegisterScope(sessionID)
	if setupFn != nil && p != nil {
		setupFn(p, sc)
	}
}

// generateSessionID creates a unique session identifier.
func generateSessionID() string {
	return fmt.Sprintf("session-%d", sessionIDCounter.Add(1))
}
