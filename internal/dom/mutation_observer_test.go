package dom

import "testing"

func TestMutationObserver_ChildList_AppendChild(t *testing.T) {
	doc := NewDocument()
	parent := doc.CreateElement("div")
	doc.AppendChild(parent)

	var received []*MutationRecord
	observer := NewMutationObserver(func(records []*MutationRecord, o *MutationObserver) {
		received = records
	})
	observer.Observe(parent, MutationObserverInit{ChildList: true})

	child := doc.CreateElement("span")
	parent.AppendChild(child)
	FlushMutationObservers()

	if len(received) != 1 {
		t.Fatalf("expected 1 record, got %d", len(received))
	}
	r := received[0]
	if r.Type != "childList" {
		t.Errorf("type: got %q, want %q", r.Type, "childList")
	}
	if r.Target != Node(parent) {
		t.Error("target should be parent")
	}
	if len(r.AddedNodes) != 1 || r.AddedNodes[0] != Node(child) {
		t.Error("addedNodes should contain the child")
	}
	if len(r.RemovedNodes) != 0 {
		t.Error("removedNodes should be empty")
	}
}

func TestMutationObserver_ChildList_RemoveChild(t *testing.T) {
	doc := NewDocument()
	parent := doc.CreateElement("div")
	child := doc.CreateElement("span")
	parent.AppendChild(child)

	var received []*MutationRecord
	observer := NewMutationObserver(func(records []*MutationRecord, o *MutationObserver) {
		received = records
	})
	observer.Observe(parent, MutationObserverInit{ChildList: true})

	parent.RemoveChild(child)
	FlushMutationObservers()

	if len(received) != 1 {
		t.Fatalf("expected 1 record, got %d", len(received))
	}
	r := received[0]
	if r.Type != "childList" {
		t.Errorf("type: got %q, want %q", r.Type, "childList")
	}
	if len(r.RemovedNodes) != 1 || r.RemovedNodes[0] != Node(child) {
		t.Error("removedNodes should contain the child")
	}
}

func TestMutationObserver_ChildList_InsertBefore(t *testing.T) {
	doc := NewDocument()
	parent := doc.CreateElement("div")
	existing := doc.CreateElement("span")
	parent.AppendChild(existing)

	var received []*MutationRecord
	observer := NewMutationObserver(func(records []*MutationRecord, o *MutationObserver) {
		received = records
	})
	observer.Observe(parent, MutationObserverInit{ChildList: true})

	newChild := doc.CreateElement("p")
	parent.InsertBefore(newChild, existing)
	FlushMutationObservers()

	if len(received) != 1 {
		t.Fatalf("expected 1 record, got %d", len(received))
	}
	r := received[0]
	if len(r.AddedNodes) != 1 || r.AddedNodes[0] != Node(newChild) {
		t.Error("addedNodes should contain newChild")
	}
	if r.NextSibling != Node(existing) {
		t.Error("nextSibling should be existing")
	}
}

func TestMutationObserver_ChildList_ReplaceChild(t *testing.T) {
	doc := NewDocument()
	parent := doc.CreateElement("div")
	oldChild := doc.CreateElement("span")
	parent.AppendChild(oldChild)

	var received []*MutationRecord
	observer := NewMutationObserver(func(records []*MutationRecord, o *MutationObserver) {
		received = records
	})
	observer.Observe(parent, MutationObserverInit{ChildList: true})

	newChild := doc.CreateElement("p")
	parent.ReplaceChild(newChild, oldChild)
	FlushMutationObservers()

	if len(received) != 1 {
		t.Fatalf("expected 1 record, got %d", len(received))
	}
	r := received[0]
	if len(r.AddedNodes) != 1 || r.AddedNodes[0] != Node(newChild) {
		t.Error("addedNodes should contain newChild")
	}
	if len(r.RemovedNodes) != 1 || r.RemovedNodes[0] != Node(oldChild) {
		t.Error("removedNodes should contain oldChild")
	}
}

func TestMutationObserver_Subtree(t *testing.T) {
	doc := NewDocument()
	root := doc.CreateElement("div")
	child := doc.CreateElement("span")
	root.AppendChild(child)

	var received []*MutationRecord
	observer := NewMutationObserver(func(records []*MutationRecord, o *MutationObserver) {
		received = records
	})
	observer.Observe(root, MutationObserverInit{ChildList: true, Subtree: true})

	grandchild := doc.CreateElement("a")
	child.AppendChild(grandchild)
	FlushMutationObservers()

	if len(received) != 1 {
		t.Fatalf("expected 1 record, got %d", len(received))
	}
	if received[0].Target != Node(child) {
		t.Error("target should be child (where mutation happened)")
	}
}

