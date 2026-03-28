package js

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nyasuto/uzura/internal/css"
	"github.com/nyasuto/uzura/internal/html"
)

func TestExecuteScripts(t *testing.T) {
	doc, err := html.Parse(strings.NewReader(`
		<html><body>
		<script>document.title</script>
		<div id="test">Hello</div>
		<script>
			var el = document.getElementById("test");
			el.textContent = "Modified";
		</script>
		</body></html>
	`))
	if err != nil {
		t.Fatal(err)
	}
	doc.SetQueryEngine(css.NewEngine())

	var buf bytes.Buffer
	vm := New(WithWriter(&buf))
	BindDocument(vm, doc)
	errs := ExecuteScripts(vm, doc)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	el := doc.GetElementById("test")
	if el.TextContent() != "Modified" {
		t.Errorf("textContent = %q, want 'Modified'", el.TextContent())
	}
}

func TestExecuteScriptsOrder(t *testing.T) {
	doc, err := html.Parse(strings.NewReader(`
		<html><body>
		<script>var order = [];</script>
		<script>order.push("first");</script>
		<script>order.push("second");</script>
		<script>document.getElementById("result").textContent = order.join(",");</script>
		<div id="result"></div>
		</body></html>
	`))
	if err != nil {
		t.Fatal(err)
	}
	doc.SetQueryEngine(css.NewEngine())

	vm := New()
	BindDocument(vm, doc)
	errs := ExecuteScripts(vm, doc)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestExecuteScriptsErrorContinues(t *testing.T) {
	doc, err := html.Parse(strings.NewReader(`
		<html><body>
		<script>throw new Error("boom");</script>
		<script>document.getElementById("ok").textContent = "survived";</script>
		<div id="ok">original</div>
		</body></html>
	`))
	if err != nil {
		t.Fatal(err)
	}
	doc.SetQueryEngine(css.NewEngine())

	vm := New()
	BindDocument(vm, doc)
	errs := ExecuteScripts(vm, doc)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}

	el := doc.GetElementById("ok")
	if el.TextContent() != "survived" {
		t.Errorf("textContent = %q, want 'survived'", el.TextContent())
	}
}

func TestExecuteScriptsDeferAttribute(t *testing.T) {
	var buf bytes.Buffer
	doc, err := html.Parse(strings.NewReader(`
		<html><head>
		<script defer>console.log("deferred");</script>
		</head><body>
		<script>console.log("inline");</script>
		</body></html>
	`))
	if err != nil {
		t.Fatal(err)
	}
	doc.SetQueryEngine(css.NewEngine())

	vm := New(WithWriter(&buf))
	BindDocument(vm, doc)
	_ = ExecuteScripts(vm, doc)

	got := buf.String()
	if got != "inline\ndeferred\n" {
		t.Errorf("output = %q, want %q", got, "inline\ndeferred\n")
	}
}
