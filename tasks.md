# Uzura — Phase 1: HTML Parser + DOM Tree

### Task 1.1: Project Scaffolding

- [x] `go mod init github.com/nyasuto/uzura`
- [x] ディレクトリ作成: `cmd/uzura/main.go`, `internal/dom/`, `internal/html/`
- [x] `golang.org/x/net/html` を依存に追加
- [x] `parse` サブコマンド付きの基本CLI（stdinからHTMLを読みDOM出力）
- [x] `go build ./cmd/uzura` でバイナリ生成確認
- [x] `uzura version` でバージョン出力

### Task 1.2: Node Interface and Base Types

- [x] `NodeType` 定数を定義（ElementNode, TextNode, CommentNode, DocumentNode）
- [x] `Node` インターフェースをWHATWG準拠で定義
- [x] `baseNode` 構造体（parent/child/sibling管理）を実装
- [x] `AppendChild` — 旧親からの切り離しを含む
- [x] `RemoveChild` — sibling ポインタ更新を含む
- [x] `InsertBefore` — nil ref = appendのエッジケース
- [x] テーブル駆動テストで全ツリー操作を検証

### Task 1.3: Text and Comment Nodes

- [x] `Text` 構造体（`baseNode` 埋め込み）
- [x] `Comment` 構造体（`baseNode` 埋め込み）
- [x] `NodeName()` → `#text` / `#comment`
- [x] `TextContent()` の取得・設定
- [x] `CloneNode()` の実装
- [x] テスト: 生成、内容get/set、クローン

### Task 1.4: Element Implementation

- [x] `Element` 構造体（`baseNode` 埋め込み）
- [x] `TagName`（大文字）と `LocalName`（小文字）
- [x] 属性: `GetAttribute`, `SetAttribute`, `HasAttribute`, `RemoveAttribute`
- [x] `Id()`, `ClassName()` ショートハンド
- [x] テスト: 属性CRUD、大文字小文字

### Task 1.5: Document Implementation

- [x] `Document` 構造体（`baseNode` 埋め込み）
- [x] `DocumentElement()`, `Head()`, `Body()`, `Title()`
- [x] `CreateElement`, `CreateTextNode`, `CreateComment`
- [x] `GetElementById`, `GetElementsByTagName`, `GetElementsByClassName`
- [x] テスト: 要素生成、id/tag/classでの検索

### Task 1.6: HTML Parser → DOM Tree Conversion

- [x] `internal/html/parser.go` — `golang.org/x/net/html` のアダプター
- [x] `Parse(r io.Reader) (*dom.Document, error)`
- [x] `html.Node` → `dom.Node` の再帰変換
- [x] 不正HTML（暗黙タグ挿入）の処理確認
- [x] テスト: 基本HTML、ネストテーブル、閉じタグ欠落、空ドキュメント

### Task 1.7: DOM Serializer

- [x] `Serialize(node Node) string` — DOM → HTML文字列
- [x] void要素（`<br>`, `<img>`等）、属性エスケープ、テキストエスケープ
- [x] raw text要素（`<script>`, `<style>`）
- [x] `InnerHTML()`, `OuterHTML()`
- [x] ラウンドトリップテスト: parse → serialize → parse → deep-equal

### Task 1.8: CLI `parse` コマンド仕上げ

- [x] stdin / ファイル入力対応
- [x] `--format text` インデント付きツリー表示（デフォルト）
- [x] `--format json` JSON構造
- [x] `--format html` 再シリアライズHTML
- [x] stderr/stdoutの分離、終了コード

### Task 1.9: Phase 1 検証

- [x] `go test ./... -race` 全パス
- [x] `go vet ./...` クリーン
- [x] ベンチマーク: 100KB HTML を 50ms以内でパース（実測: ~0.8ms）
- [x] `curl -s https://example.com | ./uzura parse` 動作確認

### Phase 1 Status: COMPLETE ✅

---

## 開発エコシステム整備

### Task 0.1: Makefile

