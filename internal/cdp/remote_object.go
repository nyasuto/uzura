package cdp

import (
	"fmt"
	"sync"
)

// RemoteObject is the CDP Runtime.RemoteObject structure.
type RemoteObject struct {
	Type        string      `json:"type"`
	Subtype     string      `json:"subtype,omitempty"`
	ClassName   string      `json:"className,omitempty"`
	Value       interface{} `json:"value,omitempty"`
	Description string      `json:"description,omitempty"`
	ObjectID    string      `json:"objectId,omitempty"`
}

// ObjectStore manages remote object references for a CDP session.
type ObjectStore struct {
	mu     sync.Mutex
	nextID int
	byID   map[string]interface{}
}

// NewObjectStore creates an empty ObjectStore.
func NewObjectStore() *ObjectStore {
	return &ObjectStore{
		nextID: 1,
		byID:   make(map[string]interface{}),
	}
}

// Store saves a value and returns its objectId.
func (s *ObjectStore) Store(v interface{}) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := fmt.Sprintf("obj-%d", s.nextID)
	s.nextID++
	s.byID[id] = v
	return id
}

// Get retrieves a stored value by objectId.
func (s *ObjectStore) Get(id string) (interface{}, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.byID[id]
	return v, ok
}

// SerializeValue converts a Go value to a CDP RemoteObject.
// For complex types (objects, arrays), it stores them and assigns objectIds.
func (s *ObjectStore) SerializeValue(v interface{}) RemoteObject {
	if v == nil {
		return RemoteObject{Type: "undefined"}
	}

	switch val := v.(type) {
	case bool:
		return RemoteObject{Type: "boolean", Value: val}
	case string:
		return RemoteObject{Type: "string", Value: val}
	case int64:
		return RemoteObject{Type: "number", Value: val, Description: fmt.Sprintf("%d", val)}
	case float64:
		return RemoteObject{Type: "number", Value: val, Description: fmt.Sprintf("%v", val)}
	case int:
		return RemoteObject{Type: "number", Value: val, Description: fmt.Sprintf("%d", val)}
	case []interface{}:
		id := s.Store(val)
		return RemoteObject{
			Type:        "object",
			Subtype:     "array",
			ClassName:   "Array",
			Description: fmt.Sprintf("Array(%d)", len(val)),
			ObjectID:    id,
		}
	case map[string]interface{}:
		id := s.Store(val)
		return RemoteObject{
			Type:        "object",
			ClassName:   "Object",
			Description: "Object",
			ObjectID:    id,
		}
	default:
		// For other types, serialize as string description.
		desc := fmt.Sprintf("%v", val)
		return RemoteObject{Type: "object", Description: desc}
	}
}

// SerializeForConsole converts a list of Go values to RemoteObjects for console events.
func (s *ObjectStore) SerializeForConsole(args []interface{}) []RemoteObject {
	result := make([]RemoteObject, len(args))
	for i, arg := range args {
		result[i] = s.SerializeValue(arg)
	}
	return result
}
