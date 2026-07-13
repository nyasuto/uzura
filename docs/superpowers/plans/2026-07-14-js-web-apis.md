# JS Web API バインディング（fetch / XHR / storage / location / history）実装計画

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** CSR サイトが `browse`/`query` で本文を返せるように、fetch / XMLHttpRequest / localStorage / location / history を goja VM にバインドし、page.Navigate に JS 実行を組み込む。

**Architecture:** 既存のタイマー専用イベントループを「タスクキュー + 飛行中カウンター + ctx デッドライン」に拡張。ネットワーク I/O は goroutine で行い、完了時にループスレッド上のタスクとして Promise 解決 / XHR イベント発火する。`internal/js` は `internal/network` を import せず、`HTTPClient` 関数型を page が注入する。

**Tech Stack:** Go 1.26 / goja（Promise, DynamicObject, ConstructorCall）/ net/url / httptest

**Spec:** `docs/superpowers/specs/2026-07-14-js-web-apis-design.md`（承認済み。CORS 適用なし・sync XHR 非対応・storage は Page 単位）

## Global Constraints

- Pure Go、cgo 禁止。新規外部依存の追加禁止
- 1 ファイル 300 行以内（`internal/js/binding_*.go` の分割方針は本計画のファイル構成に従う)
- エクスポート名には godoc コメント必須。エラーは返す、panic 禁止（goroutine 内は recover）
- テストはテーブル駆動。`go test ./... -race` 全パス必須
- **goja の Runtime に触れるのはループスレッドのみ**。goroutine から `goja.Runtime` / `goja.Value` / resolve/reject を直接呼ばない（タスク経由で行う）
- コミットは `git commit` 時に pre-commit フック（gofmt + golangci-lint + 全テスト）が走る。数分かかっても待つ
- コミットメッセージ末尾に必ず付ける:
  ```
  Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
  Claude-Session: https://claude.ai/code/session_01YYp6mqV9NHcfY7YaJtTGrJ
  ```
- 作業ブランチ: `feat/js-web-apis`（作成済み、スペックがコミット済み）

## 重要な前提知識（コードベースの現状）

- `internal/js/vm.go`: `VM` 構造体（`runtime *goja.Runtime`, `loop *eventLoop`）。`New(opts...)` → `init()` で globals/console/timers を設定。`window` = グローバルオブジェクト
- `internal/js/eventloop.go`: タイマー heap のみ。`run()` は「heap が空になったら終了」で、飛行中 I/O を待てない
- **`page.Navigate`（`internal/page/navigate_helpers.go:18`）は現状 `<script>` を実行していない**。`js.ExecuteScripts` + `RunEventLoop` の利用者は WPT ランナーのみ
- `network.Fetcher` は GET 専用（`doFetch` が `http.MethodGet` 固定）。POST/body 対応の追加が必要
- `page.Page` は `fetcher *network.Fetcher`, `url string`, `vm *js.VM`, `vmOptions []js.Option` を持つ。VM は `Page.VM()` で遅延生成
- goja: `runtime.NewPromise()` は `(promise, resolveFn, rejectFn)` を返す。resolve/reject はループスレッドから呼ぶこと。`goja.ConstructorCall` でコンストラクタ、`runtime.NewDynamicObject` でプロパティフック、`runtime.NewArrayBuffer([]byte)` で ArrayBuffer を作れる

---

### Task 1: イベントループ v2（タスクキュー + pending + ctx）

**Files:**
- Modify: `internal/js/eventloop.go`
- Test: `internal/js/eventloop_test.go`（追記）

**Interfaces:**
- Consumes: 既存 `eventLoop`（timers heap）、`VM.loop`
- Produces:
  - `(el *eventLoop) enqueueTask(fn func())` — ループスレッドで実行する関数を積む
  - `(el *eventLoop) addPending()` / `(el *eventLoop) donePending()` — 飛行中 I/O の増減
  - `(vm *VM) RunEventLoopContext(ctx context.Context) error` — タイマー・タスクを処理し、pending>0 の間は待機。ctx 打ち切りで `ctx.Err()` を返す
  - `(vm *VM) RunEventLoop()` — 互換維持: `RunEventLoopContext(context.Background())` のラッパー
  - `(vm *VM) LoopContext() context.Context` — 実行中ループの ctx（未実行時は `context.Background()`）。fetch/XHR の goroutine が HTTP リクエストに使う

- [ ] **Step 1: 失敗するテストを書く**

`internal/js/eventloop_test.go` に追記:

```go
func TestEventLoop_TaskQueue(t *testing.T) {
	vm := New(WithWriter(io.Discard))
	var order []string
	vm.loop.enqueueTask(func() { order = append(order, "task1") })
	vm.loop.enqueueTask(func() { order = append(order, "task2") })
	if err := vm.RunEventLoopContext(context.Background()); err != nil {
		t.Fatalf("RunEventLoopContext: %v", err)
	}
	if len(order) != 2 || order[0] != "task1" || order[1] != "task2" {
		t.Errorf("order = %v, want [task1 task2]", order)
	}
}

func TestEventLoop_WaitsForPending(t *testing.T) {
	vm := New(WithWriter(io.Discard))
	var got bool
	vm.loop.addPending()
	go func() {
		time.Sleep(50 * time.Millisecond)
		vm.loop.enqueueTask(func() { got = true })
		vm.loop.donePending()
	}()
	if err := vm.RunEventLoopContext(context.Background()); err != nil {
		t.Fatalf("RunEventLoopContext: %v", err)
	}
	if !got {
		t.Error("loop exited before pending async work completed")
	}
}

func TestEventLoop_ContextDeadline(t *testing.T) {
	vm := New(WithWriter(io.Discard))
	vm.loop.addPending() // 誰も donePending しない = ハングのシミュレーション
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := vm.RunEventLoopContext(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want DeadlineExceeded", err)
	}
}

func TestEventLoop_TasksAndTimersInterleave(t *testing.T) {
	// 検証意図: タスクは「まだ期限が来ていないタイマー」より先に処理される
	vm := New(WithWriter(io.Discard))
	_, _ = vm.Eval(`var __order__ = []; setTimeout(function() { __order__.push("timer"); }, 20);`)
	vm.loop.enqueueTask(func() {
		_, _ = vm.Eval(`__order__.push("task")`)
	})
	_ = vm.RunEventLoopContext(context.Background())
	got, _ := vm.Eval(`JSON.stringify(__order__)`)
	if got != `["task","timer"]` {
		t.Errorf("order = %v, want [task timer]", got)
	}
}
```

