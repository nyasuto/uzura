// Package bench provides an integrated benchmark suite for Uzura.
package bench

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"github.com/nyasuto/uzura/internal/css"
	"github.com/nyasuto/uzura/internal/dom"
	htmlpkg "github.com/nyasuto/uzura/internal/html"
	"github.com/nyasuto/uzura/internal/js"
	"github.com/nyasuto/uzura/internal/page"
)

// buildRealisticHTML generates a realistic HTML page with n items.
func buildRealisticHTML(n int) string {
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html><html><head><title>Bench Page</title></head><body>`)
	sb.WriteString(`<header><nav id="nav"><ul>`)
	for i := 0; i < 10; i++ {
		fmt.Fprintf(&sb, `<li><a href="/page/%d" class="nav-link">Page %d</a></li>`, i, i)
	}
	sb.WriteString(`</ul></nav></header><main>`)
	for i := 0; i < n; i++ {
		cls := "item"
		if i%2 == 0 {
			cls += " even"
		}
		if i%5 == 0 {
			cls += " featured"
		}
		fmt.Fprintf(&sb, `<div class="%s" id="item-%d" data-index="%d">`, cls, i, i)
		fmt.Fprintf(&sb, `<h3>Item %d</h3>`, i)
		fmt.Fprintf(&sb, `<p>Description for item %d with some text content.</p>`, i)
		fmt.Fprintf(&sb, `<span class="price">$%d.99</span>`, i*10+99)
		sb.WriteString(`</div>`)
	}
	sb.WriteString(`</main><footer><p>Footer content</p></footer></body></html>`)
	return sb.String()
}

func parseDoc(html string) *dom.Document {
	doc, err := htmlpkg.Parse(strings.NewReader(html))
	if err != nil {
		panic(err)
	}
	doc.SetQueryEngine(css.NewEngine())
	return doc
}

// --- Page Load Benchmarks ---

func BenchmarkPageLoad_Small(b *testing.B) {
	html := buildRealisticHTML(10)
	benchPageLoad(b, html)
}

func BenchmarkPageLoad_Medium(b *testing.B) {
	html := buildRealisticHTML(100)
	benchPageLoad(b, html)
}

func BenchmarkPageLoad_Large(b *testing.B) {
	html := buildRealisticHTML(1000)
	benchPageLoad(b, html)
}

func benchPageLoad(b *testing.B, html string) {
	b.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	}))
	defer srv.Close()

	b.SetBytes(int64(len(html)))
	b.ResetTimer()

	for b.Loop() {
		p := page.New(nil)
		if err := p.Navigate(context.Background(), srv.URL); err != nil {
			b.Fatal(err)
		}
		p.Close()
	}
}

// --- DOM Operation Benchmarks ---

func BenchmarkDOM_QuerySelector(b *testing.B) {
	doc := parseDoc(buildRealisticHTML(500))
	b.ResetTimer()
	for b.Loop() {
		doc.QuerySelector(".featured")
	}
}

func BenchmarkDOM_QuerySelectorAll(b *testing.B) {
	doc := parseDoc(buildRealisticHTML(500))
	b.ResetTimer()
	for b.Loop() {
		doc.QuerySelectorAll(".item.even")
	}
}

func BenchmarkDOM_GetElementById(b *testing.B) {
	doc := parseDoc(buildRealisticHTML(500))
	b.ResetTimer()
	for b.Loop() {
		doc.GetElementById("item-250")
	}
}

func BenchmarkDOM_GetElementsByClassName(b *testing.B) {
	doc := parseDoc(buildRealisticHTML(500))
	b.ResetTimer()
	for b.Loop() {
		doc.GetElementsByClassName("featured")
	}
}

func BenchmarkDOM_AppendChild(b *testing.B) {
	doc := parseDoc(buildRealisticHTML(10))
	body := doc.Body()
	b.ResetTimer()
	for b.Loop() {
		el := doc.CreateElement("div")
		body.AppendChild(el)
	}
}

func BenchmarkDOM_RemoveChild(b *testing.B) {
	doc := parseDoc(buildRealisticHTML(10))
	body := doc.Body()
	// Pre-create elements to remove.
	elems := make([]dom.Node, b.N)
	for i := range elems {
		elems[i] = doc.CreateElement("div")
		body.AppendChild(elems[i])
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body.RemoveChild(elems[i])
	}
}

// --- Complex Selector Benchmarks ---

func BenchmarkSelector_Descendant(b *testing.B) {
	doc := parseDoc(buildRealisticHTML(500))
	b.ResetTimer()
	for b.Loop() {
		doc.QuerySelectorAll("main div.item h3")
	}
}

func BenchmarkSelector_Attribute(b *testing.B) {
	doc := parseDoc(buildRealisticHTML(500))
	b.ResetTimer()
	for b.Loop() {
		doc.QuerySelectorAll("[data-index]")
	}
}

func BenchmarkSelector_NthChild(b *testing.B) {
	doc := parseDoc(buildRealisticHTML(500))
	b.ResetTimer()
	for b.Loop() {
		doc.QuerySelectorAll("div.item:nth-child(2n)")
	}
}

// --- JS Execution Benchmarks ---

func BenchmarkJS_SimpleExpr(b *testing.B) {
	vm := js.New()
	b.ResetTimer()
	for b.Loop() {
		vm.Eval("1 + 2 * 3")
	}
}

func BenchmarkJS_DOMQuery(b *testing.B) {
	doc := parseDoc(buildRealisticHTML(100))
	vm := js.New()
	js.BindDocument(vm, doc)
	b.ResetTimer()
	for b.Loop() {
		vm.Eval(`document.querySelectorAll('.item').length`)
	}
}

func BenchmarkJS_DOMMutation(b *testing.B) {
	doc := parseDoc(buildRealisticHTML(10))
	vm := js.New()
	js.BindDocument(vm, doc)
	b.ResetTimer()
	for b.Loop() {
		vm.Eval(`
			var el = document.createElement('div');
			el.textContent = 'bench';
			document.body.appendChild(el);
		`)
	}
}

func BenchmarkJS_ScriptExecution(b *testing.B) {
	html := `<!DOCTYPE html><html><body>
		<script>var x = 0; for(var i=0;i<100;i++){x+=i;}</script>
	</body></html>`

	b.ResetTimer()
	for b.Loop() {
		doc := parseDoc(html)
		vm := js.New()
		js.BindDocument(vm, doc)
		js.ExecuteScripts(vm, doc)
	}
}

// --- Memory Usage Benchmarks ---

func BenchmarkMemory_Parse1000(b *testing.B) {
	html := buildRealisticHTML(1000)
	b.ResetTimer()

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	for b.Loop() {
		parseDoc(html)
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)
	b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc)/float64(b.N), "bytes/op-total")
}

func BenchmarkMemory_PageLoad(b *testing.B) {
	html := buildRealisticHTML(100)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	}))
	defer srv.Close()

	b.ResetTimer()

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	for b.Loop() {
		p := page.New(nil)
		p.Navigate(context.Background(), srv.URL)
		p.Close()
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)
	b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc)/float64(b.N), "bytes/op-total")
}
