# Uzura — Go製ミニマルCDP互換ヘッドレスブラウザ

## プロジェクト名

**Uzura** (うずら / 鶉)

小さくて素早い鳥。ミニマルで高速なヘッドレスブラウザにふさわしい名前。

- リポジトリ: `github.com/nyasuto/uzura`
- バイナリ名: `uzura`

## コンセプト

AIエージェント（Claude Code、Puppeteer、Playwright）が使うことに最適化された、
Go製のミニマルヘッドレスブラウザ。人間が見るための描画を一切排除し、
DOM構築とJavaScript実行のみに集中する。

Lightpanda（Zig製）の思想をGoで再解釈し、µシリーズ設計哲学で構築する:
- **シングルバイナリ、ゼロ外部依存**（pure Go、cgo不要）
- **WPTテスト駆動**（Web Platform Testsを段階的にパスしていく）
- **CDP互換**（既存のPuppeteer/chromedpスクリプトがそのまま動く）
- **教育的段階構造**（ブラウザの内部構造を学びながら作る）

## なぜ作るのか

1. **学び**: ブラウザの内部構造（HTML解析→DOM構築→JS実行→CDP通信）を根本から理解する
2. **実用**: Claude Code等のAIエージェントが使う軽量ブラウザとして
3. **Go力の強化**: ネットワーク、並行処理、WebSocket、プロトコル実装の総合演習
4. **Ralph Loop適性**: WPTテストという明確なパス基準があり、tasks.mdに落とし込みやすい

## 技術スタック

| 要素 | 選定 | 理由 |
|------|------|------|
| 言語 | Go | シングルバイナリ、クロスプラットフォーム |
| HTMLパーサー | `golang.org/x/net/html` | WHATWG準拠、Go公式、HTML5エラー訂正対応 |
| CSS Selector | `github.com/andybalholm/cascadia` | `golang.org/x/net/html`と組み合わせ実績 |
| JSエンジン | `github.com/dop251/goja` | pure Go、ES6 80%対応、cgo不要、k6で実績 |
| WebSocket | `nhooyr.io/websocket` | CDP通信用、モダンなAPI |
| HTTP | `net/http` (標準ライブラリ) | ページフェッチ用 |
| テスト | WPT (Web Platform Tests) サブセット | 段階的に対応テスト数を増やす |

### 意図的に含めないもの（Lightpandaと同じ哲学）
- CSS解析・レンダリング
- 画像デコード・表示
- レイアウト計算
- GPU合成
- フォントレンダリング
- スクリーンショット機能（Phase 1では）

## フェーズ設計（9フェーズ）

### Phase 1: HTML Parser + DOM Tree（基盤）
**目標**: HTML文字列を受け取り、DOM Treeを構築して返す

- `golang.org/x/net/html` を使ったHTML5パース
- 独自DOM Tree構造体の設計（Node, Element, Text, Document, Comment）
- `getElementById`, `getElementsByTagName` の実装
- DOM Treeのシリアライズ（HTML出力）

**成功基準**: `echo '<html>...' | uzura parse` でDOM Tree出力

### Phase 2: HTTP Fetcher + Document Loading
**目標**: URLを指定してHTMLを取得し、DOMに変換する

- `net/http`クライアントでのページフェッチ
- リダイレクト処理、Content-Type判定
- 文字エンコーディング検出と変換（UTF-8正規化）
- robots.txt 遵守フラグ、Cookie Jar基本実装

**成功基準**: `uzura fetch https://example.com`

### Phase 3: CSS Selector Engine
**目標**: querySelector / querySelectorAll の実装

- `cascadia` ライブラリの統合
- querySelector, querySelectorAll, matches, closest

### Phase 4: DOM API（Web標準準拠）
**目標**: 主要なDOM操作APIを実装

- Node: appendChild, removeChild, insertBefore, cloneNode
- Element: getAttribute, setAttribute, classList, innerHTML
- Document: createElement, createTextNode
- NodeList / HTMLCollection
- WPT `dom/nodes/` テストのパス

### Phase 5: JavaScript Execution（Goja統合）
**目標**: DOM操作をJavaScriptから実行可能にする

- Goja VMの初期化とサンドボックス設定
- document/window オブジェクトのJSバインディング
- console, setTimeout/setInterval, イベントリスナー
- `<script>` タグの実行

### Phase 6: CDP WebSocket Server（最小実装）
**目標**: Chrome DevTools Protocolの基本ドメインを実装

- WebSocket サーバー起動（`--port 9222`）
- Page, DOM, Runtime, Network ドメイン
- Puppeteerからの接続テスト

### Phase 7: Network Interception + Event System
**目標**: Fetch/XHR のインターセプト・改変機能

### Phase 8: Multi-Page + Browser Context
**目標**: 複数タブとコンテキスト分離

### Phase 9: WPT Integration + Benchmark
**目標**: WPTテストスイート統合、ベンチマーク公開

## CLI設計

```
uzura - AI-optimized headless browser

USAGE:
  uzura <command> [options]

COMMANDS:
  fetch <url>          URLからDOMを取得して出力
  parse                stdinからHTMLをパースしてDOM Treeを出力
  serve [--port 9222]  CDPサーバーを起動
  eval <url> <script>  URLを開いてJavaScriptを実行
  wpt <test-path>      WPTテストを実行
  version              バージョン情報

OPTIONS:
  --format json|text|html    出力フォーマット（デフォルト: text）
  --timeout <seconds>        ページロードタイムアウト（デフォルト: 30）
  --user-agent <string>      User-Agent文字列
  --obey-robots              robots.txtを遵守
  --log-level <level>        ログレベル (debug|info|warn|error)
```

## 先行事例・参考資料

| プロジェクト | 言語 | 特徴 |
|-------------|------|------|
| Lightpanda | Zig | AI特化ヘッドレス、V8統合、CDP互換 |
| mizchi/tui-poc | Rust | ターミナルブラウザ、CDP対応 |
| goja | Go (pure) | ES6 80%対応JSエンジン、k6で実績 |
| chromedp | Go | Go向けCDPクライアント（CDP仕様理解に有用） |
| golang.org/x/net/html | Go | WHATWG準拠HTMLパーサー |
| stroiman headless-browser | Go | Go製ヘッドレスブラウザの先行実装 |