- [ ] **Step 2: テストが失敗することを確認**

Run: `go test ./internal/js/ -run TestEventLoop_ -v`
Expected: FAIL（`enqueueTask` / `RunEventLoopContext` 未定義のコンパイルエラー）

- [ ] **Step 3: 実装**

`internal/js/eventloop.go` の `eventLoop` を拡張:

```go
type eventLoop struct {
	mu      sync.Mutex
	timers  timerHeap
	nextID  int
	byID    map[int]*timerEntry
	tasks   []func()
	pending int
	wake    chan struct{} // buffered(1): タスク追加/pending減少の通知
}

func newEventLoop() *eventLoop {
	return &eventLoop{
		byID: make(map[int]*timerEntry),
		wake: make(chan struct{}, 1),
	}
}

func (el *eventLoop) signalWake() {
	select {
	case el.wake <- struct{}{}:
	default:
	}
}

// enqueueTask schedules fn to run on the loop thread.
func (el *eventLoop) enqueueTask(fn func()) {
	el.mu.Lock()
	el.tasks = append(el.tasks, fn)
	el.mu.Unlock()
	el.signalWake()
}

// addPending marks one in-flight async operation (e.g. an HTTP request).
func (el *eventLoop) addPending() {
	el.mu.Lock()
	el.pending++
	el.mu.Unlock()
}

// donePending marks one in-flight async operation as finished.
func (el *eventLoop) donePending() {
	el.mu.Lock()
	el.pending--
	el.mu.Unlock()
	el.signalWake()
}
```

`run` を ctx 対応に書き換え（既存の `run(runtime)` を置換）:

```go
// run processes tasks and timers until nothing remains and no async
// operation is in flight, or ctx is done.
func (el *eventLoop) run(ctx context.Context) error {
	for {
		// 1. Drain the task queue first (tasks beat not-yet-due timers).
		for {
			el.mu.Lock()
			if len(el.tasks) == 0 {
				el.mu.Unlock()
				break
			}
			task := el.tasks[0]
			el.tasks = el.tasks[1:]
			el.mu.Unlock()
			task()
		}

		// 2. Fire due timers (one at a time so new tasks get priority).
		el.mu.Lock()
		var wait time.Duration = -1
		if el.timers.Len() > 0 {
			entry := el.timers[0]
			if entry.cleared {
				heap.Pop(&el.timers)
				el.mu.Unlock()
				continue
			}
			now := time.Now()
			if !entry.fireAt.After(now) {
				heap.Pop(&el.timers)
				cb := entry.callback
				isInterval := entry.interval > 0
				el.mu.Unlock()
				_, _ = cb(goja.Undefined())
				if isInterval {
					el.mu.Lock()
					if !entry.cleared {
						entry.fireAt = time.Now().Add(entry.interval)
						heap.Push(&el.timers, entry)
					}
					el.mu.Unlock()
				}
				continue
			}
			wait = entry.fireAt.Sub(now)
		}
		hasWork := len(el.tasks) > 0
		pending := el.pending
		hasTimer := el.timers.Len() > 0
		el.mu.Unlock()

		if hasWork {
			continue
		}
		if !hasTimer && pending == 0 {
			return nil // 完了
		}

		// 3. Wait for: next timer, a wake signal, or cancellation.
		var timerC <-chan time.Time
		var timer *time.Timer
		if wait >= 0 {
			timer = time.NewTimer(wait)
			timerC = timer.C
		}
		select {
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return ctx.Err()
		case <-el.wake:
		case <-timerC:
		}
		if timer != nil {
			timer.Stop()
		}
	}
}
```

`VM` 側（`eventloop.go` 末尾の `RunEventLoop` を置換 + ctx 保持を追加。`vm.go` の struct に `loopCtx context.Context` フィールドを追加）:

```go
// RunEventLoopContext processes timers, tasks and in-flight async work
// until everything settles or ctx is done. Returns ctx.Err() on cancellation.
func (vm *VM) RunEventLoopContext(ctx context.Context) error {
	if vm.loop == nil {
		return nil
	}
	vm.loopCtx = ctx
	defer func() { vm.loopCtx = nil }()
	return vm.loop.run(ctx)
}

// RunEventLoop processes all pending timers and callbacks until the queue is empty.
func (vm *VM) RunEventLoop() {
	_ = vm.RunEventLoopContext(context.Background())
}

// LoopContext returns the context of the currently running event loop.
// Async bindings (fetch/XHR) use it for their HTTP requests.
func (vm *VM) LoopContext() context.Context {
	if vm.loopCtx != nil {
		return vm.loopCtx
	}
	return context.Background()
}
```

- [ ] **Step 4: テストが通ることを確認**

Run: `go test ./internal/js/ -race -v -run TestEventLoop_`
Expected: PASS（既存のタイマーテストも含め `go test ./internal/js/ -race` 全パス）

- [ ] **Step 5: WPT ランナー等の既存利用者が壊れていないことを確認**

Run: `go test ./internal/wpt/ ./internal/page/ ./internal/cdp/ -race`
Expected: PASS

- [ ] **Step 6: コミット**

```bash
git add internal/js/eventloop.go internal/js/eventloop_test.go internal/js/vm.go
git commit -m "feat(js): extend event loop with task queue, pending counter and context"
```

---

### Task 2: HTTPClient 型と URL 解決

**Files:**
- Create: `internal/js/http.go`
- Modify: `internal/js/vm.go`（フィールド追加）
- Test: `internal/js/http_test.go`

**Interfaces:**
- Produces:
  - `type HTTPRequest struct { Method, URL string; Headers map[string]string; Body []byte }`
  - `type HTTPResponse struct { Status int; StatusText string; Headers http.Header; Body []byte; FinalURL string }`
  - `type HTTPClient func(ctx context.Context, req HTTPRequest) (*HTTPResponse, error)`
  - `(vm *VM) SetHTTPClient(c HTTPClient)` / `(vm *VM) httpClient() HTTPClient`（未設定なら「network access not available」エラーを返すクライアント）
  - `(vm *VM) SetBaseURL(raw string)` / `(vm *VM) resolveURL(ref string) string` — 相対 URL をページ URL 基準で解決

