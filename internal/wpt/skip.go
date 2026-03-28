package wpt

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// SkipList manages a set of test paths that should be skipped.
// Each entry is a relative file path (e.g., "dom/nodes/Document-createElement.html").
type SkipList struct {
	entries map[string]string // path -> reason
}

// NewSkipList creates an empty skip list.
func NewSkipList() *SkipList {
	return &SkipList{entries: make(map[string]string)}
}

// Add adds a path to the skip list with an optional reason.
func (sl *SkipList) Add(path, reason string) {
	sl.entries[filepath.Clean(path)] = reason
}

// ShouldSkip reports whether the given test path should be skipped.
// It returns the reason if skipped.
func (sl *SkipList) ShouldSkip(path string) (string, bool) {
	reason, ok := sl.entries[filepath.Clean(path)]
	return reason, ok
}

// Len returns the number of entries in the skip list.
func (sl *SkipList) Len() int {
	return len(sl.entries)
}

// LoadSkipFile loads a skip list from a text file.
// Format: one path per line. Lines starting with # are comments.
// A line may have a reason after the path, separated by whitespace and a # character:
//
//	dom/nodes/Document-createElement.html  # missing DOMException
//	# This is a comment
func LoadSkipFile(path string) (*SkipList, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sl := NewSkipList()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		var testPath, reason string
		if idx := strings.Index(line, " #"); idx != -1 {
			testPath = strings.TrimSpace(line[:idx])
			reason = strings.TrimSpace(line[idx+2:])
		} else {
			testPath = line
		}
		if testPath != "" {
			sl.Add(testPath, reason)
		}
	}
	return sl, scanner.Err()
}
