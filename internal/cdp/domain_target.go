package cdp

import (
	"encoding/json"
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

// TargetDomain manages CDP targets (pages) and emits lifecycle events.
type TargetDomain struct {
	mu      sync.Mutex
	targets map[string]*TargetInfo // targetID → info
	pages   map[string]*page.Page  // targetID → page
	server  *Server

	// createPage is called when Target.createTarget is received.
	createPage func() (*page.Page, error)
}

// NewTargetDomain creates a TargetDomain.
func NewTargetDomain(server *Server, createPage func() (*page.Page, error)) *TargetDomain {
	return &TargetDomain{
		targets:    make(map[string]*TargetInfo),
		pages:      make(map[string]*page.Page),
		server:     server,
		createPage: createPage,
	}
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
	s.Handle("Target.attachToTarget", td.handleAttachToTarget)
	s.Handle("Target.getBrowserContexts", handleGetBrowserContexts)
}

func (td *TargetDomain) handleSetDiscoverTargets(sess *Session, _ json.RawMessage) (json.RawMessage, []Event, error) {
	r, _ := json.Marshal(struct{}{})

	td.mu.Lock()
	targets := make([]*TargetInfo, 0, len(td.targets))
	for _, t := range td.targets {
		targets = append(targets, t)
	}
	td.mu.Unlock()

	for _, t := range targets {
		_ = sess.SendEvent("Target.targetCreated", map[string]interface{}{
			"targetInfo": t,
		})
	}

	return r, nil, nil
}

func (td *TargetDomain) handleSetAutoAttach(sess *Session, _ json.RawMessage) (json.RawMessage, []Event, error) {
	r, _ := json.Marshal(struct{}{})

	td.mu.Lock()
	targets := make([]*TargetInfo, 0, len(td.targets))
	for _, t := range td.targets {
		targets = append(targets, t)
	}
	td.mu.Unlock()

	for _, t := range targets {
		sessionID := fmt.Sprintf("session-%d", sessionIDCounter.Add(1))
		t.Attached = true
		_ = sess.SendEvent("Target.attachedToTarget", map[string]interface{}{
			"sessionId":          sessionID,
			"targetInfo":         t,
			"waitingForDebugger": false,
		})
	}

	return r, nil, nil
}

func (td *TargetDomain) handleCreateTarget(sess *Session, _ json.RawMessage) (json.RawMessage, []Event, error) {
	if td.createPage == nil {
		return nil, nil, fmt.Errorf("page creation not configured")
	}

	p, err := td.createPage()
	if err != nil {
		return nil, nil, err
	}

	td.AddPage(p, "default-context")

	sessionID := fmt.Sprintf("session-%d", sessionIDCounter.Add(1))
	info := td.targets[p.ID()]
	info.Attached = true

	r, _ := json.Marshal(map[string]interface{}{
		"targetId": p.ID(),
	})

	attachedParams, _ := json.Marshal(map[string]interface{}{
		"sessionId":          sessionID,
		"targetInfo":         info,
		"waitingForDebugger": false,
	})

	return r, []Event{
		{Method: "Target.attachedToTarget", Params: attachedParams},
	}, nil
}

func (td *TargetDomain) handleCloseTarget(params json.RawMessage) (json.RawMessage, error) {
	var req struct {
		TargetID string `json:"targetId"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, err
	}

	td.mu.Lock()
	p, ok := td.pages[req.TargetID]
	td.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("target not found: %s", req.TargetID)
	}

	_ = p.Close()
	td.RemovePage(req.TargetID)

	return json.Marshal(map[string]interface{}{
		"success": true,
	})
}

func (td *TargetDomain) handleGetTargetInfo(params json.RawMessage) (json.RawMessage, error) {
	var req struct {
		TargetID string `json:"targetId"`
	}
	_ = json.Unmarshal(params, &req)

	td.mu.Lock()
	info, ok := td.targets[req.TargetID]
	td.mu.Unlock()

	if !ok {
		// Fallback: return first target.
		targets := td.Targets()
		if len(targets) > 0 {
			info = targets[0]
		} else {
			return json.Marshal(map[string]interface{}{
				"targetInfo": map[string]interface{}{
					"targetId": "unknown",
					"type":     "page",
				},
			})
		}
	}

	return json.Marshal(map[string]interface{}{
		"targetInfo": info,
	})
}

func (td *TargetDomain) handleGetTargets(_ json.RawMessage) (json.RawMessage, error) {
	targets := td.Targets()
	return json.Marshal(map[string]interface{}{
		"targetInfos": targets,
	})
}

func (td *TargetDomain) handleAttachToTarget(params json.RawMessage) (json.RawMessage, error) {
	var req struct {
		TargetID string `json:"targetId"`
	}
	_ = json.Unmarshal(params, &req)

	td.mu.Lock()
	if info, ok := td.targets[req.TargetID]; ok {
		info.Attached = true
	}
	td.mu.Unlock()

	sessionID := fmt.Sprintf("session-%d", sessionIDCounter.Add(1))
	return json.Marshal(map[string]interface{}{
		"sessionId": sessionID,
	})
}