- [ ] **Step 1: 失敗するテストを書く**

```go
func TestResolveURL(t *testing.T) {
	tests := []struct {
		name, base, ref, want string
	}{
		{"absolute untouched", "https://example.com/app/", "https://api.example.com/v1", "https://api.example.com/v1"},
		{"relative path", "https://example.com/app/index.html", "api/items", "https://example.com/app/api/items"},
		{"root relative", "https://example.com/app/index.html", "/v1/items", "https://example.com/v1/items"},
		{"no base", "", "api/items", "api/items"},
		{"scheme relative", "https://example.com/", "//cdn.example.com/a.json", "https://cdn.example.com/a.json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := New(WithWriter(io.Discard))
			if tt.base != "" {
				vm.SetBaseURL(tt.base)
			}
			if got := vm.resolveURL(tt.ref); got != tt.want {
				t.Errorf("resolveURL(%q) = %q, want %q", tt.ref, got, tt.want)
			}
		})
	}
}

func TestHTTPClientDefault(t *testing.T) {
	vm := New(WithWriter(io.Discard))
	_, err := vm.httpClient()(context.Background(), HTTPRequest{Method: "GET", URL: "https://example.com"})
	if err == nil {
		t.Error("expected error from default client, got nil")
	}
}
```

- [ ] **Step 2: 失敗確認** — Run: `go test ./internal/js/ -run 'TestResolveURL|TestHTTPClientDefault'` → コンパイルエラーで FAIL

- [ ] **Step 3: 実装**

`internal/js/http.go`:

```go
package js

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

// HTTPRequest is a network request issued by JS (fetch / XMLHttpRequest).
type HTTPRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    []byte
}

// HTTPResponse is the result of an HTTPRequest. Body is fully buffered,
// decompressed and decoded by the injected client.
type HTTPResponse struct {
	Status     int
	StatusText string
	Headers    http.Header
	Body       []byte
	FinalURL   string
}

// HTTPClient performs network requests on behalf of JS bindings.
// uzura applies NO same-origin policy / CORS checks: it is an agent
// browser and does not carry a human user's credentials across sites.
type HTTPClient func(ctx context.Context, req HTTPRequest) (*HTTPResponse, error)

// ErrNoHTTPClient is returned when JS attempts network access before a
// client has been injected (e.g. VM used standalone in tests).
var ErrNoHTTPClient = errors.New("js: network access not available (no HTTPClient injected)")

// SetHTTPClient injects the network client used by fetch and XMLHttpRequest.
func (vm *VM) SetHTTPClient(c HTTPClient) { vm.client = c }

func (vm *VM) httpClient() HTTPClient {
	if vm.client != nil {
		return vm.client
	}
	return func(context.Context, HTTPRequest) (*HTTPResponse, error) {
		return nil, ErrNoHTTPClient
	}
}

// SetBaseURL sets the document URL used to resolve relative request URLs.
func (vm *VM) SetBaseURL(raw string) {
	if u, err := url.Parse(raw); err == nil {
		vm.baseURL = u
	}
}

func (vm *VM) resolveURL(ref string) string {
	if vm.baseURL == nil {
		return ref
	}
	r, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return vm.baseURL.ResolveReference(r).String()
}
```

`vm.go` の `VM` struct に `client HTTPClient`、`baseURL *url.URL`、（Task 1 の）`loopCtx context.Context` を追加。

- [ ] **Step 4: パス確認** — Run: `go test ./internal/js/ -race` → PASS

- [ ] **Step 5: コミット**

```bash
git add internal/js/http.go internal/js/http_test.go internal/js/vm.go
git commit -m "feat(js): add HTTPClient injection point and URL resolution"
```

---

### Task 3: fetch / Headers / Response

**Files:**
- Create: `internal/js/binding_fetch.go`
- Test: `internal/js/binding_fetch_test.go`

**Interfaces:**
- Consumes: Task 1 の `enqueueTask/addPending/donePending/LoopContext`、Task 2 の `HTTPClient/resolveURL`
- Produces:
  - `BindFetch(vm *VM)` — グローバルに `fetch` と `Headers` を登録（`New()` の `init()` から呼ぶのではなく、page/テストが明示的に呼ぶ。ただし他バインディングと合わせて Task 9 で `BindNetwork` に集約）
  - JS 側: `fetch(url, init) → Promise<Response>`、`Response.{ok,status,statusText,url,redirected,headers,text(),json(),arrayBuffer()}`
  - Go 側: `makeResponseObject(vm *VM, resp *HTTPResponse) *goja.Object`（XHR では使わないが fetch 内部で使用）

- [ ] **Step 1: 失敗するテストを書く**

fake クライアントを共有ヘルパーとして定義:

```go
// fakeClient returns an HTTPClient serving canned responses keyed by URL.
func fakeClient(responses map[string]*HTTPResponse) HTTPClient {
	return func(_ context.Context, req HTTPRequest) (*HTTPResponse, error) {
		if resp, ok := responses[req.URL]; ok {
			return resp, nil
		}
		return nil, fmt.Errorf("no canned response for %s", req.URL)
	}
}

func newFetchVM(t *testing.T, responses map[string]*HTTPResponse) *VM {
	t.Helper()
	vm := New(WithWriter(io.Discard))
	vm.SetHTTPClient(fakeClient(responses))
	vm.SetBaseURL("https://example.com/")
	BindFetch(vm)
	return vm
}
```

テーブル駆動テスト（要点のみ列挙、全ケース実装すること）:

```go
func TestFetch_Basic(t *testing.T) {
	vm := newFetchVM(t, map[string]*HTTPResponse{
		"https://example.com/api": {
			Status: 200, StatusText: "OK",
			Headers: http.Header{"Content-Type": []string{"application/json"}},
			Body:    []byte(`{"title":"hello"}`), FinalURL: "https://example.com/api",
		},
	})
	_, err := vm.Eval(`
		var result = {};
		fetch("/api").then(function(res) {
			result.ok = res.ok;
			result.status = res.status;
			result.ctype = res.headers.get("content-type");
			return res.json();
		}).then(function(data) { result.title = data.title; });
	`)
	if err != nil {
		t.Fatal(err)
	}
	if err := vm.RunEventLoopContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, _ := vm.Eval(`JSON.stringify(result)`)
	want := `{"ok":true,"status":200,"ctype":"application/json","title":"hello"}`
	if got != want {
		t.Errorf("result = %s, want %s", got, want)
	}
}
```