func TestMutationObserver_SubtreeNotEnabled(t *testing.T) {
	doc := NewDocument()
	root := doc.CreateElement("div")
	child := doc.CreateElement("span")
	root.AppendChild(child)

	var received []*MutationRecord
	observer := NewMutationObserver(func(records []*MutationRecord, o *MutationObserver) {
		received = records
	})
	observer.Observe(root, MutationObserverInit{ChildList: true}) // no Subtree

	child.AppendChild(doc.CreateElement("a"))
	FlushMutationObservers()

	if len(received) != 0 {
		t.Errorf("should not receive records without Subtree, got %d", len(received))
	}
}

func TestMutationObserver_Attributes(t *testing.T) {
	doc := NewDocument()
	el := doc.CreateElement("div")

	var received []*MutationRecord
	observer := NewMutationObserver(func(records []*MutationRecord, o *MutationObserver) {
		received = records
	})
	observer.Observe(el, MutationObserverInit{Attributes: true})

	el.SetAttribute("class", "active")
	FlushMutationObservers()

	if len(received) != 1 {
		t.Fatalf("expected 1 record, got %d", len(received))
	}
	r := received[0]
	if r.Type != "attributes" {
		t.Errorf("type: got %q, want %q", r.Type, "attributes")
	}
	if r.AttributeName != "class" {
		t.Errorf("attributeName: got %q, want %q", r.AttributeName, "class")
	}
}

func TestMutationObserver_AttributeOldValue(t *testing.T) {
	doc := NewDocument()
	el := doc.CreateElement("div")
	el.SetAttribute("class", "old")

	var received []*MutationRecord
	observer := NewMutationObserver(func(records []*MutationRecord, o *MutationObserver) {
		received = records
	})
	observer.Observe(el, MutationObserverInit{Attributes: true, AttributeOldValue: true})

	el.SetAttribute("class", "new")
	FlushMutationObservers()

	if len(received) != 1 {
		t.Fatalf("expected 1 record, got %d", len(received))
	}
	if received[0].OldValue != "old" {
		t.Errorf("oldValue: got %q, want %q", received[0].OldValue, "old")
	}
}

func TestMutationObserver_AttributeFilter(t *testing.T) {
	doc := NewDocument()
	el := doc.CreateElement("div")

	var received []*MutationRecord
	observer := NewMutationObserver(func(records []*MutationRecord, o *MutationObserver) {
		received = records
	})
	observer.Observe(el, MutationObserverInit{
		Attributes:      true,
		AttributeFilter: []string{"class"},
	})

	el.SetAttribute("id", "test")   // should NOT be observed
	el.SetAttribute("class", "foo") // should be observed
	FlushMutationObservers()

	if len(received) != 1 {
		t.Fatalf("expected 1 record, got %d", len(received))
	}
	if received[0].AttributeName != "class" {
		t.Errorf("attributeName: got %q, want %q", received[0].AttributeName, "class")
	}
}

func TestMutationObserver_RemoveAttribute(t *testing.T) {
	doc := NewDocument()
	el := doc.CreateElement("div")
	el.SetAttribute("class", "active")

	var received []*MutationRecord
	observer := NewMutationObserver(func(records []*MutationRecord, o *MutationObserver) {
		received = records
	})
	observer.Observe(el, MutationObserverInit{Attributes: true, AttributeOldValue: true})

	el.RemoveAttribute("class")
	FlushMutationObservers()

	if len(received) != 1 {
		t.Fatalf("expected 1 record, got %d", len(received))
	}
	if received[0].OldValue != "active" {
		t.Errorf("oldValue: got %q, want %q", received[0].OldValue, "active")
	}
}

func TestMutationObserver_CharacterData(t *testing.T) {
	doc := NewDocument()
	text := doc.CreateTextNode("hello")
	parent := doc.CreateElement("div")
	parent.AppendChild(text)

	var received []*MutationRecord
	observer := NewMutationObserver(func(records []*MutationRecord, o *MutationObserver) {
		received = records
	})
	observer.Observe(text, MutationObserverInit{CharacterData: true})

	text.SetTextContent("world")
	FlushMutationObservers()

	if len(received) != 1 {
		t.Fatalf("expected 1 record, got %d", len(received))
	}
	r := received[0]
	if r.Type != "characterData" {
		t.Errorf("type: got %q, want %q", r.Type, "characterData")
	}
	if r.Target != Node(text) {
		t.Error("target should be text node")
	}
}

func TestMutationObserver_CharacterDataOldValue(t *testing.T) {
	doc := NewDocument()
	text := doc.CreateTextNode("hello")

	var received []*MutationRecord
	observer := NewMutationObserver(func(records []*MutationRecord, o *MutationObserver) {
		received = records
	})
	observer.Observe(text, MutationObserverInit{CharacterData: true, CharacterDataOldValue: true})

	text.SetTextContent("world")
	FlushMutationObservers()

	if len(received) != 1 {
		t.Fatalf("expected 1 record, got %d", len(received))
	}
	if received[0].OldValue != "hello" {
		t.Errorf("oldValue: got %q, want %q", received[0].OldValue, "hello")
	}
}

