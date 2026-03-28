package wpt

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestDiff(t *testing.T) {
	baseline := &Summary{
		Total: 10, Pass: 7, Fail: 2, Timeout: 1,
		Results: []*TestResult{
			{
				Test: "dom/nodes/append.html", Status: StatusPass,
				SubTests: []SubTest{
					{Name: "test1", Status: StatusPass},
					{Name: "test2", Status: StatusPass},
				},
			},
			{
				Test: "dom/nodes/remove.html", Status: StatusFail,
				SubTests: []SubTest{
					{Name: "remove ok", Status: StatusPass},
					{Name: "remove bad", Status: StatusFail},
				},
			},
		},
	}

	current := &Summary{
		Total: 10, Pass: 8, Fail: 1, Timeout: 1,
		Results: []*TestResult{
			{
				Test: "dom/nodes/append.html", Status: StatusPass,
				SubTests: []SubTest{
					{Name: "test1", Status: StatusPass},
					{Name: "test2", Status: StatusFail}, // regression
				},
			},
			{
				Test: "dom/nodes/remove.html", Status: StatusPass,
				SubTests: []SubTest{
					{Name: "remove ok", Status: StatusPass},
					{Name: "remove bad", Status: StatusPass}, // fixed
				},
			},
		},
	}

	diff := Diff(baseline, current)

	if diff.TotalDelta != 0 {
		t.Errorf("total delta: got %d, want 0", diff.TotalDelta)
	}
	if diff.PassDelta != 1 {
		t.Errorf("pass delta: got %d, want 1", diff.PassDelta)
	}
	if diff.FailDelta != -1 {
		t.Errorf("fail delta: got %d, want -1", diff.FailDelta)
	}
	if len(diff.Regressions) != 1 {
		t.Fatalf("regressions: got %d, want 1", len(diff.Regressions))
	}
	if diff.Regressions[0].Test != "dom/nodes/append.html" ||
		diff.Regressions[0].SubTest != "test2" {
		t.Errorf("unexpected regression: %+v", diff.Regressions[0])
	}
	if len(diff.Fixes) != 1 {
		t.Fatalf("fixes: got %d, want 1", len(diff.Fixes))
	}
	if diff.Fixes[0].SubTest != "remove bad" {
		t.Errorf("unexpected fix: %+v", diff.Fixes[0])
	}
}

func TestDiffNewTests(t *testing.T) {
	baseline := &Summary{
		Total: 1, Pass: 1,
		Results: []*TestResult{
			{Test: "dom/old.html", Status: StatusPass},
		},
	}
	current := &Summary{
		Total: 2, Pass: 2,
		Results: []*TestResult{
			{Test: "dom/old.html", Status: StatusPass},
			{Test: "dom/new.html", Status: StatusPass},
		},
	}

	diff := Diff(baseline, current)
	if len(diff.NewTests) != 1 || diff.NewTests[0] != "dom/new.html" {
		t.Errorf("new tests: got %v, want [dom/new.html]", diff.NewTests)
	}
}

func TestDiffWriteReport(t *testing.T) {
	diff := &DiffReport{
		TotalDelta: 2,
		PassDelta:  3,
		FailDelta:  -1,
		Regressions: []DiffEntry{
			{Test: "dom/a.html", SubTest: "sub1", OldStatus: StatusPass, NewStatus: StatusFail},
		},
		Fixes: []DiffEntry{
			{Test: "dom/b.html", SubTest: "sub2", OldStatus: StatusFail, NewStatus: StatusPass},
		},
		NewTests: []string{"dom/c.html"},
	}

	var buf bytes.Buffer
	diff.WriteReport(&buf)
	out := buf.String()

	if !strings.Contains(out, "+3") {
		t.Errorf("missing pass delta in report:\n%s", out)
	}
	if !strings.Contains(out, "Regressions") {
		t.Errorf("missing regressions section:\n%s", out)
	}
	if !strings.Contains(out, "Fixes") {
		t.Errorf("missing fixes section:\n%s", out)
	}
}

func TestDiffReportJSON(t *testing.T) {
	diff := &DiffReport{
		TotalDelta: 1,
		PassDelta:  1,
		Regressions: []DiffEntry{
			{Test: "dom/a.html", SubTest: "x", OldStatus: StatusPass, NewStatus: StatusFail},
		},
	}

	var buf bytes.Buffer
	if err := diff.WriteJSON(&buf); err != nil {
		t.Fatal(err)
	}

	var decoded DiffReport
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.PassDelta != 1 || len(decoded.Regressions) != 1 {
		t.Errorf("unexpected decoded diff: %+v", decoded)
	}
}
