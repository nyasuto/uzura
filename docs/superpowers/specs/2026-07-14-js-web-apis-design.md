# JS Web API バインディング設計（fetch / XHR / storage / location / history）

- イシュー: [#32](https://github.com/nyasuto/uzura/issues/32)
- 日付: 2026-07-14
- ステータス: 承認済み

## 目的と成功基準

CSR サイト（Qiita、NHK 等）が `browse` / `query` で本文・h1 を返せるようにする。
実サイト互換ドリブン: 仕様準拠より「compat テスト対象の CSR サイトで動く」ことを合格基準とする。

- axios は内部で XMLHttpRequest を使うため、fetch と XHR の**両方**が必須
- SPA ルーターには history に加えて未実装の `window.location` が必要
- 合格判定: `-tags compat` に CSR サイト（Qiita 等）を追加し、query h1 / browse markdown が非空になること

## 決定事項

| 論点 | 決定 |
|------|------|
| スコープ | fetch + XHR + localStorage/sessionStorage + location + history を 1 フェーズで、各 API は最小忠実度 |
| CORS | **適用しない**（全リクエスト許可）。AI エージェント専用でユーザー認証情報を使い回さないため。godoc と README に明記 |
| 実行モデル | 自前イベントループ拡張（タスクキュー + 飛行中カウンター + ctx デッドライン） |
| storage 分離 | MVP は localStorage / sessionStorage とも **Page 単位**。インターフェース分離し将来 BrowserContext+origin へ移行可能にする |
| sync XHR | 非対応（`open(..., false)` は例外） |

## アーキテクチャ

### 依存方向

`internal/js` は `internal/network` を import しない。js 側に最小 HTTP インターフェースを定義し、page が adapter を注入する:

```go
// internal/js/http.go
type HTTPRequest  struct { Method, URL string; Headers map[string]string; Body []byte }
type HTTPResponse struct { Status int; StatusText string; Headers http.Header; Body []byte; FinalURL string }
type HTTPClient   func(ctx context.Context, req HTTPRequest) (*HTTPResponse, error)
```

- adapter（`internal/page/jsclient.go`）は `network.Fetcher` をラップし、Cookie jar・圧縮展開・エンコーディング変換・レスポンスサイズ制限・robots.txt 設定をページナビゲーションと共有する（同一 Fetcher を使うため自動的に同じポリシーが適用される）
- 相対 URL はページ URL 基準で解決する
- テストでは fake HTTPClient を注入できる

### イベントループ v2（internal/js/eventloop.go 拡張）

- 既存: タイマー heap のみ、「heap が空になったら run() 終了」
- 拡張: 汎用タスクキュー + 飛行中（pending）カウンター + ctx デッドライン
  - 終了条件: タイマーなし && タスクなし && pending == 0
  - pending > 0 の間はタスク到着か ctx.Done() まで待機
  - `RunEventLoop()` は互換維持。`RunEventLoopContext(ctx)` を新設し `page.Navigate` の ctx を貫通
- fetch/XHR の流れ: JS スレッドで pending++ → goroutine で HTTP I/O → 完了時にタスクを enqueue（Promise 解決 / XHR イベント発火）→ pending--
- **JS の実行は常にループスレッドのみ**（goja の単一スレッド制約を維持）。goroutine から goja に触れない

## API サーフェス（MVP）

### fetch（binding_fetch.go）

- `fetch(url, init)` → Promise\<Response\>。init: `method` / `headers` / `body` / `signal`。`credentials` / `mode` 等は受理して無視
- `Response`: `ok` / `status` / `statusText` / `url` / `redirected` / `headers` / `text()` / `json()` / `arrayBuffer()`
- `Headers`: `get` / `has` / `forEach` / `entries`（最小実装）
- `AbortController` / `AbortSignal`: **実装する**。abort() は context cancel に接続（axios / React Query が多用するため）
- 非対応: streams（`response.body`）、`FormData` / `Blob`、`clone()`

### XMLHttpRequest（binding_xhr.go）

- readyState マシン（UNSENT=0 〜 DONE=4）
- `open` / `send` / `abort` / `setRequestHeader` / `getResponseHeader` / `getAllResponseHeaders`
- イベント: `onreadystatechange` / `onload` / `onerror` / `ontimeout` + `addEventListener`
- `responseType`: `''` / `'text'` / `'json'`、`responseText` / `response` / `status` / `statusText`、`timeout` プロパティ
- `open(method, url, false)`（sync）は例外を投げる

### localStorage / sessionStorage（binding_storage.go）

- `getItem` / `setItem` / `removeItem` / `clear` / `key` / `length`
- **プロパティアクセス対応**（`localStorage.foo = "x"`）: goja の DynamicObject で実装
- 実体は Go 側の map。Storage インターフェースを分離し Page が注入（将来 Context+origin 単位に差し替え可能）

### location / history（binding_location.go）

- `Location`: `href` / `origin` / `protocol` / `host` / `hostname` / `port` / `pathname` / `search` / `hash` の読み取り + `toString()`
  - `hash` setter のみ書き込み可（`hashchange` イベントを発火）
  - `href` 代入・`assign` / `replace` / `reload` は console.warn 付き no-op（実ナビゲーションはスコープ外）
- `History`: `pushState` / `replaceState` / `back` / `forward` / `go` / `state` / `length`
  - スタックを管理し location と同期。`back` / `forward` / `go` は `popstate` をタスクキュー経由で非同期発火

## データフロー（CSR サイトの典型）

```
page.Navigate
  → HTML パース → BindDocument / BindNetwork / BindStorage / BindLocation
  → ExecuteScripts（React 初期化、fetch/XHR 発行 = pending++）
  → RunEventLoopContext(ctx)（レスポンス待機 → タスクで Promise 解決 → JS が DOM 構築）
  → ループ終了（pending == 0）
  → 既存の text / markdown / semantic 出力が JS 実行後の DOM を反映
```

MCP の browse / query / semantic_tree は変更なしで CSR 対応になる。

## エラー処理

- ネットワークエラー → fetch は TypeError で reject、XHR は error イベント。**Go 側にエラーを返さない**（1 リクエストの失敗でページ全体を壊さない）
- ctx デッドライン到達 → 飛行中リクエストを中断してループを抜け、その時点の DOM で続行（Phase 15.3 の部分結果方針と整合）
- goroutine 内の panic は recover してリクエスト失敗に変換（プロジェクト方針: Errors returned, never panic）

## テスト戦略

- ユニット（テーブル駆動、httptest.Server）:
  - fetch: text / json / HTTP エラー / ネットワークエラー / `Promise.all` 並行 / タイマーとの順序
  - XHR: readyState 遷移、イベント発火順、timeout、abort
  - storage: メソッド + プロパティアクセス、clear/key/length
  - location/history: pushState → location 反映、popstate / hashchange 発火
  - イベントループ: pending 待機、ctx デッドライン、タスクとタイマーの混在
- 統合: fetch で JSON を取得して DOM を構築する模擬 CSR ページ → `page.Navigate` → querySelector で h1 検証
- 実サイト（`-tags compat`）: Qiita（XHR/axios 系）を対象に追加し、query h1 / browse markdown 非空を確認

## スコープ外

CORS、streams、FormData / Blob / File、WebSocket、`document.cookie`、sync XHR、location 代入による実ナビゲーション、Service Worker、storage のオリジン分離・永続化。

## ファイル構成（1 ファイル 300 行規約）

| ファイル | 内容 |
|---------|------|
| `internal/js/eventloop.go` | タスクキュー + pending + ctx 対応の拡張 |
| `internal/js/http.go` | HTTPClient インターフェースと型 |
| `internal/js/binding_fetch.go` | fetch / Response / Headers / AbortController |
| `internal/js/binding_xhr.go` | XMLHttpRequest |
| `internal/js/binding_storage.go` | Storage + DynamicObject |
| `internal/js/binding_location.go` | Location + History |
| `internal/page/jsclient.go` | network.Fetcher → js.HTTPClient adapter と配線 |

各ファイルに対応する `_test.go` を用意する。
