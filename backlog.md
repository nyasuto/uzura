# Uzura — Backlog

フェーズ完了時、次フェーズの内容を tasks.md にコピーしてループを再開する。

---

## Phase 2: HTTP Fetcher + Document Loading

### Task 2.1: Basic HTTP Fetcher
- [ ] `internal/network/fetcher.go`
- [ ] `Fetch(url string) (*Response, error)` タイムアウト付き
- [ ] User-Agent設定、リダイレクト追跡（最大10回）
- [ ] `net/http/httptest` でテスト

### Task 2.2: Content-Type and Encoding
- [ ] Content-Typeヘッダーからcharset検出
- [ ] `<meta charset>` タグからのcharset検出
- [ ] `golang.org/x/text/encoding` でUTF-8変換
- [ ] テスト: Shift-JIS, EUC-JP, ISO-8859-1

### Task 2.3: Cookie Jar
- [ ] `net/http/cookiejar` 統合
- [ ] セッション内のCookie永続化
- [ ] テスト: set/get, リダイレクト跨ぎ

### Task 2.4: Document Loading Pipeline
- [ ] Fetch → Decode → Parse → Document パイプライン
- [ ] エラーハンドリング（ネットワーク、タイムアウト、不正HTML）
- [ ] httptest での結合テスト

### Task 2.5: CLI `fetch` コマンド
- [ ] `uzura fetch <url>`
- [ ] `--format`, `--timeout`, `--user-agent` フラグ
- [ ] 動作確認: `uzura fetch https://example.com`

### Task 2.6: robots.txt
- [ ] `--obey-robots` フラグ
- [ ] robots.txtパースとallow/disallowチェック

### Task 2.7: Phase 2 検証
- [ ] テスト全パス、日本語サイトのエンコーディング確認

---

## Phase 3: CSS Selector Engine

### Task 3.1: cascadia統合
- [ ] `github.com/andybalholm/cascadia` 依存追加
- [ ] `internal/css/selector.go` アダプター

### Task 3.2: querySelector / querySelectorAll
- [ ] Element/DocumentにquerySelector/querySelectorAllを追加
- [ ] テスト: tag, class, id, 属性, 結合子, 擬似クラス

### Task 3.3: matches / closest
- [ ] `Element.Matches()`, `Element.Closest()`

### Task 3.4: Phase 3 検証
- [ ] 複雑セレクターのテスト、大規模DOMでのベンチマーク

---

## Phase 4: DOM API（Web標準準拠）

- [ ] Node: appendChild, removeChild, insertBefore, cloneNode（Phase 1で基本実装済み、拡張）
- [ ] Element: classList, dataset, innerHTML setter（パーサー連携）
- [ ] Document: createDocumentFragment, importNode
- [ ] MutationObserver 基本実装
- [ ] WPT `dom/nodes/` テストのパス

---

## Phase 5: JavaScript Execution（Goja統合）

- [ ] `github.com/dop251/goja` 依存追加
- [ ] Goja VMの初期化とサンドボックス
- [ ] `document` オブジェクトのJSバインディング
- [ ] `window`, `globalThis`, `console` の実装
- [ ] `setTimeout` / `setInterval`
- [ ] addEventListener / removeEventListener
- [ ] `<script>` タグのパースと実行順序
- [ ] WPT `dom/events/` テストのパス

---

## Phase 6: CDP WebSocket Server

- [ ] `nhooyr.io/websocket` 依存追加
- [ ] WebSocket サーバー（`uzura serve --port 9222`）
- [ ] CDP Target ディスカバリー（`/json/version`, `/json/list`）
- [ ] Page ドメイン: navigate, enable
- [ ] DOM ドメイン: getDocument, querySelector, getOuterHTML
- [ ] Runtime ドメイン: evaluate
- [ ] Network ドメイン: enable, events
- [ ] Puppeteer接続テスト

---

## Phase 7: Network Interception + Event System

- [ ] Fetch/XHR のインターセプト
- [ ] CDP Fetch ドメイン（requestPaused, fulfillRequest）
- [ ] HTTPヘッダー操作、レスポンスボディ書き換え

---

## Phase 8: Multi-Page + Browser Context

- [ ] 複数タブ（Page）サポート
- [ ] BrowserContext によるセッション分離
- [ ] 並行ページ操作（goroutineベース）

---

## Phase 9: WPT Integration + Benchmark

- [ ] WPTテストランナー組み込み
- [ ] パス率ダッシュボード
- [ ] Lightpanda / Headless Chrome とのベンチマーク比較
- [ ] CI/CD自動テスト