package wpt

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nyasuto/uzura/internal/css"
	htmlpkg "github.com/nyasuto/uzura/internal/html"
	"github.com/nyasuto/uzura/internal/js"
)

// Runner executes WPT test files using the Uzura DOM + JS engine.
type Runner struct {
	// WPTDir is the root directory of the WPT checkout.
	// If empty, tests must be specified as absolute paths.
	WPTDir string

	// SkipList is the set of tests to skip. May be nil.
	SkipList *SkipList

	// Timeout per test file. Zero means 30 seconds.
	Timeout time.Duration

	// Output writer for progress. May be nil for silent.
	Output io.Writer
}

// RunDir runs all .html test files under the given directory (relative to WPTDir).
// Returns a Summary of all results.
func (r *Runner) RunDir(dir string) (*Summary, error) {
	root := dir
	if r.WPTDir != "" && !filepath.IsAbs(dir) {
		root = filepath.Join(r.WPTDir, dir)
	}

	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".html") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking directory %s: %w", root, err)
	}

	summary := &Summary{}
	for _, f := range files {
		result := r.RunFile(f)
		summary.Add(result)
	}
	return summary, nil
}

// RunFile runs a single WPT test file and returns the result.
func (r *Runner) RunFile(path string) *TestResult {
	absPath := path
	if r.WPTDir != "" && !filepath.IsAbs(path) {
		absPath = filepath.Join(r.WPTDir, path)
	}

	// Compute relative path for display.
	relPath := path
	if r.WPTDir != "" {
		if rel, err := filepath.Rel(r.WPTDir, absPath); err == nil {
			relPath = rel
		}
	}

	// Check skip list.
	if r.SkipList != nil {
		if reason, skip := r.SkipList.ShouldSkip(relPath); skip {
			r.log("SKIP %s (%s)\n", relPath, reason)
			return &TestResult{
				Test:    relPath,
				Status:  StatusSkip,
				Message: reason,
			}
		}
	}

	start := time.Now()
	result := r.runSingle(absPath, relPath)
	result.Duration = time.Since(start)
	result.DurationMS = float64(result.Duration.Milliseconds())

	// Log progress.
	pass := result.PassCount()
	total := len(result.SubTests)
	if total == 0 {
		r.log("%-4s %s (%.0fms)\n", result.Status, relPath, result.DurationMS)
	} else {
		r.log("%-4s %s [%d/%d] (%.0fms)\n", result.Status, relPath, pass, total, result.DurationMS)
	}
	return result
}

func (r *Runner) runSingle(absPath, relPath string) *TestResult {
	result := &TestResult{Test: relPath}

	// Read the test file.
	data, err := os.ReadFile(absPath)
	if err != nil {
		result.Status = StatusFail
		result.Message = fmt.Sprintf("read error: %v", err)
		return result
	}

	// Parse HTML.
	doc, err := htmlpkg.Parse(strings.NewReader(string(data)))
	if err != nil {
		result.Status = StatusFail
		result.Message = fmt.Sprintf("parse error: %v", err)
		return result
	}

	// Set up query engine for CSS selectors.
	doc.SetQueryEngine(css.NewEngine())

	// Create JS VM and bind DOM (also sets up Event constructor).
	vm := js.New(js.WithWriter(io.Discard))
	js.BindDocument(vm, doc)

	// Inject our testharness shim.
	if _, err := vm.Eval(harnessJS); err != nil {
		result.Status = StatusFail
		result.Message = fmt.Sprintf("harness init error: %v", err)
		return result
	}

	// Execute scripts from the document.
	_ = js.ExecuteScripts(vm, doc)

	// Run event loop to resolve any pending timers.
	vm.RunEventLoop()

	// Force completion for any unfinished async tests.
	_, _ = vm.Eval("__wpt_force_complete__()")

	// Collect results from the global __wpt_results__ object.
	r.collectResults(vm, result)
	return result
}

func (r *Runner) collectResults(vm *js.VM, result *TestResult) {
	// Read overall status.
	statusVal, err := vm.Eval("__wpt_results__.status")
	if err != nil {
		result.Status = StatusFail
		result.Message = "failed to read results"
		return
	}

	switch v := statusVal.(type) {
	case int64:
		switch v {
		case 0:
			result.Status = StatusPass
		case 2:
			result.Status = StatusTimeout
		default:
			result.Status = StatusFail
		}
	case float64:
		switch int(v) {
		case 0:
			result.Status = StatusPass
		case 2:
			result.Status = StatusTimeout
		default:
			result.Status = StatusFail
		}
	default:
		result.Status = StatusFail
	}

	// Read message.
	msgVal, msgErr := vm.Eval("__wpt_results__.message")
	if msgErr == nil {
		if s, ok := msgVal.(string); ok && s != "" {
			result.Message = s
		}
	}

	// Read sub-test count.
	countVal, err := vm.Eval("__wpt_results__.tests.length")
	if err != nil {
		return
	}
	count := toInt(countVal)

	for i := 0; i < count; i++ {
		st := SubTest{}

		nameVal, _ := vm.Eval(fmt.Sprintf("__wpt_results__.tests[%d].name", i))
		if s, ok := nameVal.(string); ok {
			st.Name = s
		}

		stVal, _ := vm.Eval(fmt.Sprintf("__wpt_results__.tests[%d].status", i))
		switch v := stVal.(type) {
		case int64:
			st.Status = intToStatus(int(v))
		case float64:
			st.Status = intToStatus(int(v))
		}

		msgVal, _ := vm.Eval(fmt.Sprintf("__wpt_results__.tests[%d].message", i))
		if s, ok := msgVal.(string); ok {
			st.Message = s
		}

		result.SubTests = append(result.SubTests, st)
	}

	// If all sub-tests pass, status is PASS; otherwise FAIL.
	if len(result.SubTests) > 0 && result.Status == StatusPass {
		for _, st := range result.SubTests {
			if st.Status != StatusPass {
				result.Status = StatusFail
				break
			}
		}
	}
}

func (r *Runner) log(format string, args ...interface{}) {
	if r.Output != nil {
		fmt.Fprintf(r.Output, format, args...)
	}
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

func intToStatus(n int) Status {
	switch n {
	case 0:
		return StatusPass
	case 1:
		return StatusFail
	case 2:
		return StatusTimeout
	case 3:
		return StatusNotRun
	default:
		return StatusFail
	}
}
