package js

import "github.com/dop251/goja"

// Storage is the Go-side contract behind localStorage/sessionStorage. It is
// exported so later tasks (e.g. persistence backends) can supply their own
// implementation in place of NewMemStorage.
type Storage interface {
	// GetItem returns the value stored under key, and false if key is unset.
	GetItem(key string) (string, bool)
	// SetItem stores value under key, preserving key's original insertion
	// position if it already exists.
	SetItem(key, value string)
	// RemoveItem deletes key. It is a no-op if key does not exist.
	RemoveItem(key string)
	// Clear removes all keys.
	Clear()
	// Key returns the key at insertion-order index n, and false if n is out
	// of range.
	Key(n int) (string, bool)
	// Len returns the number of stored keys.
	Len() int
}

// memStorage is an in-memory Storage implementation that preserves
// insertion order for Key(n) iteration, matching the Web Storage spec.
type memStorage struct {
	data map[string]string
	keys []string
}

// NewMemStorage returns a Storage backed by an in-memory map. Insertion
// order is preserved: overwriting an existing key does not change its
// position, matching real localStorage/sessionStorage behavior.
func NewMemStorage() Storage {
	return &memStorage{data: make(map[string]string)}
}

func (m *memStorage) GetItem(key string) (string, bool) {
	v, ok := m.data[key]
	return v, ok
}

func (m *memStorage) SetItem(key, value string) {
	if _, exists := m.data[key]; !exists {
		m.keys = append(m.keys, key)
	}
	m.data[key] = value
}

func (m *memStorage) RemoveItem(key string) {
	if _, exists := m.data[key]; !exists {
		return
	}
	delete(m.data, key)
	for i, k := range m.keys {
		if k == key {
			m.keys = append(m.keys[:i], m.keys[i+1:]...)
			break
		}
	}
}

func (m *memStorage) Clear() {
	m.data = make(map[string]string)
	m.keys = nil
}

func (m *memStorage) Key(n int) (string, bool) {
	if n < 0 || n >= len(m.keys) {
		return "", false
	}
	return m.keys[n], true
}

func (m *memStorage) Len() int {
	return len(m.keys)
}

// storageObject implements goja.DynamicObject, exposing a Storage value to
// JS with both method-call (getItem/setItem/...) and bare property
// (localStorage.foo) access.
//
// Method lookups and the "length" property share the same Get(key) hook as
// data keys: a data key named "getItem" (etc.) is shadowed by the method —
// sites virtually never store keys with those names, so this is an
// acceptable trade-off for a simpler implementation.
type storageObject struct {
	backing Storage
	rt      *goja.Runtime
	methods map[string]goja.Value
}

// newStorageObject builds a storageObject and pre-builds its method table
// once, so repeated Get("getItem") calls return the same cached goja
// function value rather than allocating a new one each time.
func newStorageObject(rt *goja.Runtime, backing Storage) *storageObject {
	s := &storageObject{backing: backing, rt: rt}
	s.methods = map[string]goja.Value{
		"getItem": rt.ToValue(func(call goja.FunctionCall) goja.Value {
			key := call.Argument(0).String()
			v, ok := s.backing.GetItem(key)
			if !ok {
				return goja.Null()
			}
			return rt.ToValue(v)
		}),
		"setItem": rt.ToValue(func(call goja.FunctionCall) goja.Value {
			s.backing.SetItem(call.Argument(0).String(), call.Argument(1).String())
			return goja.Undefined()
		}),
		"removeItem": rt.ToValue(func(call goja.FunctionCall) goja.Value {
			s.backing.RemoveItem(call.Argument(0).String())
			return goja.Undefined()
		}),
		"clear": rt.ToValue(func(call goja.FunctionCall) goja.Value {
			s.backing.Clear()
			return goja.Undefined()
		}),
		"key": rt.ToValue(func(call goja.FunctionCall) goja.Value {
			n := int(call.Argument(0).ToInteger())
			k, ok := s.backing.Key(n)
			if !ok {
				return goja.Null()
			}
			return rt.ToValue(k)
		}),
	}
	return s
}

// Get implements goja.DynamicObject. Method names and "length" take
// priority over stored data keys (see storageObject doc comment).
func (s *storageObject) Get(key string) goja.Value {
	if m, ok := s.methods[key]; ok {
		return m
	}
	if key == "length" {
		return s.rt.ToValue(s.backing.Len())
	}
	v, ok := s.backing.GetItem(key)
	if !ok {
		return goja.Undefined()
	}
	return s.rt.ToValue(v)
}

// Set implements goja.DynamicObject, storing val's string form under key.
func (s *storageObject) Set(key string, val goja.Value) bool {
	s.backing.SetItem(key, val.String())
	return true
}

// Has implements goja.DynamicObject.
func (s *storageObject) Has(key string) bool {
	if _, ok := s.methods[key]; ok {
		return true
	}
	if key == "length" {
		return true
	}
	_, ok := s.backing.GetItem(key)
	return ok
}

// Delete implements goja.DynamicObject.
func (s *storageObject) Delete(key string) bool {
	s.backing.RemoveItem(key)
	return true
}

// Keys implements goja.DynamicObject, returning only data keys (not method
// names or "length"), matching how real Storage objects enumerate.
func (s *storageObject) Keys() []string {
	keys := make([]string, s.backing.Len())
	for i := range keys {
		k, _ := s.backing.Key(i)
		keys[i] = k
	}
	return keys
}

// BindStorage registers localStorage and sessionStorage as globals backed
// by the given Storage values, each independently accessible via both
// method calls (getItem/setItem/...) and bare property access.
func BindStorage(vm *VM, local, session Storage) {
	rt := vm.runtime
	_ = rt.Set("localStorage", rt.NewDynamicObject(newStorageObject(rt, local)))
	_ = rt.Set("sessionStorage", rt.NewDynamicObject(newStorageObject(rt, session)))
}
