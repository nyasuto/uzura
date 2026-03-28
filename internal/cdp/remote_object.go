package cdp

import (
	"fmt"
	"sync"

	"github.com/dop251/goja"
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
		// For other types (including DOM proxy objects), store and return objectId.
		id := s.Store(val)
		desc := fmt.Sprintf("%v", val)
		return RemoteObject{
			Type:        "object",
			ClassName:   "Object",
			Description: desc,
			ObjectID:    id,
		}
	}
}

// SerializeGojaValue converts a raw goja.Value to a CDP RemoteObject.
// It stores the original goja.Value (not exported) so callFunctionOn can use it.
func (s *ObjectStore) SerializeGojaValue(v goja.Value) RemoteObject {
	if v == nil || goja.IsUndefined(v) {
		return RemoteObject{Type: "undefined"}
	}
	if goja.IsNull(v) {
		return RemoteObject{Type: "object", Subtype: "null"}
	}

	exported := v.Export()

	switch val := exported.(type) {
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
	}

	// Complex type — store the raw goja.Value to preserve JS identity.
	id := s.Store(v)

	if obj, ok := v.(*goja.Object); ok {
		// Check if it's a DOM node by looking for nodeType property.
		nodeType := obj.Get("nodeType")
		if nodeType != nil && !goja.IsUndefined(nodeType) {
			desc := ""
			if tn := obj.Get("nodeName"); tn != nil && !goja.IsUndefined(tn) {
				desc = tn.String()
			}
			return RemoteObject{
				Type:        "object",
				Subtype:     "node",
				ClassName:   "Node",
				Description: desc,
				ObjectID:    id,
			}
		}

		// Check if it's an array.
		if arr, isArr := exported.([]interface{}); isArr {
			return RemoteObject{
				Type:        "object",
				Subtype:     "array",
				ClassName:   "Array",
				Description: fmt.Sprintf("Array(%d)", len(arr)),
				ObjectID:    id,
			}
		}

		// Generic object — use className from goja if available.
		className := obj.ClassName()
		return RemoteObject{
			Type:        "object",
			ClassName:   className,
			Description: className,
			ObjectID:    id,
		}
	}

	return RemoteObject{
		Type:        "object",
		ClassName:   "Object",
		Description: "Object",
		ObjectID:    id,
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
