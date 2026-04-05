# Uzura

[![CI](https://github.com/nyasuto/uzura/actions/workflows/ci.yml/badge.svg)](https://github.com/nyasuto/uzura/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/nyasuto/uzura/branch/main/graph/badge.svg)](https://codecov.io/gh/nyasuto/uzura)

AI エージェント向けに最適化された、Go 製ミニマルヘッドレスブラウザ。

レンダリング（CSS、画像、レイアウト、GPU）をすべて排除し、DOM 構築と JavaScript 実行のみに集中します。MCP（Model Context Protocol）サーバー内蔵で Claude Code / Claude Desktop からそのまま使えます。CDP 互換のため、Puppeteer / chromedp からの接続にも対応。

## 特徴

- **Pure Go** — cgo 不使用、`go build` でシングルバイナリを生成
- **MCP サーバー内蔵** — Claude Code / Claude Desktop からブラウズ・クエリ・操作が可能
- **CDP 互換** — Puppeteer、Playwright、chromedp から接続可能
- **高速** — 100KB HTML を ~0.8ms でパース
- **軽量** — レンダリングエンジンを持たないため、メモリ使用量が極めて少ない
- **AI 最適化出力** — Markdown（LLM フィード用）とセマンティックツリー（構造化ページ理解）
- **JavaScript 実行** — goja による ES6 対応の JS エンジン内蔵
- **ネットワークインターセプト** — リクエスト/レスポンスの監視・書き換え
- **マルチタブ** — 複数ページの並行操作、BrowserContext によるセッション分離

## 実サイト互換性（30サイトテスト）

| カテゴリ | サイト数 | 成功率 | 備考 |
|----------|----------|--------|------|
| 日本語サイト | 10 | 88% | 食べログ、SUUMO、NHK等 |
| 英語テックサイト | 10 | 82% | GitHub、Go、Rust、HN等 |
| SPA (SSR/SSG) | 5 | 96% | React、Vue、Svelte、Vercel等 |
| 大規模・複雑 | 5 | 32% | Wikipedia は完全対応、ボット検知サイトは未対応 |
| **全体** | **30** | **75%** | |

テスト種別では browse text / semantic_tree が最も安定（各90%）。詳細は [site-compatibility-2026-04-05.md](site-compatibility-2026-04-05.md) を参照。

## インストール

### バイナリ（GitHub Releases）

[Releases](https://github.com/nyasuto/uzura/releases) からプラットフォームに合ったバイナリをダウンロード。

対応プラットフォーム: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64

### ソースから

```bash
go install github.com/nyasuto/uzura/cmd/uzura@latest
```

## 使い方

### MCP サーバー（Claude Code / Claude Desktop 統合）

```bash
# stdio モードで MCP サーバーを起動
uzura mcp
```

Claude Code の `.mcp.json` または Claude Desktop の `claude_desktop_config.json` に追加:

```json
{
  "mcpServers": {
    "uzura": {
      "command": "uzura",
      "args": ["mcp"]
    }
  }
}
```

利用可能なツール:

| ツール | 説明 | 入力例 |
|--------|------|--------|
| `browse` | URL を開いてコンテンツ取得 | `url`, `format` (text/html/json/markdown/semantic) |
| `semantic_tree` | ページの構造化表現を取得 | `url`, `max_depth` |
| `query` | CSS セレクターで要素を検索 | `url`, `selector`, `attribute` |
| `evaluate` | ページ上で JavaScript を実行 | `url`, `script` |
| `interact` | 要素のクリック・フォーム入力 | `url`, `selector`, `action` (click/fill), `value` |

### CLI

```bash
# HTML パース（stdin / ファイル）
echo '<div><p>Hello</p></div>' | uzura parse
uzura parse --format markdown index.html

# URL フェッチ
uzura fetch https://example.com
uzura fetch --format markdown https://example.com
uzura fetch --format semantic --semantic-depth 5 https://example.com

# CDP サーバー起動
uzura serve --port 9222

# WPT テスト
uzura wpt path/to/test.html
```

出力フォーマット: `text`（デフォルト）, `json`, `html`, `markdown`, `semantic`

### CDP サーバー（Puppeteer / chromedp）

```javascript
const puppeteer = require('puppeteer-core');
const browser = await puppeteer.connect({
  browserWSEndpoint: 'ws://localhost:9222/devtools/browser'
});
const page = await browser.newPage();
await page.goto('https://example.com');
const title = await page.title();
console.log(title);
```

## 開発

```bash
# 全品質チェック + ビルド
make all

# テスト
make test

# ベンチマーク
make bench
make bench-report

# カバレッジ
make cover

# WPT テスト
make wpt-fetch   # 初回のみ: WPT スイートをダウンロード
make wpt

# E2E テスト（Node.js 必要）
make e2e
```

## アーキテクチャ

```
internal/
├── dom/        DOM ツリーとノード型（WHATWG 準拠）
├── html/       HTML パーサー（golang.org/x/net/html アダプター）
├── css/        CSS セレクターエンジン（cascadia アダプター）
├── js/         JavaScript エンジン（goja 統合、ES6 対応）
├── cdp/        CDP WebSocket サーバー
├── network/    HTTP フェッチャー（Cookie, リダイレクト, robots.txt）
├── browser/    Browser / BrowserContext（セッション分離）
├── page/       ページライフサイクル管理
├── mcp/        MCP サーバー（5 ツール）
├── markdown/   DOM → Markdown 変換（readability 統合）
├── semantic/   セマンティックツリー構築（AI 向け構造化表現）
├── wpt/        WPT テストランナー
├── bench/      ベンチマークスイート
└── errors/     Sentinel エラー型
```

| 指標 | 数値 |
|------|------|
| 実装コード | ~12,000 行 |
| テストコード | ~15,000 行（69 ファイル） |
| パッケージ数 | 14 |

## 依存ライブラリ

| ライブラリ | 用途 |
|-----------|------|
| `golang.org/x/net/html` | HTML5 パーサー（WHATWG 準拠） |
| `github.com/andybalholm/cascadia` | CSS セレクターエンジン |
| `github.com/dop251/goja` | JavaScript エンジン（pure Go） |
| `github.com/coder/websocket` | CDP サーバー用 WebSocket |
| `golang.org/x/text` | 文字エンコーディング変換 |
| `codeberg.org/readeck/go-readability/v2` | Markdown 本文抽出 |

## 着想

[Lightpanda](https://github.com/nicholasgasior/lightpanda)（Zig 製 AI 特化ヘッドレスブラウザ）の思想を Go で再解釈したプロジェクトです。

## ライセンス

MIT
