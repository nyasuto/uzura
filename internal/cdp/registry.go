package cdp

// HandlerRegistry is the interface for registering CDP method handlers.
// Both Server and handlerScope implement this interface.
type HandlerRegistry interface {
	Handle(method string, h Handler)
	HandleSession(method string, h SessionHandler)
}
