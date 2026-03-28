package dom

import "strings"

// ClassList implements a DOMTokenList-like interface for CSS class manipulation.
type ClassList struct {
	element *Element
}

// newClassList creates a ClassList bound to the given element.
func newClassList(el *Element) *ClassList {
	return &ClassList{element: el}
}

func (cl *ClassList) tokens() []string {
	raw := cl.element.GetAttribute("class")
	if raw == "" {
		return nil
	}
	return strings.Fields(raw)
}

func (cl *ClassList) sync(tokens []string) {
	cl.element.SetAttribute("class", strings.Join(tokens, " "))
}

// Length returns the number of classes.
func (cl *ClassList) Length() int {
	return len(cl.tokens())
}

// Item returns the class at the given index, or empty string if out of range.
func (cl *ClassList) Item(index int) string {
	t := cl.tokens()
	if index < 0 || index >= len(t) {
		return ""
	}
	return t[index]
}

// Contains reports whether the class list contains the given token.
func (cl *ClassList) Contains(token string) bool {
	for _, t := range cl.tokens() {
		if t == token {
			return true
		}
	}
	return false
}

// Add adds one or more classes. Duplicates are ignored.
func (cl *ClassList) Add(tokens ...string) {
	current := cl.tokens()
	for _, token := range tokens {
		if !contains(current, token) {
			current = append(current, token)
		}
	}
	cl.sync(current)
}

// Remove removes one or more classes.
func (cl *ClassList) Remove(tokens ...string) {
	current := cl.tokens()
	result := current[:0]
	remove := make(map[string]bool, len(tokens))
	for _, t := range tokens {
		remove[t] = true
	}
	for _, t := range current {
		if !remove[t] {
			result = append(result, t)
		}
	}
	cl.sync(result)
}

// Toggle adds the class if absent, removes if present. Returns the resulting state.
func (cl *ClassList) Toggle(token string) bool {
	if cl.Contains(token) {
		cl.Remove(token)
		return false
	}
	cl.Add(token)
	return true
}

// ToggleForce adds or removes the class based on the force flag. Returns the resulting state.
func (cl *ClassList) ToggleForce(token string, force bool) bool {
	if force {
		cl.Add(token)
		return true
	}
	cl.Remove(token)
	return false
}

// Replace replaces oldToken with newToken. Returns true if oldToken was found.
func (cl *ClassList) Replace(oldToken, newToken string) bool {
	current := cl.tokens()
	for i, t := range current {
		if t == oldToken {
			current[i] = newToken
			cl.sync(current)
			return true
		}
	}
	return false
}

// Value returns the serialized class string.
func (cl *ClassList) Value() string {
	return strings.Join(cl.tokens(), " ")
}

func contains(s []string, v string) bool {
	for _, item := range s {
		if item == v {
			return true
		}
	}
	return false
}
