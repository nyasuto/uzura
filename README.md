# Uzura

[![CI](https://github.com/nyasuto/uzura/actions/workflows/ci.yml/badge.svg)](https://github.com/nyasuto/uzura/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/nyasuto/uzura/branch/main/graph/badge.svg)](https://codecov.io/gh/nyasuto/uzura)

AI エージェント向けに最適化された、Go 製ミニマルヘッドレスブラウザ。

レンダリング（CSS、画像、レイアウト、GPU）をすべて排除し、DOM 構築と JavaScript 実行のみに集中します。Chrome DevTools Protocol（CDP）互換のため、既存の Puppeteer / chromedp スクリプトがそのまま動作します。

## 特徴

- **Pure Go** — cgo 不使用、`go build` でシングルバイナリを生成
- **CDP 互換** — Puppeteer、Playwright、chromedp から接続可能
- **高速** — 100KB HTML を ~0.8ms でパース
- **軽量** — レンダリングエンジンを持たないため、メモリ使用量が極めて少ない
- **JavaScript 実行** — goja による ES6 対応の JS エンジン内蔵
- **ネットワークインターセプト** — リクエスト/レスポンスの監視・書き換え
- **マルチタブ** — 複数ページの並行操作、BrowserContext によるセッション分離

## インストール

### バイナリ（GitHub Releases）

[Releases](https://github.com/nyasuto/uzura/releases) からプラットフォームに合ったバイナリをダウンロード。

対応プラットフォーム: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64

### ソースから

```bash
go install github.com/nyasuto/uzura/cmd/uzura@latest
```

## 使い方

### HTML パース

```bash
# stdin から
echo '<div id="main"><p>Hello</p></div>' | uzura parse

# ファイルから
uzura parse index.html

# 出力フォーマット指定
uzura parse --format json index.html
uzura parse --format html index.html
```

### URL フェッチ

```bash
uzura fetch https://example.com
uzura fetch --format json --timeout 10 https://example.com
```

### CDP サーバー

```bash
# CDP サーバーを起動（デフォルト: ポート 9222）
uzura serve --port 9222
```

Puppeteer から接続:

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

### JavaScript 実行

```bash
uzura eval https://example.com 'document.title'
```

### WPT テスト

```bash
uzura wpt path/to/test.html
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
├── dom/        DOM ツリーとノード型
├── html/       HTML パーサー（golang.org/x/net/html アダプター）
├── css/        CSS セレクターエンジン（cascadia アダプター）
├── js/         JavaScript エンジン（goja 統合）
├── cdp/        CDP WebSocket サーバー
├── network/    HTTP フェッチャー
├── browser/    Browser / Page / Context
├── page/       ページライフサイクル管理
└── wpt/        WPT テストランナー
```

## 依存ライブラリ

| ライブラリ | 用途 |
|-----------|------|
| `golang.org/x/net/html` | HTML5 パーサー（WHATWG 準拠） |
| `github.com/andybalholm/cascadia` | CSS セレクターエンジン |
| `github.com/dop251/goja` | JavaScript エンジン（pure Go） |
| `github.com/coder/websocket` | CDP サーバー用 WebSocket |
| `golang.org/x/text` | 文字エンコーディング変換 |

## 着想

[Lightpanda](https://github.com/nicholasgasior/lightpanda)（Zig 製 AI 特化ヘッドレスブラウザ）の思想を Go で再解釈したプロジェクトです。

## ライセンス

MIT