追加ケース（同形式で書く）:
- `TestFetch_HTTPError`: status 404 → `ok === false`、Promise は **resolve** される（reject ではない）
- `TestFetch_NetworkError`: fake が error を返す → `.catch` に TypeError 相当が渡る
- `TestFetch_Concurrent`: `Promise.all([fetch("/a"), fetch("/b"), fetch("/c")])` が全部完了する
- `TestFetch_TextAndArrayBuffer`: `res.text()` の文字列、`res.arrayBuffer()` の `byteLength`
- `TestFetch_PostBody`: `fetch("/post", {method:"POST", headers:{"X-Token":"t1"}, body:"payload"})` — fake クライアント側で受信した `HTTPRequest.Method/Headers/Body` を記録して検証

- [ ] **Step 2: 失敗確認** — Run: `go test ./internal/js/ -run TestFetch_` → FAIL（BindFetch 未定義）

- [ ] **Step 3: 実装**

`internal/js/binding_fetch.go` の骨子:

```go
package js

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/dop251/goja"
)

// BindFetch registers the fetch() function and Headers class on the VM.
// No CORS / same-origin checks are applied (agent browser; see HTTPClient).
func BindFetch(vm *VM) {
	_ = vm.runtime.Set("fetch", vm.jsFetch)
	registerHeadersClass(vm)
}

func (vm *VM) jsFetch(call goja.FunctionCall) goja.Value {
	rt := vm.runtime
	promise, resolve, reject := rt.NewPromise()

	req, err := parseFetchArgs(vm, call)
	if err != nil {
		reject(rt.NewTypeError(err.Error()))
		return rt.ToValue(promise)
	}

	client := vm.httpClient()
	ctx := vm.LoopContext()
	vm.loop.addPending()
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				vm.loop.enqueueTask(func() {
					reject(rt.NewTypeError(fmt.Sprintf("fetch: %v", rec)))
				})
			}
			vm.loop.donePending()
		}()
		resp, err := client(ctx, req)
		vm.loop.enqueueTask(func() {
			if err != nil {
				reject(rt.NewTypeError("fetch failed: " + err.Error()))
				return
			}
			resolve(makeResponseObject(vm, resp))
		})
	}()

	return rt.ToValue(promise)
}
```

`parseFetchArgs`: 第1引数 string（`vm.resolveURL` で解決）、第2引数 object から `method`（既定 GET、大文字化）/ `headers`（object の全キー）/ `body`（string）を取り出す。未知のオプション（credentials, mode 等）は無視。

`makeResponseObject`: `rt.NewObject()` に以下を設定:
- `ok`（bool: 200-299）、`status`、`statusText`、`url`（FinalURL）、`redirected`（FinalURL != req.URL）
- `headers`: Headers インスタンス（`get(name)` は大文字小文字無視 = `http.Header.Get`、`has`、`forEach(cb)`）
- `text()`: 解決済み Promise で body 文字列を返す（`rt.NewPromise()` で作り即 resolve）
- `json()`: body を `JSON.parse` 相当で返す（実装は `rt.RunString` ではなく、`text()` と同様に promise を作り `goja` の JSON.parse を `rt.Get("JSON")` 経由で呼ぶか、単純に resolve 内で `rt.RunString("(" + ... )")` を避けて `json.Unmarshal` → `rt.ToValue`。**推奨: Go 側で `encoding/json` にアンマーシャルして `rt.ToValue(map/slice)`**。パース失敗時は reject）
- `arrayBuffer()`: `rt.NewArrayBuffer(body)` を resolve

`registerHeadersClass`: `new Headers(obj)` コンストラクタ（`goja.ConstructorCall`）。fetch init 用には内部で map を使うだけでよいが、`typeof Headers === 'function'` の feature detect を通すために登録する。

分量注意: Response/Headers 生成が長くなったら `binding_fetch_response.go` に分割してよい（300 行規約優先）。

- [ ] **Step 4: パス確認** — Run: `go test ./internal/js/ -race -run TestFetch_` → 全ケース PASS

- [ ] **Step 5: コミット**

```bash
git add internal/js/binding_fetch.go internal/js/binding_fetch_test.go
git commit -m "feat(js): add fetch binding with Response and Headers"
```

---

### Task 4: AbortController / AbortSignal

**Files:**
- Create: `internal/js/binding_abort.go`
- Modify: `internal/js/binding_fetch.go`（signal 対応）
- Test: `internal/js/binding_abort_test.go`

**Interfaces:**
- Produces: JS グローバル `AbortController`（`.signal`, `.abort()`）、`AbortSignal`（`.aborted`, `.addEventListener("abort", cb)`）。fetch init の `signal` を受理し、abort 時に飛行中リクエストをキャンセルして promise を reject（`err.name === "AbortError"` 相当のオブジェクト）
- 実装方式: AbortController ごとに `context.CancelFunc` ではなく **Go チャネル + フラグ**を持つ `abortState` 構造体。fetch は `signal` があれば `context.WithCancel(vm.LoopContext())` を作り、abort タスクで cancel を呼ぶ

- [ ] **Step 1: 失敗するテストを書く**

```go
func TestAbortController_AbortsFetch(t *testing.T) {
	block := make(chan struct{})
	vm := New(WithWriter(io.Discard))
	vm.SetHTTPClient(func(ctx context.Context, _ HTTPRequest) (*HTTPResponse, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-block:
			return &HTTPResponse{Status: 200}, nil
		}
	})
	vm.SetBaseURL("https://example.com/")
	BindFetch(vm)
	BindAbort(vm)
	defer close(block)

	_, err := vm.Eval(`
		var result = "";
		var ac = new AbortController();
		fetch("/slow", {signal: ac.signal}).catch(function(e) { result = e.name; });
		setTimeout(function() { ac.abort(); }, 20);
	`)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := vm.RunEventLoopContext(ctx); err != nil {
		t.Fatalf("loop should settle after abort, got %v", err)
	}
	got, _ := vm.Eval(`result`)
	if got != "AbortError" {
		t.Errorf("result = %v, want AbortError", got)
	}
}

func TestAbortSignal_AbortedFlag(t *testing.T) {
	vm := New(WithWriter(io.Discard))
	BindAbort(vm)
	got, err := vm.Eval(`
		var ac = new AbortController();
		var before = ac.signal.aborted;
		ac.abort();
		JSON.stringify([before, ac.signal.aborted]);
	`)
	if err != nil {
		t.Fatal(err)
	}
	if got != `[false,true]` {
		t.Errorf("got %v, want [false,true]", got)
	}
}
```

