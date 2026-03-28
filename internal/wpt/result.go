// Package wpt provides a Web Platform Tests runner for Uzura.
package wpt

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// Status represents the outcome of a single WPT sub-test.
type Status int

const (
	// StatusPass indicates the test passed.
	StatusPass Status = iota
	// StatusFail indicates the test failed.
	StatusFail
	// StatusTimeout indicates the test timed out.
	StatusTimeout
	// StatusNotRun indicates the test was not run.
	StatusNotRun
	// StatusSkip indicates the test was skipped.
	StatusSkip
)

// String returns the human-readable status name.
func (s Status) String() string {
	switch s {
	case StatusPass:
		return "PASS"
	case StatusFail:
		return "FAIL"
	case StatusTimeout:
		return "TIMEOUT"
	case StatusNotRun:
		return "NOTRUN"
	case StatusSkip:
		return "SKIP"
	default:
		return "UNKNOWN"
	}
}

// MarshalJSON implements json.Marshaler.
func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON implements json.Unmarshaler.
func (s *Status) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	switch str {
	case "PASS":
		*s = StatusPass
	case "FAIL":
		*s = StatusFail
	case "TIMEOUT":
		*s = StatusTimeout
	case "NOTRUN":
		*s = StatusNotRun
	case "SKIP":
		*s = StatusSkip
	default:
		return fmt.Errorf("unknown status: %s", str)
	}
	return nil
}

// SubTest represents the result of a single sub-test within a WPT test file.
type SubTest struct {
	Name    string `json:"name"`
	Status  Status `json:"status"`
	Message string `json:"message,omitempty"`
}

// TestResult represents the result of running a single WPT test file.
type TestResult struct {
	Test       string        `json:"test"`
	Status     Status        `json:"status"`
	Message    string        `json:"message,omitempty"`
	SubTests   []SubTest     `json:"subtests,omitempty"`
	Duration   time.Duration `json:"-"`
	DurationMS float64       `json:"duration_ms"`
}

// PassCount returns the number of passing sub-tests.
func (r *TestResult) PassCount() int {
	n := 0
	for _, st := range r.SubTests {
		if st.Status == StatusPass {
			n++
		}
	}
	return n
}

// Summary holds aggregated results across multiple test files.
type Summary struct {
	Total   int           `json:"total"`
	Pass    int           `json:"pass"`
	Fail    int           `json:"fail"`
	Timeout int           `json:"timeout"`
	Skip    int           `json:"skip"`
	Results []*TestResult `json:"results"`
}

// Add incorporates a TestResult into the summary.
func (s *Summary) Add(r *TestResult) {
	s.Results = append(s.Results, r)
	for _, st := range r.SubTests {
		s.Total++
		switch st.Status {
		case StatusPass:
			s.Pass++
		case StatusFail:
			s.Fail++
		case StatusTimeout:
			s.Timeout++
		case StatusSkip, StatusNotRun:
			s.Skip++
		}
	}
	// If no sub-tests, count the top-level status.
	if len(r.SubTests) == 0 {
		s.Total++
		switch r.Status {
		case StatusPass:
			s.Pass++
		case StatusFail:
			s.Fail++
		case StatusTimeout:
			s.Timeout++
		case StatusSkip, StatusNotRun:
			s.Skip++
		}
	}
}

// WriteJSON writes the summary as JSON to w.
func (s *Summary) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(s)
}

// PassRate returns the pass rate as a percentage.
func (s *Summary) PassRate() float64 {
	if s.Total == 0 {
		return 0
	}
	return float64(s.Pass) / float64(s.Total) * 100
}
