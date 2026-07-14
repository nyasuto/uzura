package page

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// TestNavigateConcurrentWithVM_NoRace drives a concurrent Navigate (running a
// busy inline script) alongside repeated VM()+Eval calls on the same Page —
// the same access shape CDP's Runtime.evaluate/callFunctionOn has (fetch the
// VM pointer via Page.VM(), then drive it directly).
//
// Before the Finding 1 fix, runScripts published p.vm *before* script
// execution finished, so a concurrent VM() call could hand out the exact
// same *goja.Runtime pointer runScripts was actively executing scripts on.
// goja.Runtime is not safe for concurrent use, so two goroutines touching it
// at once is a data race — this test fails under `go test -race` against
// the unfixed code and must pass afterward.
func TestNavigateConcurrentWithVM_NoRace(t *testing.T) {
	// A script that keeps the runtime busy long enough that a concurrent
	// VM()/Eval call is very likely to land while runScripts is still
	// running, if the two ever end up sharing one *goja.Runtime.
	html := `<html><body><script>
		var sum = 0;
		for (var i = 0; i < 2e7; i++) { sum += i; }
	</script></body></html>`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, html)
	}))
	defer ts.Close()

	p := New(nil)
	defer func() { _ = p.Close() }()

	done := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(done)
		if err := p.Navigate(context.Background(), ts.URL); err != nil {
			t.Errorf("Navigate: %v", err)
		}
	}()

	// Mimic a concurrent CDP Runtime.evaluate: grab the VM pointer and drive
	// it directly, over and over, for as long as Navigate is in flight.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
			}
			vm := p.VM()
			if vm != nil {
				_, _ = vm.Eval("1 + 1")
			}
		}
	}()

	wg.Wait()
}
