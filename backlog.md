# Uzura — Backlog

フェーズ完了時、次フェーズの内容を tasks.md にコピーしてループを再開する。

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