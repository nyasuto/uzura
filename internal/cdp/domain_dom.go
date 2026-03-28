package cdp

import (
	"encoding/json"
	"fmt"

	"github.com/nyasuto/uzura/internal/dom"
	"github.com/nyasuto/uzura/internal/page"
)

// DOMDomain handles CDP DOM domain methods.
type DOMDomain struct {
	page  *page.Page
	store *NodeStore
}

// NewDOMDomain creates a DOMDomain wrapping the given page.
func NewDOMDomain(p *page.Page) *DOMDomain {
	return &DOMDomain{page: p, store: NewNodeStore()}
}

// Register adds DOM domain handlers to the server.
func (d *DOMDomain) Register(s *Server) {
	s.HandleSession("DOM.enable", d.enable)
	s.HandleSession("DOM.getDocument", d.getDocument)
	s.HandleSession("DOM.querySelector", d.querySelector)
	s.HandleSession("DOM.querySelectorAll", d.querySelectorAll)
	s.HandleSession("DOM.getOuterHTML", d.getOuterHTML)
	s.HandleSession("DOM.setOuterHTML", d.setOuterHTML)
	s.HandleSession("DOM.getAttributes", d.getAttributes)
	s.HandleSession("DOM.setAttributeValue", d.setAttributeValue)
	s.HandleSession("DOM.removeAttribute", d.removeAttribute)
	s.HandleSession("DOM.requestChildNodes", d.requestChildNodes)
}

// Store returns the NodeStore for external access (e.g., after navigation reset).
func (d *DOMDomain) Store() *NodeStore {
	return d.store
}

func (d *DOMDomain) enable(_ *Session, _ json.RawMessage) (json.RawMessage, []Event, error) {
	r, err := json.Marshal(struct{}{})
	return r, nil, err
}

func (d *DOMDomain) getDocument(_ *Session, params json.RawMessage) (json.RawMessage, []Event, error) {
	doc := d.page.Document()
	if doc == nil {
		return nil, nil, fmt.Errorf("no document loaded")
	}

	var p struct {
		Depth *int `json:"depth"`
	}
	if params != nil {
		_ = json.Unmarshal(params, &p)
	}
	depth := 2 // CDP default
	if p.Depth != nil {
		depth = *p.Depth
	}

	d.store.Reset()
	node := d.serializeNode(doc, depth)
	r, err := json.Marshal(map[string]interface{}{"root": node})
	return r, nil, err
}

func (d *DOMDomain) querySelector(_ *Session, params json.RawMessage) (json.RawMessage, []Event, error) {
	var p struct {
		NodeID   int    `json:"nodeId"`
		Selector string `json:"selector"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, nil, fmt.Errorf("invalid params: %w", err)
	}

	n := d.store.Lookup(p.NodeID)
	if n == nil {
		return nil, nil, fmt.Errorf("node not found: %d", p.NodeID)
	}

	var found *dom.Element
	var qerr error
	switch v := n.(type) {
	case *dom.Document:
		found, qerr = v.QuerySelector(p.Selector)
	case *dom.Element:
		found, qerr = v.QuerySelector(p.Selector)
	default:
		return nil, nil, fmt.Errorf("node %d does not support querySelector", p.NodeID)
	}
	if qerr != nil {
		return nil, nil, fmt.Errorf("querySelector: %w", qerr)
	}

	nodeID := 0
	if found != nil {
		nodeID = d.store.Bind(found)
	}
	r, err := json.Marshal(map[string]int{"nodeId": nodeID})
	return r, nil, err
}

func (d *DOMDomain) querySelectorAll(_ *Session, params json.RawMessage) (json.RawMessage, []Event, error) {
	var p struct {
		NodeID   int    `json:"nodeId"`
		Selector string `json:"selector"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, nil, fmt.Errorf("invalid params: %w", err)
	}

	n := d.store.Lookup(p.NodeID)
	if n == nil {
		return nil, nil, fmt.Errorf("node not found: %d", p.NodeID)
	}

	var elems []*dom.Element
	var qerr error
	switch v := n.(type) {
	case *dom.Document:
		elems, qerr = v.QuerySelectorAll(p.Selector)
	case *dom.Element:
		elems, qerr = v.QuerySelectorAll(p.Selector)
	default:
		return nil, nil, fmt.Errorf("node %d does not support querySelectorAll", p.NodeID)
	}
	if qerr != nil {
		return nil, nil, fmt.Errorf("querySelectorAll: %w", qerr)
	}

	ids := make([]int, len(elems))
	for i, e := range elems {
		ids[i] = d.store.Bind(e)
	}
	r, err := json.Marshal(map[string]interface{}{"nodeIds": ids})
	return r, nil, err
}

func (d *DOMDomain) getOuterHTML(_ *Session, params json.RawMessage) (json.RawMessage, []Event, error) {
	var p struct {
		NodeID int `json:"nodeId"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, nil, fmt.Errorf("invalid params: %w", err)
	}

	n := d.store.Lookup(p.NodeID)
	if n == nil {
		return nil, nil, fmt.Errorf("node not found: %d", p.NodeID)
	}

	html := dom.OuterHTML(n)
	r, err := json.Marshal(map[string]string{"outerHTML": html})
	return r, nil, err
}

func (d *DOMDomain) setOuterHTML(_ *Session, params json.RawMessage) (json.RawMessage, []Event, error) {
	var p struct {
		NodeID    int    `json:"nodeId"`
		OuterHTML string `json:"outerHTML"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, nil, fmt.Errorf("invalid params: %w", err)
	}

	n := d.store.Lookup(p.NodeID)
	if n == nil {
		return nil, nil, fmt.Errorf("node not found: %d", p.NodeID)
	}

	elem, ok := n.(*dom.Element)
	if !ok {
		return nil, nil, fmt.Errorf("setOuterHTML only supported on elements")
	}

	parent := elem.ParentNode()
	if parent == nil {
		return nil, nil, fmt.Errorf("cannot replace root element")
	}

	if err := elem.SetInnerHTML(p.OuterHTML); err != nil {
		return nil, nil, fmt.Errorf("setOuterHTML: %w", err)
	}
	r, err := json.Marshal(struct{}{})
	return r, nil, err
}

