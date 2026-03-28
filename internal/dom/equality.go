package dom

// isEqualNode compares two nodes for structural equality per WHATWG spec.
func isEqualNode(a, b Node) bool {
	if b == nil {
		return false
	}
	if a.NodeType() != b.NodeType() {
		return false
	}

	switch na := a.(type) {
	case *Element:
		nb, ok := b.(*Element)
		if !ok {
			return false
		}
		if na.localName != nb.localName {
			return false
		}
		if len(na.attributes) != len(nb.attributes) {
			return false
		}
		for _, aa := range na.attributes {
			found := false
			for _, ba := range nb.attributes {
				if aa.Key == ba.Key && aa.Val == ba.Val && aa.Namespace == ba.Namespace {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	case *Text:
		nb, ok := b.(*Text)
		if !ok {
			return false
		}
		if na.Data != nb.Data {
			return false
		}
	case *Comment:
		nb, ok := b.(*Comment)
		if !ok {
			return false
		}
		if na.Data != nb.Data {
			return false
		}
	case *Document:
		if _, ok := b.(*Document); !ok {
			return false
		}
	case *DocumentFragment:
		if _, ok := b.(*DocumentFragment); !ok {
			return false
		}
	}

	// Compare children
	ac := a.ChildNodes()
	bc := b.ChildNodes()
	if len(ac) != len(bc) {
		return false
	}
	for i := range ac {
		if !ac[i].IsEqualNode(bc[i]) {
			return false
		}
	}
	return true
}