- [ ] **Step 2: 失敗確認** — Run: `go test ./internal/js/ -run TestAbort` → FAIL
- [ ] **Step 3: 実装** — `BindAbort(vm)` で `AbortController` コンストラクタ登録。signal オブジェクトに Go 側 `*abortState`（`aborted bool` / `cancels []context.CancelFunc` / `listeners []goja.Callable`）を `goja.Object.Set("__state__", ...)` ではなく Go 側 map（`vm` に `abortStates map[*goja.Object]*abortState`）で紐付けるか、`obj.Set` の隠しフィールドで持つ（どちらでも可、テストが通ること）。`abort()` はループスレッドで実行される（JS からしか呼ばれない）ので、フラグ設定 → cancel 群呼び出し → listener 呼び出しを直接行ってよい。fetch 側: `parseFetchArgs` で signal を受け取り、`context.WithCancel(vm.LoopContext())` を作って cancel を signal に登録、goroutine はその ctx を使う。reject 時のエラーオブジェクトは `rt.NewObject()` に `name: "AbortError"`, `message: "The operation was aborted"` を設定
- [ ] **Step 4: パス確認** — Run: `go test ./internal/js/ -race -run 'TestAbort|TestFetch_'` → PASS
- [ ] **Step 5: コミット**

```bash
git add internal/js/binding_abort.go internal/js/binding_abort_test.go internal/js/binding_fetch.go
git commit -m "feat(js): add AbortController with fetch cancellation"
```

---

### Task 5: XMLHttpRequest

**Files:**
- Create: `internal/js/binding_xhr.go`
- Test: `internal/js/binding_xhr_test.go`

**Interfaces:**
- Consumes: Task 1/2 の loop・HTTPClient
- Produces: JS グローバル `XMLHttpRequest`。`open(method, url[, async])`（async=false は例外）、`setRequestHeader`、`send([body])`、`abort()`、`getResponseHeader(name)`、`getAllResponseHeaders()`、`readyState`（0-4 定数 UNSENT/OPENED/HEADERS_RECEIVED/LOADING/DONE 付き）、`status`/`statusText`/`responseText`/`response`/`responseType`（''/'text'/'json'）、`timeout`、イベント: `onreadystatechange`/`onload`/`onerror`/`ontimeout`/`onabort` + `addEventListener(type, cb)`

- [ ] **Step 1: 失敗するテストを書く**（テーブル駆動。fake クライアントは Task 3 のものを再利用）

必須ケース:

```go
func TestXHR_BasicGet(t *testing.T) {
	// open→send→DONE で responseText が取れ、readystatechange が
	// OPENED(1)→HEADERS_RECEIVED(2)→LOADING(3)→DONE(4) の順で発火する
	vm := newFetchVM(t, map[string]*HTTPResponse{
		"https://example.com/api": {Status: 200, StatusText: "OK",
			Headers: http.Header{"Content-Type": []string{"text/plain"}},
			Body:    []byte("hello"), FinalURL: "https://example.com/api"},
	})
	BindXHR(vm)
	_, err := vm.Eval(`
		var states = [], text = "", loaded = false;
		var xhr = new XMLHttpRequest();
		xhr.onreadystatechange = function() { states.push(xhr.readyState); };
		xhr.onload = function() { loaded = true; text = xhr.responseText; };
		xhr.open("GET", "/api");
		xhr.send();
	`)
	if err != nil {
		t.Fatal(err)
	}
	_ = vm.RunEventLoopContext(context.Background())
	got, _ := vm.Eval(`JSON.stringify({states: states, text: text, loaded: loaded, status: xhr.status})`)
	want := `{"states":[1,2,3,4],"text":"hello","loaded":true,"status":200}`
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
```

追加ケース:
- `TestXHR_JSONResponseType`: `responseType="json"` → `xhr.response` がオブジェクト
- `TestXHR_NetworkError`: onerror が呼ばれ readyState==4, status==0
- `TestXHR_SyncThrows`: `open("GET", "/api", false)` が例外
- `TestXHR_PostBodyAndHeaders`: method/ヘッダー/body が HTTPRequest に載る（fake で記録）
- `TestXHR_GetAllResponseHeaders`: `"content-type: text/plain\r\n"` 形式を含む
- `TestXHR_Timeout`: `xhr.timeout = 20` + 遅い fake → ontimeout 発火

- [ ] **Step 2: 失敗確認** — Run: `go test ./internal/js/ -run TestXHR_` → FAIL
- [ ] **Step 3: 実装** — `BindXHR(vm)`: `rt.Set("XMLHttpRequest", constructor)`。コンストラクタは `goja.ConstructorCall` を受け、`call.This` に全プロパティとメソッドを設定。Go 側状態は closure でよい（`type xhrState struct { method, url string; headers map[string]string; readyState int; ... }`）。`send()`: fetch と同じ pending/goroutine/タスク方式で、応答受信タスク内で readyState を 2→3→4 と遷移させつつ `onreadystatechange` を各段階で呼び、最後に `onload`（エラー時 `onerror`、タイムアウト時 `ontimeout`）。timeout は goroutine 内で `context.WithTimeout`。`addEventListener(type, cb)` は type→[]callable の map に積み、on* と両方呼ぶ。readystatechange 遷移の 2/3/4 は**同一タスク内で連続実行**でよい（実ブラウザの分割配送は模倣しない）
- [ ] **Step 4: パス確認** — Run: `go test ./internal/js/ -race -run TestXHR_` → PASS
- [ ] **Step 5: コミット**

```bash
git add internal/js/binding_xhr.go internal/js/binding_xhr_test.go
git commit -m "feat(js): add XMLHttpRequest binding"
```

---

### Task 6: localStorage / sessionStorage

