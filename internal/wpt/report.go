package wpt

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// DomainSummary holds aggregated pass/fail counts for a single domain.
type DomainSummary struct {
	Domain  string  `json:"domain"`
	Total   int     `json:"total"`
	Pass    int     `json:"pass"`
	Fail    int     `json:"fail"`
	Skip    int     `json:"skip"`
	Timeout int     `json:"timeout"`
	Rate    float64 `json:"pass_rate"`
}

// DomainBreakdown returns per-domain summaries keyed by top-level directory.
func (s *Summary) DomainBreakdown() map[string]*DomainSummary {
	domains := make(map[string]*DomainSummary)

	for _, r := range s.Results {
		domain := extractDomain(r.Test)
		ds, ok := domains[domain]
		if !ok {
			ds = &DomainSummary{Domain: domain}
			domains[domain] = ds
		}

		if len(r.SubTests) == 0 {
			ds.Total++
			addStatus(ds, r.Status)
		} else {
			for _, st := range r.SubTests {
				ds.Total++
				addStatus(ds, st.Status)
			}
		}
	}

	for _, ds := range domains {
		if ds.Total > 0 {
			ds.Rate = float64(ds.Pass) / float64(ds.Total) * 100
		}
	}
	return domains
}

// WriteDomainReport writes a formatted domain-level report to w.
func (s *Summary) WriteDomainReport(w io.Writer) {
	domains := s.DomainBreakdown()

	// Sort domain names for stable output.
	names := make([]string, 0, len(domains))
	for name := range domains {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Fprintf(w, "\n%-20s %6s %6s %6s %6s %6s %8s\n",
		"Domain", "Total", "Pass", "Fail", "Skip", "T/O", "Rate")
	fmt.Fprintf(w, "%s\n", strings.Repeat("-", 72))

	for _, name := range names {
		ds := domains[name]
		fmt.Fprintf(w, "%-20s %6d %6d %6d %6d %6d %7.1f%%\n",
			ds.Domain, ds.Total, ds.Pass, ds.Fail, ds.Skip, ds.Timeout, ds.Rate)
	}

	fmt.Fprintf(w, "%s\n", strings.Repeat("-", 72))
	fmt.Fprintf(w, "%-20s %6d %6d %6d %6d %6d %7.1f%%\n",
		"TOTAL", s.Total, s.Pass, s.Fail, s.Skip, s.Timeout, s.PassRate())
}

// WriteCSV writes per-subtest results in CSV format.
func (s *Summary) WriteCSV(w io.Writer) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	if err := cw.Write([]string{"test", "subtest", "status", "message", "duration_ms"}); err != nil {
		return err
	}

	for _, r := range s.Results {
		dur := fmt.Sprintf("%.1f", r.DurationMS)
		if len(r.SubTests) == 0 {
			if err := cw.Write([]string{r.Test, "", r.Status.String(), r.Message, dur}); err != nil {
				return err
			}
		} else {
			for _, st := range r.SubTests {
				if err := cw.Write([]string{r.Test, st.Name, st.Status.String(), st.Message, dur}); err != nil {
					return err
				}
			}
		}
	}
	return cw.Error()
}

// DiffEntry represents a single sub-test that changed status.
type DiffEntry struct {
	Test      string `json:"test"`
	SubTest   string `json:"subtest,omitempty"`
	OldStatus Status `json:"old_status"`
	NewStatus Status `json:"new_status"`
}

// DiffReport holds the diff between a baseline and current run.
type DiffReport struct {
	TotalDelta   int         `json:"total_delta"`
	PassDelta    int         `json:"pass_delta"`
	FailDelta    int         `json:"fail_delta"`
	TimeoutDelta int         `json:"timeout_delta"`
	SkipDelta    int         `json:"skip_delta"`
	Regressions  []DiffEntry `json:"regressions,omitempty"`
	Fixes        []DiffEntry `json:"fixes,omitempty"`
	NewTests     []string    `json:"new_tests,omitempty"`
}

