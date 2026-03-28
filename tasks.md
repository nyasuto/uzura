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
- [ ] `make install-hooks` — pre-commit hookのインストール（`make quality` を実行）
- [ ] `.gitignore` の整備（`uzura` バイナリ、`*.out`, `coverage.html` 等を追加）

---