- [x] `Makefile` を作成
- [x] `make build` — `go build -o uzura ./cmd/uzura`
- [x] `make test` — `go test ./... -race`
- [x] `make bench` — `go test ./... -bench=. -benchmem`
- [x] `make vet` — `go vet ./...`
- [x] `make lint` — `golangci-lint run`（未インストールならスキップ）
- [x] `make fmt` — `gofmt -w .` + `goimports -w .`
- [x] `make clean` — バイナリ・キャッシュ削除
- [x] `make quality` — `fmt` + `vet` + `lint` + `test` を一括実行
- [x] `make all` — `quality` + `build`（デフォルトターゲット）
- [x] `make cover` — カバレッジレポート生成（`go test -coverprofile` + `go tool cover -html`）

### Task 0.2: CI用ヘルパー

- [x] `.golangci.yml` — lintルール設定（unused, errcheck, govet, staticcheck等）
- [x] `.editorconfig` — インデント・改行コード統一

### Task 0.3: Git hooks

- [x] `make install-hooks` — pre-commit hookのインストール（`make quality` を実行）
- [x] `.gitignore` の整備（`uzura` バイナリ、`*.out`, `coverage.html` 等を追加）

---

## Phase 2: HTTP Fetcher + Document Loading

### Task 2.1: Basic HTTP Fetcher

- [x] `internal/network/fetcher.go`
- [x] `Fetch(url string) (*Response, error)` タイムアウト付き
- [x] User-Agent設定、リダイレクト追跡（最大10回）
- [x] `net/http/httptest` でテスト

### Task 2.2: Content-Type and Encoding

- [x] Content-Typeヘッダーからcharset検出
- [x] `<meta charset>` タグからのcharset検出
- [x] `golang.org/x/text/encoding` でUTF-8変換
- [x] テスト: Shift-JIS, EUC-JP, ISO-8859-1

### Task 2.3: Cookie Jar

- [x] `net/http/cookiejar` 統合
- [x] セッション内のCookie永続化
- [x] テスト: set/get, リダイレクト跨ぎ

### Task 2.4: Document Loading Pipeline

- [x] Fetch → Decode → Parse → Document パイプライン
- [x] エラーハンドリング（ネットワーク、タイムアウト、不正HTML）
- [x] httptest での結合テスト

### Task 2.5: CLI `fetch` コマンド

- [x] `uzura fetch <url>`
- [x] `--format`, `--timeout`, `--user-agent` フラグ
- [x] 動作確認: `uzura fetch https://example.com`

### Task 2.6: robots.txt

- [x] `--obey-robots` フラグ
- [x] robots.txtパースとallow/disallowチェック

### Task 2.7: Phase 2 検証

- [x] テスト全パス、日本語サイトのエンコーディング確認

### Phase 2 Status: COMPLETE ✅

---

## Phase 3: CSS Selector Engine

### Task 3.1: cascadia統合

- [x] `github.com/andybalholm/cascadia` 依存追加
- [x] `internal/css/selector.go` アダプター

### Task 3.2: querySelector / querySelectorAll

- [x] Element/DocumentにquerySelector/querySelectorAllを追加
- [x] テスト: tag, class, id, 属性, 結合子, 擬似クラス

### Task 3.3: matches / closest

- [x] `Element.Matches()`, `Element.Closest()`

### Task 3.4: Phase 3 検証

- [x] 複雑セレクターのテスト、大規模DOMでのベンチマーク

### Phase 3 Status: COMPLETE ✅

---

## Phase 4: DOM API（Web標準準拠）

- [x] Node: appendChild, removeChild, insertBefore, cloneNode（Phase 1で基本実装済み、拡張）
- [x] Element: classList, dataset, innerHTML setter（パーサー連携）
- [x] Document: createDocumentFragment, importNode
- [x] MutationObserver 基本実装
- [x] WPT `dom/nodes/` テストのパス

### Phase 4 Status: COMPLETE ✅

---


