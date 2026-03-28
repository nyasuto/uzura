package cdp

import (
	"encoding/json"
	"net/http"
)

// versionInfo is the response for /json/version.
type versionInfo struct {
	Browser         string `json:"Browser"`
	ProtocolVersion string `json:"Protocol-Version"`
	UserAgent       string `json:"User-Agent"`
	WebSocketURL    string `json:"webSocketDebuggerUrl"`
}

// targetInfo is the response for /json/list.
type targetInfo struct {
	Description          string `json:"description"`
	DevtoolsFrontendURL  string `json:"devtoolsFrontendUrl"`
	ID                   string `json:"id"`
	Title                string `json:"title"`
	Type                 string `json:"type"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

// protocolInfo is the response for /json/protocol.
type protocolInfo struct {
	Domains []domainInfo `json:"domains"`
}

type domainInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (s *Server) handleVersion(w http.ResponseWriter, _ *http.Request) {
	info := versionInfo{
		Browser:         s.browserVersion,
		ProtocolVersion: s.protocolVersion,
		UserAgent:       s.userAgent,
		WebSocketURL:    s.webSocketURL,
	}
	writeJSONHTTP(w, info)
}

func (s *Server) handleList(w http.ResponseWriter, _ *http.Request) {
	wsURL := s.webSocketURL
	targets := []targetInfo{
		{
			Description:          "",
			DevtoolsFrontendURL:  "",
			ID:                   "default",
			Title:                "",
			Type:                 "page",
			URL:                  "",
			WebSocketDebuggerURL: wsURL,
		},
	}
	writeJSONHTTP(w, targets)
}

func (s *Server) handleProtocol(w http.ResponseWriter, _ *http.Request) {
	info := protocolInfo{
		Domains: []domainInfo{
			{Name: "Page", Version: "1.3"},
			{Name: "DOM", Version: "1.3"},
			{Name: "Runtime", Version: "1.3"},
			{Name: "Network", Version: "1.3"},
			{Name: "Fetch", Version: "1.3"},
		},
	}
	writeJSONHTTP(w, info)
}

func writeJSONHTTP(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