func TestMutationObserver_CharacterDataComment(t *testing.T) {
	doc := NewDocument()
	comment := doc.CreateComment("old")

	var received []*MutationRecord
	observer := NewMutationObserver(func(records []*MutationRecord, o *MutationObserver) {
		received = records
	})
	observer.Observe(comment, MutationObserverInit{CharacterData: true, CharacterDataOldValue: true})

	comment.SetTextContent("new")
	FlushMutationObservers()

	if len(received) != 1 {
		t.Fatalf("expected 1 record, got %d", len(received))
	}
	if received[0].OldValue != "old" {
		t.Errorf("oldValue: got %q, want %q", received[0].OldValue, "old")
	}
}

func TestMutationObserver_Disconnect(t *testing.T) {
	doc := NewDocument()
	el := doc.CreateElement("div")

	var received []*MutationRecord
	observer := NewMutationObserver(func(records []*MutationRecord, o *MutationObserver) {
		received = records
	})
	observer.Observe(el, MutationObserverInit{ChildList: true})
	observer.Disconnect()

	el.AppendChild(doc.CreateElement("span"))
	FlushMutationObservers()

	if len(received) != 0 {
		t.Errorf("should not receive records after disconnect, got %d", len(received))
	}
}

func TestMutationObserver_TakeRecords(t *testing.T) {
	doc := NewDocument()
	el := doc.CreateElement("div")

	observer := NewMutationObserver(func(records []*MutationRecord, o *MutationObserver) {
		t.Error("callback should not be called after TakeRecords")
	})
	observer.Observe(el, MutationObserverInit{ChildList: true})

	el.AppendChild(doc.CreateElement("span"))
	el.AppendChild(doc.CreateElement("p"))

	records := observer.TakeRecords()
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	// Flush should not call callback since records were taken
	FlushMutationObservers()
}

func TestMutationObserver_MultipleObservers(t *testing.T) {
	doc := NewDocument()
	el := doc.CreateElement("div")

	var received1, received2 []*MutationRecord
	o1 := NewMutationObserver(func(records []*MutationRecord, o *MutationObserver) {
		received1 = records
	})
	o2 := NewMutationObserver(func(records []*MutationRecord, o *MutationObserver) {
		received2 = records
	})

	o1.Observe(el, MutationObserverInit{ChildList: true})
	o2.Observe(el, MutationObserverInit{Attributes: true})

	el.AppendChild(doc.CreateElement("span"))
	el.SetAttribute("class", "test")
	FlushMutationObservers()

	if len(received1) != 1 {
		t.Errorf("observer1 should get 1 childList record, got %d", len(received1))
	}
	if len(received2) != 1 {
		t.Errorf("observer2 should get 1 attribute record, got %d", len(received2))
	}
}

func TestMutationObserver_PreviousNextSibling(t *testing.T) {
	doc := NewDocument()
	parent := doc.CreateElement("div")
	first := doc.CreateElement("span")
	last := doc.CreateElement("a")
	parent.AppendChild(first)
	parent.AppendChild(last)

	var received []*MutationRecord
	observer := NewMutationObserver(func(records []*MutationRecord, o *MutationObserver) {
		received = records
	})
	observer.Observe(parent, MutationObserverInit{ChildList: true})

	middle := doc.CreateElement("p")
	parent.InsertBefore(middle, last)
	FlushMutationObservers()

	if len(received) != 1 {
		t.Fatalf("expected 1 record, got %d", len(received))
	}
	r := received[0]
	if r.PreviousSibling != Node(first) {
		t.Error("previousSibling should be first")
	}
	if r.NextSibling != Node(last) {
		t.Error("nextSibling should be last")
	}
}

func TestMutationObserver_NoDuplicateRecords(t *testing.T) {
	doc := NewDocument()
	root := doc.CreateElement("div")
	child := doc.CreateElement("span")
	root.AppendChild(child)

	var received []*MutationRecord
	observer := NewMutationObserver(func(records []*MutationRecord, o *MutationObserver) {
		received = records
	})
	// Observe both root (with subtree) and child directly
	observer.Observe(root, MutationObserverInit{ChildList: true, Subtree: true})
	observer.Observe(child, MutationObserverInit{ChildList: true})

	child.AppendChild(doc.CreateElement("a"))
	FlushMutationObservers()

	// Should only get 1 record, not 2 (dedup by observer)
	if len(received) != 1 {
		t.Errorf("expected 1 record (deduped), got %d", len(received))
	}
}

func TestMutationObserver_NoChildListNotification(t *testing.T) {
	doc := NewDocument()
	el := doc.CreateElement("div")

	var received []*MutationRecord
	observer := NewMutationObserver(func(records []*MutationRecord, o *MutationObserver) {
		received = records
	})
	observer.Observe(el, MutationObserverInit{Attributes: true}) // only attributes

	el.AppendChild(doc.CreateElement("span"))
	FlushMutationObservers()

	if len(received) != 0 {
		t.Errorf("should not receive childList records, got %d", len(received))
	}
}