## Phase 4.5: アーキテクチャ改善（JS/CDP準備）

Gemini / o3 レビューで指摘された構造的改善。Phase 5（JS）以降で手戻りしないよう先に対処する。

### Task 4.5.1: QueryEngine インターフェース化
- [x] `dom.QueryEngine` インターフェースを定義（QuerySelector, QuerySelectorAll, Matches, Closest）
- [x] `Document` / `Element` 生成時にインターフェースを注入する形に変更
- [x] `query.go` の関数変数パターンを廃止
- [x] `internal/css` が `QueryEngine` を実装
- [x] 既存テストが全パス

### Task 4.5.2: Observer パターン（DOMミューテーション検知）
- [x] `dom.MutationRecord` 型を定義（childList, attributes, characterData）
- [x] `dom.MutationObserver` + `MutationCallback` 型を定義
- [x] `baseNode` の変更メソッド（AppendChild, RemoveChild, SetAttribute等）からイベント発火
- [x] テスト: ノード追加/削除/属性変更でコールバックが呼ばれることを検証（18テスト）
- [x] 将来のCDP（DOM.documentUpdated）やJS EventListener の基盤となる

### Task 4.5.3: オーケストレーション層（internal/page）
- [x] `internal/page` パッケージを新設
- [x] `Page` 構造体: DOM, Network, （将来の）JSContext を統括
- [x] `Page.Navigate(ctx, url)` — Fetch→Decode→Parse→DOM のライフサイクル管理
- [x] `context.Context` を全ブロッキング操作に導入（Fetcher.FetchContext, LoadDocumentContext）
- [x] `network/loader.go` の責務を `page` に移行、CLI を Page 経由に更新
- [x] テスト: httptest.Server を使ったページロードの E2E テスト（8テスト）

### Task 4.5.4: エラー型の整備
- [x] sentinel errors を定義（`ErrRobotsDisallowed`, `ErrTooManyRedirects`, `ErrInvalidSelector`）
- [x] `FetchError` 構造体（StatusCode, URL を保持、Unwrap対応）
- [x] `errors.Is()` / `errors.As()` で分岐可能にする
- [x] 既存のエラーハンドリングをリファクタリング（robots.go, fetcher.go, css/selector.go）

---


## GitHub Actions エコシステム整備

### Task CI.0: Makefile の `vet` 削除
- [x] `make quality` から `vet` ターゲットを除外（`golangci-lint` に一本化）
- [x] Makefile から `vet` ターゲット自体を削除
- [x] `golangci-lint` を必須化（未インストール時はスキップではなくエラー）
- [x] CLAUDE.md の `go vet ./...` 記載を更新

### Task CI.1: 基本CIワークフロー
- [x] `.github/workflows/ci.yml` を作成
- [x] Push / PR トリガーで lint + test を実行
- [x] Go バージョンマトリクス（1.26、go.mod要件に合わせ）
- [x] キャッシュ設定（`actions/setup-go` + `golangci-lint-action` の自動キャッシュ）
- [ ] ステータスバッジを README に追加（README作成時に対応）

### Task CI.2: カバレッジレポート
- [x] `make cover` でカバレッジ生成（既存）
- [x] Codecov に連携（codecov-action@v5）
- [x] PR にカバレッジ差分コメントを自動投稿（Codecov側のPR統合で対応）
- [ ] カバレッジバッジを README に追加（README作成時に対応）

### Task CI.3: リリース自動化
- [x] `.github/workflows/release.yml` を作成
- [x] タグプッシュ（`v*`）トリガーで GoReleaser 実行
- [x] マルチプラットフォームビルド（linux/amd64, linux/arm64, darwin/arm64, darwin/amd64）
- [x] GitHub Releases にバイナリ・チェックサムを自動アップロード
- [x] `.goreleaser.yml` の作成

### Task CI.4: Dependabot / セキュリティ
- [x] `.github/dependabot.yml` を作成（Go モジュール + GitHub Actions の自動更新）
- [x] `govulncheck` をCIに追加（既知の脆弱性チェック）

