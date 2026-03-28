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

---

## GitHub Actions エコシステム整備

### Task CI.1: 基本CIワークフロー
- [ ] `.github/workflows/ci.yml` を作成
- [ ] Push / PR トリガーで `make quality` を実行
- [ ] Go バージョンマトリクス（1.22, 1.23 等）
- [ ] キャッシュ設定（`actions/setup-go` のモジュールキャッシュ）
- [ ] ステータスバッジを README に追加

### Task CI.2: カバレッジレポート
- [ ] `make cover` でカバレッジ生成
- [ ] Codecov または Coveralls に連携
- [ ] PR にカバレッジ差分コメントを自動投稿
- [ ] カバレッジバッジを README に追加

### Task CI.3: リリース自動化
- [ ] `.github/workflows/release.yml` を作成
- [ ] タグプッシュ（`v*`）トリガーで GoReleaser 実行
- [ ] マルチプラットフォームビルド（linux/amd64, darwin/arm64, darwin/amd64）
- [ ] GitHub Releases にバイナリ・チェックサムを自動アップロード
- [ ] `.goreleaser.yml` の作成

### Task CI.4: Dependabot / セキュリティ
- [ ] `.github/dependabot.yml` を作成（Go モジュール + GitHub Actions の自動更新）
- [ ] `govulncheck` をCIに追加（既知の脆弱性チェック）

### Task CI.5: PR自動化
- [ ] golangci-lint の `reviewdog` 連携（PR にインラインコメント）
- [ ] ベンチマーク結果の PR コメント自動投稿（`benchstat` 差分）