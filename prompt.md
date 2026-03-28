# Uzura 開発プロンプト

あなたはAIエージェント向けに最適化された、Go製ミニマルヘッドレスブラウザ **Uzura** を開発しています。
リポジトリ: `github.com/nyasuto/uzura`

## Uzuraとは

Uzura（うずら/鶉）は Lightpanda に着想を得た軽量ヘッドレスブラウザです。
レンダリング（CSS、画像、レイアウト、GPU）をすべて排除し、
DOM構築とJavaScript実行のみに集中します。
Chrome DevTools Protocol（CDP）を喋るので、既存のPuppeteer/chromedpスクリプトが
そのまま動作します。

## 基本原則

1. **Pure Goのみ** — cgo不使用、C依存なし。`go build` でシングルバイナリを生成する。
2. **テストファースト** — まずテストを書き、次に実装する。公開関数には必ずテストを用意する。
3. **WPT準拠** — Web Platform Tests を正しさの基準とする。
4. **1ファイル1責務** — 1つの型または1つの関心事をファイル単位で管理し、300行以内に収める。
5. **CDP互換** — Puppeteer および chromedp で動作すること。
6. **段階的に進める** — 1つのタスクを完全に終わらせてから次に進む。

## ディレクトリ構成

```
github.com/nyasuto/uzura/
├── cmd/uzura/main.go          # CLIエントリーポイント
├── internal/
│   ├── dom/                   # DOMツリーとノード型
│   │   ├── node.go            # Nodeインターフェースと基本実装
│   │   ├── document.go        # Document型
│   │   ├── element.go         # Element型
│   │   ├── text.go            # TextノードとCommentノード
│   │   ├── nodelist.go        # NodeListとHTMLCollection
│   │   └── serialize.go       # DOM → HTMLシリアライザ
│   ├── html/                  # HTMLパーサー（golang.org/x/net/html のアダプター）
│   │   └── parser.go          # HTML → DOMツリー変換
│   ├── css/                   # CSSセレクターエンジン（cascadiaのアダプター）
│   │   └── selector.go        # querySelector/querySelectorAll
│   ├── js/                    # JavaScriptエンジン（gojaの統合）
│   │   ├── runtime.go         # VMのライフサイクルとサンドボックス
│   │   ├── binding_document.go # documentオブジェクトのバインディング
│   │   └── binding_window.go  # window/console/timerのバインディング
│   ├── cdp/                   # Chrome DevTools Protocol サーバー
│   │   ├── server.go          # WebSocketサーバーとセッション管理
│   │   ├── domain_page.go     # Pageドメインハンドラ
│   │   ├── domain_dom.go      # DOMドメインハンドラ
│   │   └── domain_runtime.go  # Runtimeドメインハンドラ
│   ├── network/               # HTTPクライアントとリクエスト処理
│   │   ├── fetcher.go         # リダイレクト・エンコーディング対応のフェッチャー
│   │   └── cookie.go          # Cookieジャー
│   └── browser/               # Browser/Page/Context のオーケストレーション
│       ├── browser.go         # トップレベルのブラウザインスタンス
│       ├── page.go            # 単一ページ（タブ）
│       └── context.go         # ブラウザコンテキスト（セッション分離）
├── PROMPT.md                  # このファイル
├── tasks.md                   # 現在のフェーズのタスクリスト
├── backlog.md                 # Phase 2以降の全タスク（待機中）
└── PRD.md                     # プロジェクト全体の仕様書
```

## 現在のタスク

`tasks.md` を読み、現在のフェーズと未完了タスクを確認すること。
最初の未チェック項目（`- [ ]`）を選び、テスト付きで完全に実装してから `- [x]` に更新する。

## タスクごとのワークフロー

1. tasks.md のタスク説明を読む
2. まず失敗するテストを書く（`*_test.go`）
3. テストを通す最小限のコードを実装する
4. `go test ./...` を実行 — 全テストがパスすること
5. `go vet ./...` を実行 — 警告がないこと
6. tasks.md のタスクを `- [x]` に更新する
7. gitコミット。メッセージ形式: `phase{N}.{T}: {説明}`
   - 例: `phase1.3: Elementの属性操作メソッドを実装`
8. 現在のフェーズの全タスクが完了したらループが停止する。
   開発者が backlog.md から次フェーズを tasks.md にコピーして再開する。

## コードスタイル

- Go命名規則に従う: 公開名はPascalCase、非公開名はcamelCase
- 公開される関数・型にはすべてgodocコメントをつける（名前で始まること）
- エラーは戻り値で返す。panicは使わない（テスト内は除く）
- `internal/` パッケージを使い、外部からインポートできないようにする
- テストはテーブル駆動を推奨
- パフォーマンスに影響する箇所にはベンチマークテスト: `func BenchmarkX(b *testing.B)`

## 主要な依存ライブラリ

```
golang.org/x/net/html          # HTML5パーサー（WHATWG準拠）
github.com/andybalholm/cascadia # CSSセレクターエンジン（Phase 3以降）
github.com/dop251/goja          # JavaScriptエンジン、pure Go（Phase 5以降）
nhooyr.io/websocket             # CDPサーバー用WebSocket（Phase 6以降）
```

依存はそのフェーズに入るまで追加しないこと。Phase 1で必要なのは `golang.org/x/net/html` のみ。

## DOM設計メモ

DOM は WHATWG仕様に準拠しつつ、Goのイディオムで実装する:
- `Node` はインターフェース（structではない）。ポリモーフィズムのため。
- 各ノード型（Element, Text, Comment, Document）は `baseNode` 構造体を埋め込む
- 親子・兄弟参照は `Node` インターフェースを使う
- `NodeList` は `[]Node` をラップし、ライブクエリ機能を持つ
- 属性は標準ライブラリの `html.Attribute` を使う

```go
// Node はすべてのDOMノードの基本インターフェース。
type Node interface {
    NodeType() NodeType
    NodeName() string
    ParentNode() Node
    ChildNodes() NodeList
    FirstChild() Node
    LastChild() Node
    AppendChild(child Node) Node
    RemoveChild(child Node) Node
    TextContent() string
    SetTextContent(text string)
    OwnerDocument() *Document
}
```

## 重要な制約

- CSSの解析やレンダリングは実装しないこと
- 画像の読み込みやデコードは実装しないこと
- レイアウトやペイントは実装しないこと
- goja依存はPhase 5まで追加しないこと
- websocket依存はPhase 6まで追加しないこと
- 各ファイルは300行以内。超える場合は分割する
- 各タスク完了後に `go build ./cmd/uzura` でコンパイルが通ることを確認する