### Task CI.5: PR自動化
- [ ] golangci-lint の `reviewdog` 連携（PR にインラインコメント）
- [ ] ベンチマーク結果の PR コメント自動投稿（`benchstat` 差分）

## Phase 5: JavaScript Execution（Goja統合）

### Phase 5.1: VM基盤 + グローバルオブジェクト
- [ ] `github.com/dop251/goja` 依存追加
- [ ] `internal/js/vm.go` — Goja Runtime のラッパー（生成、実行、リセット）
- [ ] サンドボックス: ファイルI/O・ネットワーク等のネイティブアクセスを遮断
- [ ] `window` / `globalThis` オブジェクトの基本構造
- [ ] `console.log` / `console.warn` / `console.error`（Go の io.Writer に出力）
- [ ] テスト: JS式の評価、console出力のキャプチャ、禁止操作のエラー

### Phase 5.2: Document バインディング（読み取り系）
- [ ] `document` オブジェクトを goja に登録
- [ ] `document.getElementById`, `document.querySelector`, `document.querySelectorAll`
- [ ] `document.getElementsByTagName`, `document.getElementsByClassName`
- [ ] `document.title`, `document.documentElement`, `document.head`, `document.body`
- [ ] Element プロキシ: `tagName`, `id`, `className`, `textContent`, `innerHTML`（getter）
- [ ] `element.getAttribute`, `element.hasAttribute`
- [ ] `element.querySelector`, `element.querySelectorAll`, `element.matches`, `element.closest`
- [ ] NodeList / HTMLCollection の JS 表現（length, index アクセス, forEach）
- [ ] テスト: JS からの DOM クエリが Go 側の DOM と一致することを検証

### Phase 5.3: Document バインディング（書き込み系）
- [ ] `document.createElement`, `document.createTextNode`, `document.createDocumentFragment`
- [ ] `element.setAttribute`, `element.removeAttribute`
- [ ] `element.textContent` setter, `element.innerHTML` setter
- [ ] `node.appendChild`, `node.removeChild`, `node.insertBefore`, `node.replaceChild`
- [ ] `element.classList`（add, remove, toggle, contains）の JS バインディング
- [ ] `element.dataset` の JS Proxy バインディング
- [ ] テスト: JS で DOM を変更 → Go 側の DOM ツリーに反映されることを検証

### Phase 5.4: イベントシステム
- [ ] `EventTarget` インターフェース（addEventListener, removeEventListener, dispatchEvent）
- [ ] `Event` オブジェクト（type, target, currentTarget, bubbles, cancelable, preventDefault, stopPropagation）
- [ ] イベントバブリング・キャプチャリングの実装
- [ ] `document`, `window`, 各 `Element` を EventTarget として登録
- [ ] テスト: イベント発火、バブリング、preventDefault の動作検証

### Phase 5.5: タイマーとイベントループ
- [ ] イベントループの基本構造（タスクキュー + 実行サイクル）
- [ ] `setTimeout` / `clearTimeout`
- [ ] `setInterval` / `clearInterval`
- [ ] Go goroutine との同期（goja は単一スレッド、タイマーコールバックはループ内で実行）
- [ ] テスト: タイマーの順序保証、クリア動作、ネストしたタイマー

### Phase 5.6: `<script>` タグ実行
- [ ] HTML パーサーからの `<script>` タグ検出
- [ ] インラインスクリプトの実行（document 解析順）
- [ ] `defer` / `async` 属性のセマンティクス
- [ ] スクリプトエラー時のハンドリング（他のスクリプトは続行）
- [ ] テスト: 複数スクリプトの実行順序、defer/async の動作

### Phase 5.7: WPT テスト + 検証
- [ ] WPT `dom/events/` テストのパス
- [ ] `go test ./... -race` 全パス
- [ ] JS 実行のベンチマーク（単純スクリプト、DOM操作スクリプト）

---