**Files:**
- Create: `internal/js/binding_storage.go`
- Test: `internal/js/binding_storage_test.go`

**Interfaces:**
- Produces:
  - `type Storage interface { GetItem(key string) (string, bool); SetItem(key, value string); RemoveItem(key string); Clear(); Key(n int) (string, bool); Len() int }`
  - `func NewMemStorage() Storage` — 挿入順を保持するメモリ実装（map + keys slice）
  - `func BindStorage(vm *VM, local, session Storage)` — `localStorage` / `sessionStorage` を goja の DynamicObject として登録（メソッド + プロパティアクセス両対応）

- [ ] **Step 1: 失敗するテストを書く**

```go
func TestStorage_MethodsAndProperties(t *testing.T) {
	vm := New(WithWriter(io.Discard))
	BindStorage(vm, NewMemStorage(), NewMemStorage())
	got, err := vm.Eval(`
		localStorage.setItem("a", "1");
		localStorage.b = "2";                    // プロパティ書き込み
		var r = [];
		r.push(localStorage.getItem("a"));       // "1"
		r.push(localStorage.a);                  // "1"  プロパティ読み出し
		r.push(localStorage.getItem("b"));       // "2"
		r.push(localStorage.length);             // 2
		r.push(localStorage.key(0));             // "a"
		r.push(localStorage.getItem("missing")); // null
		localStorage.removeItem("a");
		r.push(localStorage.length);             // 1
		localStorage.clear();
		r.push(localStorage.length);             // 0
		sessionStorage.setItem("s", "x");
		r.push(localStorage.getItem("s"));       // null (分離)
		JSON.stringify(r);
	`)
	if err != nil {
		t.Fatal(err)
	}
	want := `["1","1","2",2,"a",null,1,0,null]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
