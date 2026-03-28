package cdp

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/coder/websocket"
)

// Session represents an active CDP client connection.
// It provides the ability to push events to the client.
type Session struct {
	conn *websocket.Conn
	ctx  context.Context
	mu   sync.Mutex
}

// newSession creates a session wrapping a WebSocket connection.
func newSession(ctx context.Context, conn *websocket.Conn) *Session {
	return &Session{conn: conn, ctx: ctx}
}

// SendEvent pushes a CDP event to the connected client.
func (s *Session) SendEvent(method string, params interface{}) error {
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}
	evt := Event{Method: method, Params: data}
	evtData, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn.Write(s.ctx, websocket.MessageText, evtData)
}

// WriteJSON serializes v as JSON and writes it to the WebSocket connection.
// It is safe for concurrent use.
func (s *Session) WriteJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn.Write(s.ctx, websocket.MessageText, data)
}

// SessionHandler processes a CDP method call with access to the session.
// It returns a result, optional post-response events, and an error.
type SessionHandler func(session *Session, params json.RawMessage) (json.RawMessage, []Event, error)
