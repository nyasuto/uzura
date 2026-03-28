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
- [ ] `Text` 構造体（`baseNode` 埋め込み）
- [ ] `Comment` 構造体（`baseNode` 埋め込み）
- [ ] `NodeName()` → `#text` / `#comment`
- [ ] `TextContent()` の取得・設定
- [ ] `CloneNode()` の実装
- [ ] テスト: 生成、内容get/set、クローン

### Task 1.4: Element Implementation
- [ ] `Element` 構造体（`baseNode` 埋め込み）
- [ ] `TagName`（大文字）と `LocalName`（小文字）
- [ ] 属性: `GetAttribute`, `SetAttribute`, `HasAttribute`, `RemoveAttribute`
- [ ] `Id()`, `ClassName()` ショートハンド
- [ ] テスト: 属性CRUD、大文字小文字

### Task 1.5: Document Implementation
- [ ] `Document` 構造体（`baseNode` 埋め込み）
- [ ] `DocumentElement()`, `Head()`, `Body()`, `Title()`
- [ ] `CreateElement`, `CreateTextNode`, `CreateComment`
- [ ] `GetElementById`, `GetElementsByTagName`, `GetElementsByClassName`
- [ ] テスト: 要素生成、id/tag/classでの検索

### Task 1.6: HTML Parser → DOM Tree Conversion
- [ ] `internal/html/parser.go` — `golang.org/x/net/html` のアダプター
- [ ] `Parse(r io.Reader) (*dom.Document, error)`
- [ ] `html.Node` → `dom.Node` の再帰変換
- [ ] 不正HTML（暗黙タグ挿入）の処理確認
- [ ] テスト: 基本HTML、ネストテーブル、閉じタグ欠落、空ドキュメント

### Task 1.7: DOM Serializer
- [ ] `Serialize(node Node) string` — DOM → HTML文字列
- [ ] void要素（`<br>`, `<img>`等）、属性エスケープ、テキストエスケープ
- [ ] raw text要素（`<script>`, `<style>`）
- [ ] `InnerHTML()`, `OuterHTML()`
- [ ] ラウンドトリップテスト: parse → serialize → parse → deep-equal

### Task 1.8: CLI `parse` コマンド仕上げ
- [ ] stdin / ファイル入力対応
- [ ] `--format text` インデント付きツリー表示（デフォルト）
- [ ] `--format json` JSON構造
- [ ] `--format html` 再シリアライズHTML
- [ ] stderr/stdoutの分離、終了コード

### Task 1.9: Phase 1 検証
- [ ] `go test ./... -race` 全パス
- [ ] `go vet ./...` クリーン
- [ ] ベンチマーク: 100KB HTML を 50ms以内でパース
- [ ] `curl -s https://example.com | ./uzura parse` 動作確認