func (d *DOMDomain) getAttributes(_ *Session, params json.RawMessage) (json.RawMessage, []Event, error) {
	var p struct {
		NodeID int `json:"nodeId"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, nil, fmt.Errorf("invalid params: %w", err)
	}

	n := d.store.Lookup(p.NodeID)
	if n == nil {
		return nil, nil, fmt.Errorf("node not found: %d", p.NodeID)
	}

	elem, ok := n.(*dom.Element)
	if !ok {
		return nil, nil, fmt.Errorf("node %d is not an element", p.NodeID)
	}

	attrs := elem.Attributes()
	flat := make([]string, 0, len(attrs)*2)
	for _, a := range attrs {
		flat = append(flat, a.Key, a.Val)
	}
	r, err := json.Marshal(map[string]interface{}{"attributes": flat})
	return r, nil, err
}

func (d *DOMDomain) setAttributeValue(_ *Session, params json.RawMessage) (json.RawMessage, []Event, error) {
	var p struct {
		NodeID int    `json:"nodeId"`
		Name   string `json:"name"`
		Value  string `json:"value"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, nil, fmt.Errorf("invalid params: %w", err)
	}

	n := d.store.Lookup(p.NodeID)
	if n == nil {
		return nil, nil, fmt.Errorf("node not found: %d", p.NodeID)
	}

	elem, ok := n.(*dom.Element)
	if !ok {
		return nil, nil, fmt.Errorf("node %d is not an element", p.NodeID)
	}

	elem.SetAttribute(p.Name, p.Value)
	r, err := json.Marshal(struct{}{})
	return r, nil, err
}

func (d *DOMDomain) removeAttribute(_ *Session, params json.RawMessage) (json.RawMessage, []Event, error) {
	var p struct {
		NodeID int    `json:"nodeId"`
		Name   string `json:"name"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, nil, fmt.Errorf("invalid params: %w", err)
	}

	n := d.store.Lookup(p.NodeID)
	if n == nil {
		return nil, nil, fmt.Errorf("node not found: %d", p.NodeID)
	}

	elem, ok := n.(*dom.Element)
	if !ok {
		return nil, nil, fmt.Errorf("node %d is not an element", p.NodeID)
	}

	elem.RemoveAttribute(p.Name)
	r, err := json.Marshal(struct{}{})
	return r, nil, err
}

func (d *DOMDomain) requestChildNodes(sess *Session, params json.RawMessage) (json.RawMessage, []Event, error) {
	var p struct {
		NodeID int  `json:"nodeId"`
		Depth  *int `json:"depth"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, nil, fmt.Errorf("invalid params: %w", err)
	}

	n := d.store.Lookup(p.NodeID)
	if n == nil {
		return nil, nil, fmt.Errorf("node not found: %d", p.NodeID)
	}

	depth := 1
	if p.Depth != nil {
		depth = *p.Depth
	}

	children := d.serializeChildren(n, depth)
	evtData, _ := json.Marshal(map[string]interface{}{
		"parentId": p.NodeID,
		"nodes":    children,
	})

	events := []Event{
		{Method: "DOM.setChildNodes", Params: evtData},
	}

	r, _ := json.Marshal(struct{}{})
	return r, events, nil
}

// serializeNode converts a DOM node to the CDP Node JSON representation.
func (d *DOMDomain) serializeNode(n dom.Node, depth int) map[string]interface{} {
	nodeID := d.store.Bind(n)
	result := map[string]interface{}{
		"nodeId":        nodeID,
		"backendNodeId": nodeID,
		"nodeType":      int(n.NodeType()),
		"nodeName":      n.NodeName(),
		"localName":     localName(n),
		"nodeValue":     nodeValue(n),
	}

	childCount := countChildren(n)
	result["childNodeCount"] = childCount

	if elem, ok := n.(*dom.Element); ok {
		attrs := elem.Attributes()
		flat := make([]string, 0, len(attrs)*2)
		for _, a := range attrs {
			flat = append(flat, a.Key, a.Val)
		}
		result["attributes"] = flat
	}

	if depth > 0 && childCount > 0 {
		result["children"] = d.serializeChildren(n, depth)
	}

	return result
}

func (d *DOMDomain) serializeChildren(n dom.Node, depth int) []map[string]interface{} {
	var children []map[string]interface{}
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		children = append(children, d.serializeNode(c, depth-1))
	}
	return children
}

func localName(n dom.Node) string {
	if elem, ok := n.(*dom.Element); ok {
		return elem.LocalName()
	}
	return ""
}

func nodeValue(n dom.Node) string {
	switch v := n.(type) {
	case *dom.Text:
		return v.Data
	case *dom.Comment:
		return v.Data
	default:
		return ""
	}
}

func countChildren(n dom.Node) int {
	count := 0
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		count++
	}
	return count
}
