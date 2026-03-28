package dom

import "testing"

func TestDatasetGet(t *testing.T) {
	el := NewElement("div")
	el.SetAttribute("data-user-id", "42")
	el.SetAttribute("data-name", "alice")
	ds := el.Dataset()

	if ds.Get("userId") != "42" {
		t.Errorf("Get(userId) = %q, want %q", ds.Get("userId"), "42")
	}
	if ds.Get("name") != "alice" {
		t.Errorf("Get(name) = %q, want %q", ds.Get("name"), "alice")
	}
	if ds.Get("nonexistent") != "" {
		t.Errorf("Get(nonexistent) should return empty")
	}
}

func TestDatasetSet(t *testing.T) {
	el := NewElement("div")
	ds := el.Dataset()

	ds.Set("userId", "42")
	if el.GetAttribute("data-user-id") != "42" {
		t.Errorf("attribute = %q, want %q", el.GetAttribute("data-user-id"), "42")
	}

	ds.Set("name", "bob")
	if el.GetAttribute("data-name") != "bob" {
		t.Errorf("attribute = %q, want %q", el.GetAttribute("data-name"), "bob")
	}
}

func TestDatasetHas(t *testing.T) {
	el := NewElement("div")
	el.SetAttribute("data-foo", "bar")
	ds := el.Dataset()

	if !ds.Has("foo") {
		t.Error("should have foo")
	}
	if ds.Has("bar") {
		t.Error("should not have bar")
	}
}

func TestDatasetDelete(t *testing.T) {
	el := NewElement("div")
	el.SetAttribute("data-foo", "bar")
	ds := el.Dataset()

	ds.Delete("foo")
	if el.HasAttribute("data-foo") {
		t.Error("attribute should be removed")
	}
}

func TestDatasetAll(t *testing.T) {
	el := NewElement("div")
	el.SetAttribute("data-first-name", "John")
	el.SetAttribute("data-age", "30")
	el.SetAttribute("class", "person") // non-data attribute
	ds := el.Dataset()

	all := ds.All()
	if len(all) != 2 {
		t.Fatalf("All() length = %d, want 2", len(all))
	}
	if all["firstName"] != "John" {
		t.Errorf("firstName = %q, want %q", all["firstName"], "John")
	}
	if all["age"] != "30" {
		t.Errorf("age = %q, want %q", all["age"], "30")
	}
}

func TestDatasetCamelCaseConversion(t *testing.T) {
	tests := []struct {
		attr  string
		camel string
	}{
		{"data-foo", "foo"},
		{"data-foo-bar", "fooBar"},
		{"data-foo-bar-baz", "fooBarBaz"},
		{"data-a", "a"},
	}
	for _, tt := range tests {
		t.Run(tt.attr, func(t *testing.T) {
			el := NewElement("div")
			el.SetAttribute(tt.attr, "val")
			ds := el.Dataset()
			if ds.Get(tt.camel) != "val" {
				t.Errorf("Get(%q) = %q, want %q", tt.camel, ds.Get(tt.camel), "val")
			}
		})
	}
}

func TestDatasetCamelToKebab(t *testing.T) {
	tests := []struct {
		camel string
		attr  string
	}{
		{"foo", "data-foo"},
		{"fooBar", "data-foo-bar"},
		{"fooBarBaz", "data-foo-bar-baz"},
	}
	for _, tt := range tests {
		t.Run(tt.camel, func(t *testing.T) {
			el := NewElement("div")
			ds := el.Dataset()
			ds.Set(tt.camel, "val")
			if el.GetAttribute(tt.attr) != "val" {
				t.Errorf("expected attr %q, got %q", tt.attr, el.GetAttribute(tt.attr))
			}
		})
	}
}
