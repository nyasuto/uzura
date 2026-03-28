package wpt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSkipList_AddAndCheck(t *testing.T) {
	sl := NewSkipList()
	sl.Add("dom/nodes/test.html", "not implemented")

	reason, skip := sl.ShouldSkip("dom/nodes/test.html")
	if !skip {
		t.Error("expected skip")
	}
	if reason != "not implemented" {
		t.Errorf("reason: got %q, want %q", reason, "not implemented")
	}

	_, skip = sl.ShouldSkip("dom/nodes/other.html")
	if skip {
		t.Error("should not skip other.html")
	}

	if sl.Len() != 1 {
		t.Errorf("len: got %d, want 1", sl.Len())
	}
}

func TestLoadSkipFile(t *testing.T) {
	dir := t.TempDir()
	content := `# WPT skip list
dom/nodes/Document-createElement.html  # missing DOMException
html/dom/elements/global-attributes/id-name.html

# another comment
dom/traversal/TreeWalker.html  # not implemented
`
	path := filepath.Join(dir, "skip.txt")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	sl, err := LoadSkipFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if sl.Len() != 3 {
		t.Errorf("len: got %d, want 3", sl.Len())
	}

	tests := []struct {
		path   string
		skip   bool
		reason string
	}{
		{"dom/nodes/Document-createElement.html", true, "missing DOMException"},
		{"html/dom/elements/global-attributes/id-name.html", true, ""},
		{"dom/traversal/TreeWalker.html", true, "not implemented"},
		{"dom/nodes/other.html", false, ""},
	}

	for _, tc := range tests {
		reason, skip := sl.ShouldSkip(tc.path)
		if skip != tc.skip {
			t.Errorf("%s: skip=%v, want %v", tc.path, skip, tc.skip)
		}
		if reason != tc.reason {
			t.Errorf("%s: reason=%q, want %q", tc.path, reason, tc.reason)
		}
	}
}

func TestLoadSkipFile_NotFound(t *testing.T) {
	_, err := LoadSkipFile("/nonexistent/skip.txt")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
