package cdp

import (
	"sync"

	"github.com/nyasuto/uzura/internal/dom"
)

// NodeStore manages the bidirectional mapping between CDP nodeIds and DOM nodes.
// CDP uses integer nodeIds to reference DOM nodes over the protocol.
type NodeStore struct {
	mu       sync.Mutex
	nextID   int
	nodeToID map[dom.Node]int
	idToNode map[int]dom.Node
}

// NewNodeStore creates an empty NodeStore with IDs starting at 1.
func NewNodeStore() *NodeStore {
	return &NodeStore{
		nextID:   1,
		nodeToID: make(map[dom.Node]int),
		idToNode: make(map[int]dom.Node),
	}
}

// Bind assigns a nodeId to a DOM node. If the node is already bound, the
// existing ID is returned.
func (s *NodeStore) Bind(n dom.Node) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, ok := s.nodeToID[n]; ok {
		return id
	}
	id := s.nextID
	s.nextID++
	s.nodeToID[n] = id
	s.idToNode[id] = n
	return id
}

// Lookup returns the DOM node for the given nodeId, or nil if not found.
func (s *NodeStore) Lookup(id int) dom.Node {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.idToNode[id]
}

// IDOf returns the nodeId for the given DOM node, or 0 if not bound.
func (s *NodeStore) IDOf(n dom.Node) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.nodeToID[n]
}

// Reset clears all bindings and resets the ID counter.
func (s *NodeStore) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID = 1
	s.nodeToID = make(map[dom.Node]int)
	s.idToNode = make(map[int]dom.Node)
}