// Diff compares baseline and current summaries and produces a DiffReport.
func Diff(baseline, current *Summary) *DiffReport {
	d := &DiffReport{
		TotalDelta:   current.Total - baseline.Total,
		PassDelta:    current.Pass - baseline.Pass,
		FailDelta:    current.Fail - baseline.Fail,
		TimeoutDelta: current.Timeout - baseline.Timeout,
		SkipDelta:    current.Skip - baseline.Skip,
	}

	// Build a map of baseline sub-test statuses: "test|subtest" -> Status.
	baseMap := buildStatusMap(baseline)
	currMap := buildStatusMap(current)

	// Track which baseline tests exist.
	baseTests := make(map[string]bool)
	for _, r := range baseline.Results {
		baseTests[r.Test] = true
	}

	// Find regressions and fixes.
	for key, newStatus := range currMap {
		oldStatus, existed := baseMap[key]
		if !existed {
			continue
		}
		test, subtest := splitKey(key)
		if oldStatus == StatusPass && newStatus != StatusPass {
			d.Regressions = append(d.Regressions, DiffEntry{
				Test: test, SubTest: subtest,
				OldStatus: oldStatus, NewStatus: newStatus,
			})
		} else if oldStatus != StatusPass && newStatus == StatusPass {
			d.Fixes = append(d.Fixes, DiffEntry{
				Test: test, SubTest: subtest,
				OldStatus: oldStatus, NewStatus: newStatus,
			})
		}
	}

	// Find new tests.
	for _, r := range current.Results {
		if !baseTests[r.Test] {
			d.NewTests = append(d.NewTests, r.Test)
		}
	}

	sort.Slice(d.Regressions, func(i, j int) bool { return d.Regressions[i].Test < d.Regressions[j].Test })
	sort.Slice(d.Fixes, func(i, j int) bool { return d.Fixes[i].Test < d.Fixes[j].Test })
	sort.Strings(d.NewTests)

	return d
}

// WriteReport writes a human-readable diff report to w.
func (d *DiffReport) WriteReport(w io.Writer) {
	fmt.Fprintf(w, "\n=== Diff Report ===\n")
	fmt.Fprintf(w, "Total: %s  Pass: %s  Fail: %s  Skip: %s  Timeout: %s\n",
		formatDelta(d.TotalDelta), formatDelta(d.PassDelta),
		formatDelta(d.FailDelta), formatDelta(d.SkipDelta),
		formatDelta(d.TimeoutDelta))

	if len(d.Regressions) > 0 {
		fmt.Fprintf(w, "\nRegressions (%d):\n", len(d.Regressions))
		for _, e := range d.Regressions {
			name := e.Test
			if e.SubTest != "" {
				name += " > " + e.SubTest
			}
			fmt.Fprintf(w, "  %s → %s  %s\n", e.OldStatus, e.NewStatus, name)
		}
	}

	if len(d.Fixes) > 0 {
		fmt.Fprintf(w, "\nFixes (%d):\n", len(d.Fixes))
		for _, e := range d.Fixes {
			name := e.Test
			if e.SubTest != "" {
				name += " > " + e.SubTest
			}
			fmt.Fprintf(w, "  %s → %s  %s\n", e.OldStatus, e.NewStatus, name)
		}
	}

	if len(d.NewTests) > 0 {
		fmt.Fprintf(w, "\nNew tests (%d):\n", len(d.NewTests))
		for _, t := range d.NewTests {
			fmt.Fprintf(w, "  %s\n", t)
		}
	}

	if len(d.Regressions) == 0 && len(d.Fixes) == 0 && len(d.NewTests) == 0 {
		fmt.Fprintf(w, "\nNo changes detected.\n")
	}
}

// WriteJSON writes the diff report as JSON.
func (d *DiffReport) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(d)
}

// LoadSummary reads a JSON Summary from r.
func LoadSummary(r io.Reader) (*Summary, error) {
	var s Summary
	if err := json.NewDecoder(r).Decode(&s); err != nil {
		return nil, fmt.Errorf("decoding summary: %w", err)
	}
	return &s, nil
}

func extractDomain(testPath string) string {
	parts := strings.SplitN(testPath, "/", 2)
	if len(parts) == 0 {
		return "unknown"
	}
	return parts[0]
}

func addStatus(ds *DomainSummary, st Status) {
	switch st {
	case StatusPass:
		ds.Pass++
	case StatusFail:
		ds.Fail++
	case StatusTimeout:
		ds.Timeout++
	case StatusSkip, StatusNotRun:
		ds.Skip++
	}
}

func buildStatusMap(s *Summary) map[string]Status {
	m := make(map[string]Status)
	for _, r := range s.Results {
		if len(r.SubTests) == 0 {
			m[r.Test+"|"] = r.Status
		} else {
			for _, st := range r.SubTests {
				m[r.Test+"|"+st.Name] = st.Status
			}
		}
	}
	return m
}

func splitKey(key string) (test, subtest string) {
	i := strings.Index(key, "|")
	if i < 0 {
		return key, ""
	}
	return key[:i], key[i+1:]
}

func formatDelta(d int) string {
	if d > 0 {
		return fmt.Sprintf("+%d", d)
	}
	return fmt.Sprintf("%d", d)
}
