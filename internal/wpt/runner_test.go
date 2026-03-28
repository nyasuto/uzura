package wpt

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// writeTestFile creates a temporary HTML test file and returns its path.
func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunFile_AllPass(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "pass.html", `<!DOCTYPE html>
<html><body>
<div id="target">hello</div>
<script>
test(function() {
    var el = document.getElementById("target");
    assert_equals(el.textContent, "hello");
}, "getElementById returns correct element");

test(function() {
    assert_true(true);
}, "assert_true works");

test(function() {
    assert_false(false);
}, "assert_false works");
</script>
</body></html>`)

	r := &Runner{WPTDir: dir}
	result := r.RunFile("pass.html")

	if result.Status != StatusPass {
		t.Errorf("expected PASS, got %s", result.Status)
	}
	if len(result.SubTests) != 3 {
		t.Fatalf("expected 3 subtests, got %d", len(result.SubTests))
	}
	for _, st := range result.SubTests {
		if st.Status != StatusPass {
			t.Errorf("subtest %q: expected PASS, got %s: %s", st.Name, st.Status, st.Message)
		}
	}
}

func TestRunFile_SomeFail(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "mixed.html", `<!DOCTYPE html>
<html><body>
<script>
test(function() {
    assert_equals(1, 1);
}, "should pass");

test(function() {
    assert_equals(1, 2);
}, "should fail");
</script>
</body></html>`)

	r := &Runner{WPTDir: dir}
	result := r.RunFile("mixed.html")

	if result.Status != StatusFail {
		t.Errorf("expected FAIL, got %s", result.Status)
	}
	if len(result.SubTests) != 2 {
		t.Fatalf("expected 2 subtests, got %d", len(result.SubTests))
	}
	if result.SubTests[0].Status != StatusPass {
		t.Errorf("subtest 0: expected PASS, got %s", result.SubTests[0].Status)
	}
	if result.SubTests[1].Status != StatusFail {
		t.Errorf("subtest 1: expected FAIL, got %s", result.SubTests[1].Status)
	}
	if result.SubTests[1].Message == "" {
		t.Error("failed subtest should have a message")
	}
}

func TestRunFile_DOMManipulation(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "dom.html", `<!DOCTYPE html>
<html><body>
<div id="container"></div>
<script>
test(function() {
    var container = document.getElementById("container");
    var child = document.createElement("span");
    child.textContent = "added";
    container.appendChild(child);
    assert_equals(container.textContent, "added");
}, "appendChild works");

test(function() {
    var el = document.createElement("div");
    el.setAttribute("class", "foo bar");
    assert_equals(el.className, "foo bar");
    assert_true(el.hasAttribute("class"));
}, "setAttribute and className");

test(function() {
    var list = document.querySelectorAll("div");
    assert_true(list.length >= 1, "at least one div");
}, "querySelectorAll returns results");
</script>
</body></html>`)

	r := &Runner{WPTDir: dir}
	result := r.RunFile("dom.html")

	if result.Status != StatusPass {
		t.Errorf("expected PASS, got %s", result.Status)
		for _, st := range result.SubTests {
			if st.Status != StatusPass {
				t.Logf("  FAIL: %s — %s", st.Name, st.Message)
			}
		}
	}
	if len(result.SubTests) != 3 {
		t.Errorf("expected 3 subtests, got %d", len(result.SubTests))
	}
}

func TestRunFile_Assertions(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "assertions.html", `<!DOCTYPE html>
<html><body>
<script>
test(function() {
    assert_not_equals(1, 2);
}, "assert_not_equals");

test(function() {
    assert_array_equals([1, 2, 3], [1, 2, 3]);
}, "assert_array_equals");

test(function() {
    assert_throws_js(TypeError, function() { null.x; });
}, "assert_throws_js");

test(function() {
    assert_unreached("should not reach");
}, "assert_unreached should fail");
</script>
</body></html>`)

	r := &Runner{WPTDir: dir}
	result := r.RunFile("assertions.html")

	if len(result.SubTests) != 4 {
		t.Fatalf("expected 4 subtests, got %d", len(result.SubTests))
	}
	// First three should pass, last should fail.
	for i := 0; i < 3; i++ {
		if result.SubTests[i].Status != StatusPass {
			t.Errorf("subtest %d (%s): expected PASS, got %s: %s",
				i, result.SubTests[i].Name, result.SubTests[i].Status, result.SubTests[i].Message)
		}
	}
	if result.SubTests[3].Status != StatusFail {
		t.Errorf("assert_unreached test should FAIL, got %s", result.SubTests[3].Status)
	}
}

