# Uzura — Backlog

フェーズ完了時、次フェーズの内容を tasks.md にコピーしてループを再開する。

---

## Phase 4.5: アーキテクチャ改善（JS/CDP準備）

Gemini / o3 レビューで指摘された構造的改善。Phase 5（JS）以降で手戻りしないよう先に対処する。

### Task 4.5.1: QueryEngine インターフェース化
- [ ] `dom.QueryEngine` インターフェースを定義（QuerySelector, QuerySelectorAll, Matches, Closest）
- [ ] `Document` / `Element` 生成時にインターフェースを注入する形に変更
- [ ] `query.go` の関数変数パターンを廃止
- [ ] `internal/css` が `QueryEngine` を実装
- [ ] 既存テストが全パス

### Task 4.5.2: Observer パターン（DOMミューテーション検知）
- [ ] `dom.MutationEvent` 型を定義（ChildAdded, ChildRemoved, AttributeChanged 等）
- [ ] `dom.MutationObserver` インターフェース（またはコールバック型）を定義
- [ ] `baseNode` の変更メソッド（AppendChild, RemoveChild, SetAttribute等）からイベント発火
- [ ] テスト: ノード追加/削除/属性変更でコールバックが呼ばれることを検証
- [ ] 将来のCDP（DOM.documentUpdated）やJS EventListener の基盤となる

### Task 4.5.3: オーケストレーション層（internal/page）
- [ ] `internal/page` パッケージを新設
- [ ] `Page` 構造体: DOM, Network, （将来の）JSContext を統括
- [ ] `Page.Navigate(ctx, url)` — Fetch→Decode→Parse→DOM のライフサイクル管理
- [ ] `context.Context` を全ブロッキング操作に導入
- [ ] `network/loader.go` の責務を `page` に移行
- [ ] テスト: httptest.Server を使ったページロードの E2E テスト

### Task 4.5.4: エラー型の整備
- [ ] sentinel errors を定義（`ErrRobotsDisallowed`, `ErrTimeout`, `ErrInvalidSelector` 等）
- [ ] `FetchError` 構造体（StatusCode, URL を保持）
- [ ] `errors.Is()` / `errors.As()` で分岐可能にする
- [ ] 既存のエラーハンドリングをリファクタリング

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