package dom

import "testing"

func TestClassListBasic(t *testing.T) {
	el := NewElement("div")
	el.SetAttribute("class", "foo bar baz")
	cl := el.ClassList()

	if cl.Length() != 3 {
		t.Errorf("Length() = %d, want 3", cl.Length())
	}
	if cl.Item(0) != "foo" {
		t.Errorf("Item(0) = %q, want %q", cl.Item(0), "foo")
	}
	if cl.Item(2) != "baz" {
		t.Errorf("Item(2) = %q, want %q", cl.Item(2), "baz")
	}
	if cl.Item(5) != "" {
		t.Errorf("Item(5) = %q, want empty", cl.Item(5))
	}
}

func TestClassListContains(t *testing.T) {
	el := NewElement("div")
	el.SetAttribute("class", "foo bar")
	cl := el.ClassList()

	if !cl.Contains("foo") {
		t.Error("should contain foo")
	}
	if cl.Contains("baz") {
		t.Error("should not contain baz")
	}
}

func TestClassListAdd(t *testing.T) {
	el := NewElement("div")
	cl := el.ClassList()

	cl.Add("foo")
	cl.Add("bar")
	cl.Add("foo") // duplicate, should be ignored

	if cl.Length() != 2 {
		t.Errorf("Length() = %d, want 2", cl.Length())
	}
	if el.GetAttribute("class") != "foo bar" {
		t.Errorf("class attr = %q, want %q", el.GetAttribute("class"), "foo bar")
	}
}

func TestClassListAddMultiple(t *testing.T) {
	el := NewElement("div")
	cl := el.ClassList()

	cl.Add("a", "b", "c")
	if cl.Length() != 3 {
		t.Errorf("Length() = %d, want 3", cl.Length())
	}
}

func TestClassListRemove(t *testing.T) {
	el := NewElement("div")
	el.SetAttribute("class", "foo bar baz")
	cl := el.ClassList()

	cl.Remove("bar")
	if cl.Length() != 2 {
		t.Errorf("Length() = %d, want 2", cl.Length())
	}
	if cl.Contains("bar") {
		t.Error("should not contain bar after remove")
	}
	if el.GetAttribute("class") != "foo baz" {
		t.Errorf("class attr = %q, want %q", el.GetAttribute("class"), "foo baz")
	}
}

func TestClassListRemoveMultiple(t *testing.T) {
	el := NewElement("div")
	el.SetAttribute("class", "a b c d")
	cl := el.ClassList()

	cl.Remove("b", "d")
	if el.GetAttribute("class") != "a c" {
		t.Errorf("class attr = %q, want %q", el.GetAttribute("class"), "a c")
	}
}

func TestClassListToggle(t *testing.T) {
	el := NewElement("div")
	el.SetAttribute("class", "foo")
	cl := el.ClassList()

	// Toggle off
	result := cl.Toggle("foo")
	if result {
		t.Error("Toggle existing class should return false")
	}
	if cl.Contains("foo") {
		t.Error("foo should be removed")
	}

	// Toggle on
	result = cl.Toggle("foo")
	if !result {
		t.Error("Toggle missing class should return true")
	}
	if !cl.Contains("foo") {
		t.Error("foo should be added")
	}
}

func TestClassListToggleForce(t *testing.T) {
	el := NewElement("div")
	cl := el.ClassList()

	// Force add
	result := cl.ToggleForce("foo", true)
	if !result || !cl.Contains("foo") {
		t.Error("ToggleForce(true) should add")
	}

	// Force add again (no-op)
	result = cl.ToggleForce("foo", true)
	if !result || cl.Length() != 1 {
		t.Error("ToggleForce(true) on existing should keep it")
	}

	// Force remove
	result = cl.ToggleForce("foo", false)
	if result || cl.Contains("foo") {
		t.Error("ToggleForce(false) should remove")
	}
}

func TestClassListReplace(t *testing.T) {
	el := NewElement("div")
	el.SetAttribute("class", "foo bar")
	cl := el.ClassList()

	ok := cl.Replace("foo", "baz")
	if !ok {
		t.Error("Replace should return true when old class exists")
	}
	if el.GetAttribute("class") != "baz bar" {
		t.Errorf("class attr = %q, want %q", el.GetAttribute("class"), "baz bar")
	}

	ok = cl.Replace("notexist", "x")
	if ok {
		t.Error("Replace should return false when old class doesn't exist")
	}
}

func TestClassListValue(t *testing.T) {
	el := NewElement("div")
	el.SetAttribute("class", "  foo   bar  ")
	cl := el.ClassList()

	if cl.Value() != "foo bar" {
		t.Errorf("Value() = %q, want %q", cl.Value(), "foo bar")
	}
}

func TestClassListEmptyClass(t *testing.T) {
	el := NewElement("div")
	cl := el.ClassList()

	if cl.Length() != 0 {
		t.Errorf("Length() = %d, want 0", cl.Length())
	}
	if cl.Contains("anything") {
		t.Error("empty classList should not contain anything")
	}
}

func TestClassListSyncsWithAttribute(t *testing.T) {
	el := NewElement("div")
	cl := el.ClassList()

	cl.Add("foo")
	if el.GetAttribute("class") != "foo" {
		t.Errorf("attribute not synced after Add")
	}

	// Changing attribute should reflect in classList
	el.SetAttribute("class", "bar baz")
	if cl.Length() != 2 {
		t.Errorf("classList not synced after SetAttribute, got %d want 2", cl.Length())
	}
	if !cl.Contains("bar") || !cl.Contains("baz") {
		t.Error("classList should reflect new attribute value")
	}
}
