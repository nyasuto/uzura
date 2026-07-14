package js

import (
	"context"
	"io"
	"testing"
)

func TestLocation_Parts(t *testing.T) {
	vm := New(WithWriter(io.Discard))
	BindLocation(vm, "https://user.example.com:8443/app/page?q=1#top")
	got, _ := vm.Eval(`JSON.stringify([location.href, location.origin, location.protocol,
		location.host, location.hostname, location.port, location.pathname, location.search, location.hash])`)
	want := `["https://user.example.com:8443/app/page?q=1#top","https://user.example.com:8443","https:","user.example.com:8443","user.example.com","8443","/app/page","?q=1","#top"]`
	if got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestHistory_PushStateAndPopstate(t *testing.T) {
	vm := New(WithWriter(io.Discard))
	BindLocation(vm, "https://example.com/app")
	_, err := vm.Eval(`
		var events = [];
		window.addEventListener("popstate", function(e) {
			events.push({path: location.pathname, state: e.state});
		});
		history.pushState({page: 2}, "", "/app/page2");
		var afterPush = location.pathname;   // 即時反映
		history.pushState({page: 3}, "", "/app/page3");
		history.back();                       // popstate は非同期
		var beforeLoop = events.length;       // まだ 0
	`)
	if err != nil {
		t.Fatal(err)
	}
	_ = vm.RunEventLoopContext(context.Background())
	got, _ := vm.Eval(`JSON.stringify({afterPush: afterPush, beforeLoop: beforeLoop, events: events, len: history.length})`)
	want := `{"afterPush":"/app/page2","beforeLoop":0,"events":[{"path":"/app/page2","state":{"page":2}}],"len":3}`
	if got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestLocation_HashChange(t *testing.T) {
	vm := New(WithWriter(io.Discard))
	BindLocation(vm, "https://example.com/app")
	_, _ = vm.Eval(`
		var fired = "";
		window.addEventListener("hashchange", function() { fired = location.hash; });
		location.hash = "#sec2";
	`)
	_ = vm.RunEventLoopContext(context.Background())
	got, _ := vm.Eval(`JSON.stringify([location.hash, fired])`)
	if got != `["#sec2","#sec2"]` {
		t.Errorf("got %v", got)
	}
}