```

- [ ] **Step 2: 失敗確認** — Run: `go test ./internal/js/ -run TestStorage_` → FAIL
- [ ] **Step 3: 実装** — `memStorage`: `data map[string]string` + `keys []string`（SetItem は既存キーなら順序維持）。DynamicObject 実装 `storageObject`: `Get(key)` はまずメソッド名（getItem/setItem/removeItem/clear/key/length）を返し、次に data。`Set(key, val)` は `SetItem(key, val.String())`。`Has`/`Delete`/`Keys` も委譲。`length` は `Get("length")` で `rt.ToValue(s.Len())` を返す（DynamicObject では accessor にできないため Get フックで対応）。`getItem` 未存在キーは `goja.Null()`
- [ ] **Step 4: パス確認** — Run: `go test ./internal/js/ -race -run TestStorage_` → PASS
- [ ] **Step 5: コミット**

```bash
git add internal/js/binding_storage.go internal/js/binding_storage_test.go
git commit -m "feat(js): add localStorage/sessionStorage with property access"
```

---

### Task 7: location / history / window イベント

**Files:**
- Create: `internal/js/binding_location.go`
- Test: `internal/js/binding_location_test.go`

**Interfaces:**
- Consumes: Task 1 の `enqueueTask`（popstate の非同期発火）
- Produces:
  - `func BindLocation(vm *VM, rawURL string)` — `location`（= `window.location` / `document.location`）、`history`、window レベルの `addEventListener`/`removeEventListener` を登録。内部で `vm.SetBaseURL(rawURL)` も呼ぶ
  - location: `href/origin/protocol/host/hostname/port/pathname/search/hash` getter + `toString()`。`hash` のみ setter（変更時に `hashchange` 発火）。`assign`/`replace`/`reload` は console.warn の no-op
  - history: `pushState(state, unused, url)` / `replaceState` / `back()` / `forward()` / `go(n)` / `state` / `length`。移動時に `popstate` イベント（`.state` 付き）をタスクキュー経由で発火
  - window イベント: `addEventListener(type, cb)` を **グローバル関数**として登録（`window === globalThis` のため）。`onpopstate` / `onhashchange` プロパティも参照する

- [ ] **Step 1: 失敗するテストを書く**

```go
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
```

- [ ] **Step 2: 失敗確認** — Run: `go test ./internal/js/ -run 'TestLocation_|TestHistory_'` → FAIL
- [ ] **Step 3: 実装** — 状態: `type locationState struct { u *url.URL; stack []historyEntry; idx int; listeners map[string][]goja.Callable }`（`historyEntry{state goja.Value; rawURL string}`、初期スタックは現在 URL 1 件）。location オブジェクトは `rt.NewObject()` + `DefineAccessorProperty` で各 getter（`u` から都度計算）と hash setter を定義。`document.location` にも同じオブジェクトを設定（`rt.Get("document")` が object なら `Set("location", ...)`）。pushState: URL を resolve して `u` を差し替え、スタックの idx 以降を捨てて追加。back/forward/go: idx を範囲内で移動し `u` を差し替え、`enqueueTask` で popstate 発火（イベントオブジェクトは `rt.NewObject()` に `type`, `state` を設定）。window の `addEventListener` はグローバル関数として `rt.Set("addEventListener", ...)` で登録し、type が popstate/hashchange 以外は黙って無視（既存 document の addEventListener とは独立）。発火時は listeners + `onpopstate`/`onhashchange`（グローバル変数として `rt.Get` で取得し callable なら呼ぶ）
- [ ] **Step 4: パス確認** — Run: `go test ./internal/js/ -race -run 'TestLocation_|TestHistory_'` → PASS
- [ ] **Step 5: コミット**

```bash
git add internal/js/binding_location.go internal/js/binding_location_test.go
git commit -m "feat(js): add location, history and window popstate/hashchange"
```

---

### Task 8: network.Fetcher の任意メソッド対応（FetchRequest）

**Files:**
- Modify: `internal/network/fetcher.go`
- Test: `internal/network/fetcher_test.go`（追記）

**Interfaces:**
- Consumes: 既存 `doFetch`（GET 固定・ブラウザ風ヘッダー・リトライ）
- Produces: `(f *Fetcher) FetchRequest(ctx context.Context, method, url string, extraHeaders http.Header, body []byte) (*http.Response, error)` — Cookie jar・UA・TLS 設定を共有しつつ任意メソッド + ボディを送る。GET と同じくブラウザ風の既定ヘッダーを付与するが、`Sec-Fetch-Dest: empty` / `Sec-Fetch-Mode: cors`（サブリソース風）にする。リトライは冪等でない可能性があるため **method が GET/HEAD のときだけ**適用

- [ ] **Step 1: 失敗するテストを書く**

```go
func TestFetchRequest_PostBody(t *testing.T) {
	var gotMethod, gotBody, gotCT string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		gotCT = r.Header.Get("Content-Type")
		w.WriteHeader(200)
	}))
	defer ts.Close()

	f := New(FetcherOptions{})
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	resp, err := f.FetchRequest(context.Background(), "POST", ts.URL, headers, []byte(`{"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if gotMethod != "POST" || gotBody != `{"a":1}` || gotCT != "application/json" {
		t.Errorf("got method=%s body=%s ct=%s", gotMethod, gotBody, gotCT)
	}
}
```

（`New(FetcherOptions{})` のシグネチャは `fetcher.go` の既存コンストラクタに合わせること。既存テストの生成方法をコピーする）

- [ ] **Step 2: 失敗確認** — Run: `go test ./internal/network/ -run TestFetchRequest_` → FAIL
- [ ] **Step 3: 実装** — `doFetch` を `doFetchMethod(ctx, method, url, extraHeaders, body)` に一般化（`doFetch` は GET で委譲）。`FetchRequest` は GET/HEAD のみ既存のリトライループを通し、それ以外は 1 回実行
- [ ] **Step 4: パス確認** — Run: `go test ./internal/network/ -race` → PASS（既存テスト含む）
- [ ] **Step 5: コミット**

```bash
git add internal/network/fetcher.go internal/network/fetcher_test.go
git commit -m "feat(network): add FetchRequest for arbitrary methods and bodies"
```

---

### Task 9: page 統合（adapter + Navigate への JS 実行組み込み）

**Files:**
- Create: `internal/page/jsclient.go`
- Modify: `internal/page/navigate_helpers.go`（ドキュメント確定後に JS 実行を追加）
- Modify: `internal/page/page.go`（`VM()` のバインド一式を共通化）
- Test: `internal/page/jsclient_test.go`

**Interfaces:**
- Consumes: Task 1-8 のすべて。`page.Page` の `fetcher`/`url`/`doc`/`vm`
- Produces:
  - `(p *Page) jsHTTPClient() js.HTTPClient` — `network.Fetcher.FetchRequest` を呼び、`network.DecompressResponse` で展開してから `js.HTTPResponse` に詰める adapter
  - Navigate 末尾（doc/url 確定後）: fresh VM を作成し `BindDocument` → `SetHTTPClient` → `BindFetch/BindAbort/BindXHR` → `BindStorage` → `BindLocation` → `js.ExecuteScripts` → `vm.RunEventLoopContext(ctx)` を実行する非公開メソッド `(p *Page) runScripts(ctx context.Context)`。スクリプトエラー・ループの ctx エラーはログのみで Navigate は成功扱い（部分結果方針）
  - storage は Page フィールド（`localStore, sessionStore js.Storage`）として保持し、**ナビゲーションをまたいで維持**する（Page 生成時に `js.NewMemStorage()`）

- [ ] **Step 1: 失敗する統合テストを書く**

`internal/page/jsclient_test.go`:

```go
// CSR を模したページ: 初期 HTML は空の #root だけで、スクリプトが
// fetch した JSON からタイトルを DOM に構築する。
func TestNavigate_ExecutesScriptsWithFetch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/post", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"title":"CSR Title","body":"loaded via fetch"}`)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html><html><body><div id="root">Loading...</div>
<script>
fetch("/api/post").then(function(r){ return r.json(); }).then(function(data){
  var h1 = document.createElement("h1");
  h1.textContent = data.title;
  document.getElementById("root").textContent = "";
  document.getElementById("root").appendChild(h1);
  var p = document.createElement("p");
  p.textContent = data.body;
  document.getElementById("root").appendChild(p);
});
</script></body></html>`)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	p := New(nil) // 既存テストの Page 生成方法に合わせる
	defer p.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := p.Navigate(ctx, ts.URL); err != nil {
		t.Fatal(err)
	}

	h1 := p.Document().QuerySelector("h1")
	if h1 == nil {
		t.Fatal("h1 not found: scripts did not run or fetch did not complete")
	}
	if h1.TextContent() != "CSR Title" {
		t.Errorf("h1 = %q, want %q", h1.TextContent(), "CSR Title")
	}
	if strings.Contains(p.Document().Body().TextContent(), "Loading...") {
		t.Error("Loading placeholder still present")
	}
}

// XHR 版（axios 相当のパス）。サーバー構成は上と同じで <script> のみ差し替え:
func TestNavigate_ExecutesScriptsWithXHR(t *testing.T) {
	// HTML の script 部分:
	//   <script>
	//   var xhr = new XMLHttpRequest();
	//   xhr.responseType = "json";
	//   xhr.onload = function() {
	//     var h1 = document.createElement("h1");
	//     h1.textContent = xhr.response.title;
	//     document.getElementById("root").textContent = "";
	//     document.getElementById("root").appendChild(h1);
	//   };
	//   xhr.open("GET", "/api/post");
	//   xhr.send();
	//   </script>
	// 検証: fetch 版と同じく QuerySelector("h1").TextContent() == "CSR Title"
	// （テスト本体は TestNavigate_ExecutesScriptsWithFetch をコピーして
	//   HTML 文字列と関数名だけ変える）
}

// localStorage / history / location がスクリプトから見えることのスモークテスト
func TestNavigate_WebAPIsAvailable(t *testing.T) {
	// HTML:
	//   <div id="out"></div>
	//   <script>
	//   localStorage.setItem("k", "v1");
	//   history.pushState({}, "", "/pushed");
	//   document.getElementById("out").textContent =
	//     [localStorage.getItem("k"), location.pathname, typeof fetch].join("|");
	//   </script>
	// 検証: Navigate 後に #out の textContent が "v1|/pushed|function"
}
```

（`p := New(nil)` の部分は `page_test.go` の既存 Page 生成コードをコピーして合わせること。`Document()` / `QuerySelector` の正確なメソッド名も既存テストからコピーする）

- [ ] **Step 2: 失敗確認** — Run: `go test ./internal/page/ -run TestNavigate_Executes -v` → FAIL（h1 not found: スクリプト未実行のため）

- [ ] **Step 3: adapter と runScripts を実装**

`internal/page/jsclient.go`:

```go
package page

import (
	"context"
	"io"
	"net/http"

	"github.com/nyasuto/uzura/internal/js"
	"github.com/nyasuto/uzura/internal/network"
)

// jsHTTPClient adapts the page's Fetcher for JS-initiated requests
// (fetch / XMLHttpRequest). Cookie jar, UA, compression handling and
// robots policy are shared with page navigation. No CORS checks are
// applied: uzura is an agent browser and does not carry a human user's
// cross-site credentials.
func (p *Page) jsHTTPClient() js.HTTPClient {
	return func(ctx context.Context, req js.HTTPRequest) (*js.HTTPResponse, error) {
		headers := http.Header{}
		for k, v := range req.Headers {
			headers.Set(k, v)
		}
		resp, err := p.fetcher.FetchRequest(ctx, req.Method, req.URL, headers, req.Body)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if err := network.DecompressResponse(resp); err != nil {
			return nil, err
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxJSResponseBytes))
		if err != nil {
			return nil, err
		}
		finalURL := req.URL
		if resp.Request != nil && resp.Request.URL != nil {
			finalURL = resp.Request.URL.String()
		}
		return &js.HTTPResponse{
			Status:     resp.StatusCode,
			StatusText: http.StatusText(resp.StatusCode),
			Headers:    resp.Header,
			Body:       body,
			FinalURL:   finalURL,
		}, nil
	}
}

// maxJSResponseBytes caps a single JS-initiated response body (10 MB).
const maxJSResponseBytes = 10 << 20

// runScripts executes the document's inline scripts and drives the event
// loop until all JS-initiated work (fetch/XHR/timers) settles or ctx is
// done. Script errors never fail navigation (partial-result policy).
func (p *Page) runScripts(ctx context.Context) {
	p.mu.Lock()
	doc := p.doc
	pageURL := p.url
	if p.localStore == nil {
		p.localStore = js.NewMemStorage()
		p.sessionStore = js.NewMemStorage()
	}
	local, session := p.localStore, p.sessionStore
	vm := js.New(p.vmOptions...)
	p.vm = vm
	p.mu.Unlock()
	if doc == nil {
		return
	}

	js.BindDocument(vm, doc)
	vm.SetHTTPClient(p.jsHTTPClient())
	js.BindFetch(vm)
	js.BindAbort(vm)
	js.BindXHR(vm)
	js.BindStorage(vm, local, session)
	js.BindLocation(vm, pageURL)

	_ = js.ExecuteScripts(vm, doc)
	_ = vm.RunEventLoopContext(ctx)
}
```

`navigate_helpers.go`: Navigate がドキュメントと URL を確定させた直後（return 前）に `p.runScripts(ctx)` を追加。`page.go` の `VM()` は「nil なら runScripts と同じバインドで生成」に揃える（バインド部を共通ヘルパー `bindAll(vm, doc, ...)` に抽出してよい）。`Page` struct に `localStore, sessionStore js.Storage` を追加。

- [ ] **Step 4: 統合テストのパス確認** — Run: `go test ./internal/page/ -race -run TestNavigate_ -v` → PASS

- [ ] **Step 5: 全パッケージの回帰確認**

Run: `go test ./... -race`
Expected: PASS。**特に注意**: `internal/mcp`（browse が JS 実行するようになる）と `internal/cdp`（Page.navigate 経由）のテストページに `<script>` が含まれる場合、挙動が変わる。落ちたテストは「JS 実行後の DOM」を前提に期待値を更新する（script/style 除去は text 出力側で既に行われるため、多くは影響なしの見込み）。

- [ ] **Step 6: コミット**

```bash
git add internal/page/ internal/js/
git commit -m "feat(page): execute scripts with web APIs during navigation"
```

---

### Task 10: compat 検証・ドキュメント・PR

**Files:**
- Modify: `internal/mcp/compat_test.go`（Qiita 追加）
- Modify: `README.md`（対応 API と CORS 非適用の明記）
- Test: 全テスト + compat

**Interfaces:** なし（仕上げタスク）

- [ ] **Step 1: compat テストに CSR サイトを追加**

`compat_test.go` の対象サイトに Qiita を追加（既存サイトの定義形式をコピー）:

```go
{name: "Qiita", url: "https://qiita.com/", checkH1: true},
```

（フィールド名・構造は既存の 5 サイト定義に合わせる。h1 が取れない場合でも browse text が「Loading」だけでないことを最低基準とする）

- [ ] **Step 2: compat 実行**

Run: `go test -tags compat -run TestCompat ./internal/mcp/ -v -count=1`
Expected: 既存 5 サイト全パス + Qiita で結果を記録（**パスしなくても即失敗とせず**、出力を PR に記載して判断材料にする。JS 実行によって既存サイトが悪化していないことが必須条件）

- [ ] **Step 3: README 更新**

「特徴」の JavaScript 実行の行を更新し、以下を追記:

```markdown
- **JavaScript 実行** — goja による ES6 対応の JS エンジン内蔵（fetch / XMLHttpRequest / localStorage / location / history 対応、CSR サイトのレンダリング可）

> **注意**: uzura は AI エージェント用途に特化しているため、JS からのネットワークリクエストに CORS（同一オリジンポリシー）を適用しません。
```

- [ ] **Step 4: 最終確認とコミット**

```bash
go test ./... -race && make bench 2>/dev/null | tail -5   # ベンチ回帰の目視確認
git add internal/mcp/compat_test.go README.md
git commit -m "test(mcp): add CSR site to compat suite and document web APIs"
```

- [ ] **Step 5: PR 作成**

```bash
git push -u origin feat/js-web-apis
gh pr create --title "feat(js): fetch / XHR / storage / location / history バインディングと Navigate への JS 実行組み込み" --body "(スペックと計画へのリンク、Closes #32、compat 結果を記載)"
```

PR 本文に必ず含める: `Closes #32`、スペック/計画ファイルへのリンク、compat テストの before/after、CORS 非適用の明記、末尾に

```
🤖 Generated with [Claude Code](https://claude.com/claude-code)

https://claude.ai/code/session_01YYp6mqV9NHcfY7YaJtTGrJ
```
