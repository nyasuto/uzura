package wpt

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
)

func TestDomainBreakdown(t *testing.T) {
	s := &Summary{}
	s.Add(&TestResult{
		Test:   "dom/nodes/append.html",
		Status: StatusPass,
		SubTests: []SubTest{
			{Name: "append single", Status: StatusPass},
			{Name: "append multiple", Status: StatusPass},
		},
	})
	s.Add(&TestResult{
		Test:   "dom/nodes/remove.html",
		Status: StatusFail,
		SubTests: []SubTest{
			{Name: "remove child", Status: StatusPass},
			{Name: "remove missing", Status: StatusFail},
		},
	})
	s.Add(&TestResult{
		Test:   "html/semantics/forms.html",
		Status: StatusPass,
		SubTests: []SubTest{
			{Name: "form submit", Status: StatusPass},
		},
	})
	s.Add(&TestResult{
		Test:   "html/dom/elements.html",
		Status: StatusFail,
		SubTests: []SubTest{
			{Name: "elem create", Status: StatusFail},
			{Name: "elem query", Status: StatusSkip},
		},
	})

	domains := s.DomainBreakdown()

	if len(domains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(domains))
	}

	dom, ok := domains["dom"]
	if !ok {
		t.Fatal("missing dom domain")
	}
	if dom.Total != 4 || dom.Pass != 3 || dom.Fail != 1 {
		t.Errorf("dom: got total=%d pass=%d fail=%d, want 4/3/1", dom.Total, dom.Pass, dom.Fail)
	}

	html, ok := domains["html"]
	if !ok {
		t.Fatal("missing html domain")
	}
	if html.Total != 3 || html.Pass != 1 || html.Fail != 1 || html.Skip != 1 {
		t.Errorf("html: got total=%d pass=%d fail=%d skip=%d, want 3/1/1/1",
			html.Total, html.Pass, html.Fail, html.Skip)
	}
}

func TestDomainBreakdownNoSubTests(t *testing.T) {
	s := &Summary{}
	s.Add(&TestResult{Test: "dom/test.html", Status: StatusPass})
	s.Add(&TestResult{Test: "css/test.html", Status: StatusFail})

	domains := s.DomainBreakdown()
	if domains["dom"].Pass != 1 {
		t.Errorf("dom pass: got %d, want 1", domains["dom"].Pass)
	}
	if domains["css"].Fail != 1 {
		t.Errorf("css fail: got %d, want 1", domains["css"].Fail)
	}
}

func TestWriteCSV(t *testing.T) {
	s := &Summary{}
	s.Add(&TestResult{
		Test:   "dom/test.html",
		Status: StatusPass,
		SubTests: []SubTest{
			{Name: "sub1", Status: StatusPass},
			{Name: "sub2", Status: StatusFail, Message: "expected true"},
		},
		DurationMS: 42.5,
	})
	s.Add(&TestResult{
		Test:       "html/test.html",
		Status:     StatusSkip,
		Message:    "not implemented",
		DurationMS: 0,
	})

	var buf bytes.Buffer
	err := s.WriteCSV(&buf)
	if err != nil {
		t.Fatal(err)
	}

	r := csv.NewReader(strings.NewReader(buf.String()))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	// Header + 3 data rows (2 subtests + 1 top-level).
	if len(records) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(records))
	}

	if records[0][0] != "test" || records[0][1] != "subtest" {
		t.Errorf("unexpected header: %v", records[0])
	}

	if records[1][0] != "dom/test.html" || records[1][1] != "sub1" || records[1][2] != "PASS" {
		t.Errorf("unexpected row: %v", records[1])
	}

	if records[3][0] != "html/test.html" || records[3][1] != "" || records[3][2] != "SKIP" {
		t.Errorf("unexpected row: %v", records[3])
	}
}

func TestWriteDomainReport(t *testing.T) {
	s := &Summary{}
	s.Add(&TestResult{
		Test:   "dom/nodes/test.html",
		Status: StatusPass,
		SubTests: []SubTest{
			{Name: "a", Status: StatusPass},
			{Name: "b", Status: StatusFail},
		},
	})
	s.Add(&TestResult{
		Test:   "css/selectors/test.html",
		Status: StatusPass,
		SubTests: []SubTest{
			{Name: "c", Status: StatusPass},
		},
	})

	var buf bytes.Buffer
	s.WriteDomainReport(&buf)
	out := buf.String()

	if !strings.Contains(out, "dom") || !strings.Contains(out, "css") {
		t.Errorf("missing domain names in report:\n%s", out)
	}
	if !strings.Contains(out, "50.0%") {
		t.Errorf("missing dom pass rate in report:\n%s", out)
	}
	if !strings.Contains(out, "100.0%") {
		t.Errorf("missing css pass rate in report:\n%s", out)
	}
}
