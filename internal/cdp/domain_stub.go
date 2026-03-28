package cdp

import "encoding/json"

// registerStubs adds no-op handlers for CDP methods that Puppeteer and
// Playwright call during connection setup. Without these, the clients
// receive -32601 errors and may fail to initialize.
func registerStubs(s *Server) {
	empty := emptyHandler

	// Browser domain.
	s.Handle("Browser.getVersion", handleBrowserGetVersion(s))
	s.Handle("Browser.setDownloadBehavior", empty)

	// Page extras that Puppeteer expects.
	s.Handle("Page.setLifecycleEventsEnabled", empty)
	s.HandleSession("Page.addScriptToEvaluateOnNewDocument", handleAddScriptToEvaluateOnNewDocument)
	s.Handle("Page.createIsolatedWorld", handleCreateIsolatedWorld)
	s.Handle("Page.setInterceptFileChooserDialog", empty)
	s.Handle("Page.getNavigationHistory", handleGetNavigationHistory)

	// Runtime extras.
	s.HandleSession("Runtime.runIfWaitingForDebugger", handleRunIfWaitingForDebugger)

	// Emulation stubs.
	s.Handle("Emulation.setDeviceMetricsOverride", empty)
	s.Handle("Emulation.setTouchEmulationEnabled", empty)
	s.Handle("Emulation.setFocusEmulationEnabled", empty)
	s.Handle("Emulation.setEmulatedMedia", empty)

	// Log and Performance stubs.
	s.Handle("Log.enable", empty)
	s.Handle("Performance.enable", empty)

	// Security stubs.
	s.Handle("Security.enable", empty)

	// Inspector stubs.
	s.Handle("Inspector.enable", empty)

	// ServiceWorker stubs.
	s.Handle("ServiceWorker.enable", empty)

	// Fetch and CSS stubs.
	s.Handle("Fetch.enable", empty)
	s.Handle("Fetch.disable", empty)
	s.Handle("CSS.enable", empty)

	// Overlay stubs.
	s.Handle("Overlay.enable", empty)
}

func emptyHandler(_ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(struct{}{})
}

func handleBrowserGetVersion(s *Server) Handler {
	return func(_ json.RawMessage) (json.RawMessage, error) {
		return json.Marshal(map[string]interface{}{
			"protocolVersion": s.protocolVersion,
			"product":         s.browserVersion,
			"userAgent":       s.userAgent,
			"jsVersion":       "goja",
		})
	}
}

func handleCreateIsolatedWorld(_ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(map[string]interface{}{
		"executionContextId": 1,
	})
}

func handleGetBrowserContexts(_ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(map[string]interface{}{
		"browserContextIds": []string{},
	})
}

func handleRunIfWaitingForDebugger(_ *Session, _ json.RawMessage) (json.RawMessage, []Event, error) {
	r, err := json.Marshal(struct{}{})
	if err != nil {
		return nil, nil, err
	}

	ctxParams, _ := json.Marshal(map[string]interface{}{
		"context": map[string]interface{}{
			"id":     1,
			"origin": "",
			"name":   "",
			"auxData": map[string]interface{}{
				"isDefault": true,
				"type":      "default",
				"frameId":   "main",
			},
		},
	})

	return r, []Event{
		{Method: "Runtime.executionContextCreated", Params: ctxParams},
	}, nil
}

var worldContextID = 2

func handleAddScriptToEvaluateOnNewDocument(_ *Session, params json.RawMessage) (json.RawMessage, []Event, error) {
	var p struct {
		WorldName string `json:"worldName"`
	}
	_ = json.Unmarshal(params, &p)

	r, err := json.Marshal(map[string]interface{}{
		"identifier": "1",
	})
	if err != nil {
		return nil, nil, err
	}

	var events []Event

	if p.WorldName != "" {
		ctxData, _ := json.Marshal(map[string]interface{}{
			"context": map[string]interface{}{
				"id":     worldContextID,
				"origin": "",
				"name":   p.WorldName,
				"auxData": map[string]interface{}{
					"isDefault": false,
					"type":      "isolated",
					"frameId":   "main",
				},
			},
		})
		events = append(events, Event{
			Method: "Runtime.executionContextCreated",
			Params: ctxData,
		})
		worldContextID++
	}

	return r, events, nil
}

func handleGetNavigationHistory(_ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(map[string]interface{}{
		"currentIndex": 0,
		"entries":      []interface{}{},
	})
}