func TestRunDir(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join("sub", "tests")

	writeTestFile(t, dir, filepath.Join(subdir, "a.html"), `<!DOCTYPE html>
<script>
test(function() { assert_true(true); }, "test a");
</script>`)

	writeTestFile(t, dir, filepath.Join(subdir, "b.html"), `<!DOCTYPE html>
<script>
test(function() { assert_true(true); }, "test b1");
test(function() { assert_true(true); }, "test b2");
</script>`)

	r := &Runner{WPTDir: dir}
	summary, err := r.RunDir(subdir)
	if err != nil {
		t.Fatal(err)
	}

	if summary.Total != 3 {
		t.Errorf("expected 3 total, got %d", summary.Total)
	}
	if summary.Pass != 3 {
		t.Errorf("expected 3 pass, got %d", summary.Pass)
	}
	if len(summary.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(summary.Results))
	}
}

func TestRunFile_SkipList(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "skipped.html", `<!DOCTYPE html>
<script>
test(function() { assert_true(false); }, "would fail");
</script>`)

	sl := NewSkipList()
	sl.Add("skipped.html", "known issue")

	r := &Runner{WPTDir: dir, SkipList: sl}
	result := r.RunFile("skipped.html")

	if result.Status != StatusSkip {
		t.Errorf("expected SKIP, got %s", result.Status)
	}
	if result.Message != "known issue" {
		t.Errorf("expected reason 'known issue', got %q", result.Message)
	}
}

func TestSummary_JSON(t *testing.T) {
	s := &Summary{}
	s.Add(&TestResult{
		Test:   "test.html",
		Status: StatusPass,
		SubTests: []SubTest{
			{Name: "a", Status: StatusPass},
			{Name: "b", Status: StatusFail, Message: "oops"},
		},
	})

	if s.Total != 2 {
		t.Errorf("total: got %d, want 2", s.Total)
	}
	if s.Pass != 1 {
		t.Errorf("pass: got %d, want 1", s.Pass)
	}
	if s.Fail != 1 {
		t.Errorf("fail: got %d, want 1", s.Fail)
	}

	var buf bytes.Buffer
	if err := s.WriteJSON(&buf); err != nil {
		t.Fatal(err)
	}

	var decoded Summary
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Total != 2 {
		t.Errorf("decoded total: got %d, want 2", decoded.Total)
	}
}

func TestSummary_PassRate(t *testing.T) {
	s := &Summary{Total: 10, Pass: 7}
	rate := s.PassRate()
	if rate != 70.0 {
		t.Errorf("expected 70.0%%, got %.1f%%", rate)
	}

	empty := &Summary{}
	if empty.PassRate() != 0 {
		t.Errorf("empty summary should have 0%% pass rate")
	}
}

func TestRunFile_OutputProgress(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "progress.html", `<!DOCTYPE html>
<script>
test(function() { assert_true(true); }, "ok");
</script>`)

	var buf bytes.Buffer
	r := &Runner{WPTDir: dir, Output: &buf}
	r.RunFile("progress.html")

	if buf.Len() == 0 {
		t.Error("expected progress output")
	}
}

func TestStatusJSON(t *testing.T) {
	tests := []struct {
		s    Status
		want string
	}{
		{StatusPass, `"PASS"`},
		{StatusFail, `"FAIL"`},
		{StatusTimeout, `"TIMEOUT"`},
		{StatusNotRun, `"NOTRUN"`},
		{StatusSkip, `"SKIP"`},
	}
	for _, tc := range tests {
		data, err := json.Marshal(tc.s)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != tc.want {
			t.Errorf("Marshal(%s): got %s, want %s", tc.s, data, tc.want)
		}
		var got Status
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatal(err)
		}
		if got != tc.s {
			t.Errorf("Unmarshal: got %s, want %s", got, tc.s)
		}
	}
}
