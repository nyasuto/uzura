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
