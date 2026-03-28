package dom

import (
	"strings"
	"unicode"
)

// Dataset provides access to data-* attributes on an element using camelCase keys.
type Dataset struct {
	element *Element
}

// newDataset creates a Dataset bound to the given element.
func newDataset(el *Element) *Dataset {
	return &Dataset{element: el}
}

// Get returns the value of the data attribute with the given camelCase name.
func (ds *Dataset) Get(name string) string {
	return ds.element.GetAttribute(camelToDataAttr(name))
}

// Set sets the value of the data attribute with the given camelCase name.
func (ds *Dataset) Set(name, value string) {
	ds.element.SetAttribute(camelToDataAttr(name), value)
}

// Has reports whether the element has a data attribute with the given camelCase name.
func (ds *Dataset) Has(name string) bool {
	return ds.element.HasAttribute(camelToDataAttr(name))
}

// Delete removes the data attribute with the given camelCase name.
func (ds *Dataset) Delete(name string) {
	ds.element.RemoveAttribute(camelToDataAttr(name))
}

// All returns a map of all data-* attributes with camelCase keys.
func (ds *Dataset) All() map[string]string {
	result := make(map[string]string)
	for _, attr := range ds.element.Attributes() {
		if strings.HasPrefix(attr.Key, "data-") {
			result[dataAttrToCamel(attr.Key)] = attr.Val
		}
	}
	return result
}

// camelToDataAttr converts a camelCase name to a data-kebab-case attribute name.
// e.g. "fooBar" → "data-foo-bar"
func camelToDataAttr(name string) string {
	var sb strings.Builder
	sb.WriteString("data-")
	for _, r := range name {
		if unicode.IsUpper(r) {
			sb.WriteByte('-')
			sb.WriteRune(unicode.ToLower(r))
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// dataAttrToCamel converts a data-kebab-case attribute name to camelCase.
// e.g. "data-foo-bar" → "fooBar"
func dataAttrToCamel(attr string) string {
	name := strings.TrimPrefix(attr, "data-")
	parts := strings.Split(name, "-")
	var sb strings.Builder
	for i, part := range parts {
		if i == 0 {
			sb.WriteString(part)
		} else if part != "" {
			sb.WriteRune(unicode.ToUpper(rune(part[0])))
			sb.WriteString(part[1:])
		}
	}
	return sb.String()
}
