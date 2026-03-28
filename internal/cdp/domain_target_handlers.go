package cdp

import (
	"encoding/json"
	"fmt"
)

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
		sessionID := generateSessionID()

		td.mu.Lock()
		t.Attached = true
		td.sessions[sessionID] = t.TargetID
		p := td.pages[t.TargetID]
		setupFn := td.pageSetup
		td.mu.Unlock()

		td.setupScope(sessionID, p, setupFn)

		_ = sess.SendEvent("Target.attachedToTarget", map[string]interface{}{
			"sessionId":          sessionID,
			"targetInfo":         t,
			"waitingForDebugger": false,
		})
	}

	return r, nil, nil
}

func (td *TargetDomain) handleCreateTarget(_ *Session, params json.RawMessage) (json.RawMessage, []Event, error) {
	if td.createPage == nil {
		return nil, nil, fmt.Errorf("page creation not configured")
	}

	var req struct {
		URL              string `json:"url"`
		BrowserContextID string `json:"browserContextId"`
	}
	if params != nil {
		_ = json.Unmarshal(params, &req)
	}

	p, err := td.createPage()
	if err != nil {
		return nil, nil, err
	}

	contextID := req.BrowserContextID
	if contextID == "" {
		contextID = "default-context"
	}
	td.AddPage(p, contextID)

	sessionID := generateSessionID()

	td.mu.Lock()
	info := td.targets[p.ID()]
	info.Attached = true
	if req.URL != "" {
		info.URL = req.URL
	}
	td.sessions[sessionID] = p.ID()
	setupFn := td.pageSetup
	td.mu.Unlock()

	td.setupScope(sessionID, p, setupFn)

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

func (td *TargetDomain) handleAttachToTarget(_ *Session, params json.RawMessage) (json.RawMessage, []Event, error) {
	var req struct {
		TargetID string `json:"targetId"`
		Flatten  bool   `json:"flatten"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, nil, err
	}

	td.mu.Lock()
	info, ok := td.targets[req.TargetID]
	if !ok {
		td.mu.Unlock()
		return nil, nil, fmt.Errorf("target not found: %s", req.TargetID)
	}
	info.Attached = true
	p := td.pages[req.TargetID]
	sessionID := generateSessionID()
	td.sessions[sessionID] = req.TargetID
	setupFn := td.pageSetup
	td.mu.Unlock()

	td.setupScope(sessionID, p, setupFn)

	r, _ := json.Marshal(map[string]interface{}{
		"sessionId": sessionID,
	})
	return r, nil, nil
}

func (td *TargetDomain) handleDetachFromTarget(params json.RawMessage) (json.RawMessage, error) {
	var req struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, err
	}

	td.mu.Lock()
	targetID, ok := td.sessions[req.SessionID]
	if !ok {
		td.mu.Unlock()
		return nil, fmt.Errorf("session not found: %s", req.SessionID)
	}
	delete(td.sessions, req.SessionID)
	if info, exists := td.targets[targetID]; exists {
		info.Attached = false
	}
	td.mu.Unlock()

	if td.server != nil {
		td.server.RemoveScope(req.SessionID)
	}

	return json.Marshal(struct{}{})
}
