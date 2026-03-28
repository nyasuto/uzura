package cdp

import "sync"

// handlerScope holds per-session handler overrides for session multiplexing.
// When a CDP request arrives with a sessionId that maps to a scope,
// the scope's handlers take precedence over global handlers.
type handlerScope struct {
	mu              sync.RWMutex
	handlers        map[string]Handler
	sessionHandlers map[string]SessionHandler
}

func newHandlerScope() *handlerScope {
	return &handlerScope{
		handlers:        make(map[string]Handler),
		sessionHandlers: make(map[string]SessionHandler),
	}
}

// Handle registers a stateless handler in this scope.
func (sc *handlerScope) Handle(method string, h Handler) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.handlers[method] = h
}

// HandleSession registers a session-aware handler in this scope.
func (sc *handlerScope) HandleSession(method string, h SessionHandler) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.sessionHandlers[method] = h
}

// lookup finds a handler for the given method in this scope.
// Returns the handler type and whether it was found.
func (sc *handlerScope) lookup(method string) (Handler, SessionHandler, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	if sh, ok := sc.sessionHandlers[method]; ok {
		return nil, sh, true
	}
	if h, ok := sc.handlers[method]; ok {
		return h, nil, true
	}
	return nil, nil, false
}
