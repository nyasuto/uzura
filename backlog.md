# Uzura — Backlog

フェーズ完了時、次フェーズの内容を tasks.md にコピーしてループを再開する。

## Phase 10: MCPサーバー内蔵

UzuraをClaude Code / Claude DesktopのMCPツールとして使えるようにする。
`uzura mcp` でstdioモードのMCPサーバーが起動し、シングルバイナリ内で完結する。

### Task 10.1: MCP プロトコル基盤

- [ ] MCP JSON-RPC メッセージ型の定義（Request, Response, Notification）
- [ ] `internal/mcp/protocol.go`: JSON-RPC 2.0 のパース・シリアライズ
- [ ] `initialize` / `initialized` ハンドシェイク実装
- [ ] `ping` / `pong` 実装
- [ ] テスト: メッセージのラウンドトリップ、不正JSONのエラー処理

### Task 10.2: stdio トランスポート

- [ ] `internal/mcp/transport.go`: stdin/stdout での行区切りJSON-RPC読み書き
- [ ] バッファリングと改行処理（MCP仕様: Content-Length or newline-delimited）
- [ ] stderr へのログ出力（stdoutはMCPプロトコル専用）
- [ ] テスト: パイプ経由での双方向通信

### Task 10.3: ツール定義 — `browse`

- [ ] `internal/mcp/tools.go`: MCPツール登録の仕組み（名前、説明、inputSchema）
- [ ] `tools/list` レスポンスの実装
- [ ] `browse` ツール定義:
  ```json
  {
    "name": "browse",
    "description": "URLを開いてページのコンテンツを取得する",
    "inputSchema": {
      "type": "object",
      "properties": {
        "url": { "type": "string", "description": "取得するURL" },
        "format": {
          "type": "string",
          "enum": ["text", "html", "json"],
          "default": "text"
        }
      },
      "required": ["url"]
    }
  }
  ```
- [ ] テスト: ツール一覧の正しいスキーマ出力

### Task 10.4: `browse` ツール実行

- [ ] `tools/call` ハンドラの実装
- [ ] `browse` 呼び出し時の処理: URL取得 → DOM構築 → JS実行 → フォーマット出力
- [ ] Phase 2 の Fetcher + Phase 1 の Parser を結合して呼ぶ
- [ ] エラーハンドリング: ネットワークエラー、タイムアウト、不正URL
- [ ] テスト: httptest.Server を使ったブラウズ→結果返却

### Task 10.5: ツール定義 — `evaluate`

- [ ] `evaluate` ツール定義:
  ```json
  {
    "name": "evaluate",
    "description": "ページ上でJavaScriptを実行して結果を返す",
    "inputSchema": {
      "type": "object",
      "properties": {
        "url": { "type": "string" },
        "script": { "type": "string", "description": "実行するJavaScript式" }
      },
      "required": ["url", "script"]
    }
  }
  ```
- [ ] `tools/call` に `evaluate` ハンドラ追加
- [ ] Goja VM で式を評価し、結果を文字列化して返却
- [ ] テスト: DOM操作スクリプトの実行、エラースクリプトの処理

### Task 10.6: ツール定義 — `query`

- [ ] `query` ツール定義:
  ```json
  {
    "name": "query",
    "description": "CSSセレクターで要素を検索し、テキストや属性を返す",
    "inputSchema": {
      "type": "object",
      "properties": {
        "url": { "type": "string" },
        "selector": { "type": "string", "description": "CSSセレクター" },
        "attribute": {
          "type": "string",
          "description": "取得する属性名（省略時はtextContent）"
        }
      },
      "required": ["url", "selector"]
    }
  }
  ```
- [ ] マッチした要素のリストを返す（テキスト、属性値、outerHTML）
- [ ] テスト: 複数要素のマッチ、属性取得、マッチなしの場合

### Task 10.7: ツール定義 — `interact`

- [ ] `interact` ツール定義:
  ```json
  {
    "name": "interact",
    "description": "ページ上の要素をクリックまたはフォーム入力する",
    "inputSchema": {
      "type": "object",
      "properties": {
        "url": { "type": "string" },
        "selector": { "type": "string" },
        "action": { "type": "string", "enum": ["click", "fill"] },
        "value": { "type": "string", "description": "fill時の入力値" }
      },
      "required": ["url", "selector", "action"]
    }
  }
  ```
- [ ] `click`: 要素のclickイベント発火
- [ ] `fill`: input/textarea/selectの値設定 + input/changeイベント発火
- [ ] テスト: フォーム入力→JS読み取り、クリック→イベントハンドラ実行

### Task 10.8: CLI `mcp` サブコマンド

- [ ] `uzura mcp` でstdioモードMCPサーバーを起動
- [ ] `--log-level` フラグ（stderrへ出力）
- [ ] Ctrl+C / EOF でのグレースフルシャットダウン
- [ ] テスト: プロセス起動→initialize→tools/list→終了の一連の流れ

### Task 10.9: Claude Code 統合テスト

- [ ] `.claude.json` 用の設定例を README に記載
- [ ] 手動テスト: Claude Code から `browse` ツールを呼び出し
- [ ] 手動テスト: Claude Code から `query` + `evaluate` の組み合わせ
- [ ] ページセッション管理: 同一URLへの連続呼び出しでDOM再利用（キャッシュ）

### Task 10.10: Phase 10 Verification

- [ ] `go test ./... -race` 全パス
- [ ] MCPプロトコルの仕様準拠確認（JSON-RPCエラーコード等）
- [ ] `uzura mcp` の起動時間が100ms以内
- [ ] README に MCP セットアップ手順を記載

---

## Phase 11: Markdown出力（LLMフィード最適化）

DOMツリーをLLMが消化しやすいMarkdown形式に変換する。
MCPの `browse` ツールで `format: "markdown"` を指定すると使える。

### Task 11.1: go-readability 統合

- [x] `github.com/go-shiori/go-readability` 依存追加（またはv2: codeberg.org/readeck/go-readability/v2）
- [x] `internal/markdown/readability.go`: DOMから本文抽出のアダプター
- [x] 入力: `*dom.Document` → 出力: タイトル、著者、本文テキスト
- [x] テスト: 記事ページ、非記事ページ（トップページ等）のフォールバック

### Task 11.2: DOM → Markdown 変換器

- [x] `internal/markdown/converter.go`: DOMノードのMarkdown変換
- [x] 見出し: `<h1>`-`<h6>` → `#`-`######`
- [x] 段落: `<p>` → 空行区切り
- [x] リスト: `<ul>/<ol>/<li>` → `- ` / `1. `（ネスト対応）
- [x] リンク: `<a href>` → `[text](url)`
- [x] 強調: `<strong>` → `**text**`, `<em>` → `*text*`
- [x] コード: `<code>` → バッククォート, `<pre>` → コードブロック
- [x] 画像: `<img>` → `![alt](src)`（altテキスト優先）
- [x] テーブル: `<table>` → Markdownテーブル（`| col1 | col2 |`）
- [x] テスト: 各要素の変換、ネスト構造、空要素

### Task 11.3: 不要要素の除去

- [x] `<script>`, `<style>`, `<noscript>` の除去
- [x] `<nav>`, `<header>`, `<footer>`, `<aside>` の除去（本文抽出モード時）
- [x] hidden属性、`display:none` インラインスタイルの要素除去
- [x] 広告系クラス名のヒューリスティック除去（`ad-`, `sidebar`, `promo`等）
- [x] テスト: クリーン前後の比較

### Task 11.4: メタデータ抽出

- [x] `<title>` タグからタイトル
- [x] `<meta name="description">` から説明文
- [x] `<meta name="author">` から著者
- [x] Open Graph タグ（`og:title`, `og:description`, `og:image`）
- [x] JSON-LD (`<script type="application/ld+json">`) のパース
- [x] 出力形式: Markdownの先頭にYAML frontmatter風のメタデータブロック

  ```markdown
  ---
  title: 記事タイトル
  author: 著者名
  url: https://example.com/article
  ---

  # 記事タイトル

  本文がここに...
  ```

- [x] テスト: メタデータあり/なし/部分的の各パターン

### Task 11.5: `browse` ツールへの `markdown` フォーマット追加

- [x] `browse` ツールの `format` に `"markdown"` を追加
- [x] readability で本文抽出 → Markdown変換 → メタデータ付与のパイプライン
- [x] readability失敗時のフォールバック: ページ全体をMarkdown変換
- [x] テスト: MCP経由でのmarkdown出力

### Task 11.6: CLIへの `--format markdown` 追加

- [x] `uzura fetch <url> --format markdown` で動作
- [x] `uzura parse --format markdown` で動作（stdin入力）
- [x] 出力のトークン数概算をstderrに表示（`--verbose`時）

### Task 11.7: Phase 11 Verification

- [x] 日本語記事サイトでのMarkdown出力テスト
- [x] 英語ニュースサイト5件でのMarkdown品質確認
- [x] SPAサイト（JS実行後DOM）からのMarkdown抽出テスト
- [x] 出力トークン数: 一般的な記事ページで2000-5000トークン以内

### Phase 11 Status: COMPLETE ✅

---

## Phase 12: Semantic Tree（構造化ページ理解）

ページの論理構造を、AIエージェントが「何ができるか」を理解するための
構造化ツリーとして出力する。フォーム入力、リンククリック等の操作可能な
要素を明示する。

### Task 12.1: Semantic Node 型定義

- [ ] `internal/semantic/tree.go`: SemanticNode 構造体
  ```go
  type SemanticNode struct {
      Role     string          // "navigation", "main", "form", "link", "button", "input", "heading", "text", "list", "image"
      Name     string          // 要素の識別名（テキスト、label、aria-label等）
      NodeID   int             // DOM上の要素ID（interact時の参照用）
      Value    string          // input の現在値、link の href 等
      Children []*SemanticNode
  }
  ```
- [ ] テスト: 構造体の生成とJSON化

### Task 12.2: DOM → Semantic Tree 変換（ランドマーク）

- [ ] `<header>` → role: "banner"
- [ ] `<nav>` → role: "navigation"
- [ ] `<main>` → role: "main"
- [ ] `<aside>` → role: "complementary"
- [ ] `<footer>` → role: "contentinfo"
- [ ] `<article>` → role: "article"
- [ ] `<section>` → role: "region"
- [ ] ARIA `role` 属性の優先適用
- [ ] テスト: ランドマーク要素のあるページ、ないページ

### Task 12.3: DOM → Semantic Tree 変換（インタラクティブ要素）

- [ ] `<a href>` → role: "link", value: href, name: テキスト
- [ ] `<button>` → role: "button", name: テキスト
- [ ] `<input type="text">` → role: "textbox", name: label/placeholder
- [ ] `<input type="checkbox">` → role: "checkbox", value: checked状態
- [ ] `<input type="radio">` → role: "radio", value: checked状態
- [ ] `<select>` → role: "combobox", value: 選択中のoption
- [ ] `<textarea>` → role: "textbox", name: label/placeholder
- [ ] `<input type="submit">` / `<button type="submit">` → role: "button"
- [ ] label要素との紐付け（for属性、ラッピング）
- [ ] テスト: 各input型、label紐付け、ネストしたフォーム

### Task 12.4: DOM → Semantic Tree 変換（コンテンツ要素）

- [ ] `<h1>`-`<h6>` → role: "heading", name: テキスト
- [ ] 連続テキストノード → role: "text", name: 結合テキスト（100文字で切る）
- [ ] `<ul>/<ol>` → role: "list", children に各 `<li>`
- [ ] `<img>` → role: "image", name: alt属性
- [ ] `<table>` → role: "table"（行数・列数をnameに含める）
- [ ] テスト: 各コンテンツ要素の変換

### Task 12.5: ツリーの圧縮・ノイズ除去

- [ ] テキストのみの中間ノード（`<div>`, `<span>`）をスキップして子を昇格
- [ ] 空テキストノードの除去
- [ ] hidden要素、aria-hidden="true" の除去
- [ ] 同一roleの連続ノードの折りたたみ（テキストブロックの結合）
- [ ] 最大深さ制限（デフォルト10）超えたら子を省略
- [ ] テスト: 圧縮前後のツリーサイズ比較

### Task 12.6: MCPツール `semantic_tree`

- [ ] `semantic_tree` ツール定義:
  ```json
  {
    "name": "semantic_tree",
    "description": "ページの論理構造を操作可能な要素付きで返す",
    "inputSchema": {
      "type": "object",
      "properties": {
        "url": { "type": "string" },
        "max_depth": { "type": "integer", "default": 10 }
      },
      "required": ["url"]
    }
  }
  ```
- [ ] 出力フォーマット: インデント付きテキスト
  ```
  [banner] サイト名
    [navigation] メインメニュー
      [link#3] ホーム → /
      [link#4] 記事一覧 → /articles
  [main]
    [heading] 記事タイトル
    [text] 本文の最初の100文字...
    [form] ログインフォーム
      [textbox#12] メールアドレス
      [textbox#13] パスワード
      [button#14] ログイン
  [contentinfo] © 2026 Example
  ```
- [ ] テスト: MCP経由でのsemantic_tree出力

### Task 12.7: `interact` ツールとの連携

- [ ] semantic_tree の NodeID を `interact` ツールの selector として使用可能にする
- [ ] NodeID → DOM要素 のマッピングテーブル管理
- [ ] ワークフロー: semantic_tree で構造把握 → interact でNodeID指定して操作
- [ ] テスト: semantic_tree取得 → フォーム入力 → 結果確認

### Task 12.8: CLIへの `--format semantic` 追加

- [ ] `uzura fetch <url> --format semantic` で動作
- [ ] インタラクティブ要素の数をサマリ表示
- [ ] `--semantic-depth N` オプション

### Task 12.9: Phase 12 Verification

- [ ] 複雑なフォームページ（ログイン、検索、入力フォーム）でのsemantic_tree品質
- [ ] SPAサイト（JS実行後）のsemantic_tree出力
- [ ] semantic_tree → interact のE2Eワークフローテスト
- [ ] 出力トークン数: 一般的なページで500-2000トークン以内
- [ ] Claude Code からの実際のワークフローテスト:
      「このサイトにログインして」→ semantic_tree → interact の流れ

---

## Phase 13: text出力ノイズ除去 & User-Agent改善

browse format=text の出力から不要な script/style コンテンツを除去し、
User-Agent を改善してボット検知を軽減する。
小さな変更で全サイトの出力品質を大幅に向上させるクイックウィン。

背景: 30サイト互換性テスト（2026-04-05）で、text出力に大量のJS/CSSが
混入する問題が判明。NHK(852KB), Amazon(752KB), Vercel(527KB) 等で
出力の大半がノイズ。また Stack Overflow, Reddit, Medium が
Cloudflare/ボット検知でブロックされた。

### Task 13.1: text出力の script/style 除去

- [ ] `internal/dom/` のテキスト抽出で `<script>`, `<style>` ノード内テキストをスキップ
- [ ] `<noscript>` ノードの扱いを決定（有用なコンテンツを含む場合がある）
- [ ] markdown変換（`internal/markdown/`）で既に実装済みの除去ロジックを参考に統一
- [ ] テスト: script/style混在HTMLでのtext出力がクリーンになることを確認
- [ ] ベンチマーク: 除去前後の出力サイズ比較

### Task 13.2: hidden要素・メタデータの除去

- [ ] `hidden` 属性を持つ要素のテキスト除去
- [ ] `aria-hidden="true"` 要素のテキスト除去
- [ ] `display:none` インラインスタイル要素の除去
- [ ] `<template>` 要素の除去
- [ ] テスト: 各hidden パターンでの除去確認

### Task 13.3: User-Agent の改善

- [ ] 現在のデフォルト User-Agent を確認
- [ ] 一般的な Chrome User-Agent 文字列に変更
  （例: `Mozilla/5.0 ... Chrome/130.0.0.0 Safari/537.36`）
- [ ] `Accept`, `Accept-Language`, `Accept-Encoding` ヘッダーの追加
- [ ] Sec-Fetch-* ヘッダー群の追加（Sec-Fetch-Mode, Sec-Fetch-Site, Sec-Fetch-Dest）
- [ ] テスト: httptest.Server でヘッダーの送信を確認

### Task 13.4: TLS フィンガープリントの改善

- [ ] `crypto/tls` の ClientHello 設定を Chrome 相当に調整
- [ ] TLS 拡張の順序と内容を一般的なブラウザに合わせる
- [ ] HTTP/2 対応の確認（ALPN ネゴシエーション）
- [ ] テスト: TLS 接続が一般ブラウザと同等のフィンガープリントになることを確認

### Task 13.5: Phase 13 Verification

- [ ] 30サイト互換性テストの再実行
- [ ] text出力サイズの削減率を測定（目標: 平均50%以上削減）
- [ ] User-Agent 改善後のボット検知回避率を確認
- [ ] Stack Overflow, Reddit, Medium への接続改善を確認
- [ ] 既存テストの全パス（`go test ./... -race`）

---

## Phase 14: Markdown品質の安定化

readability 失敗時のフォールバック戦略を改善し、
SPA サイトや構造が特殊なサイトでの markdown 出力品質を向上させる。

背景: SPA系サイト（Anthropic Docs, Claude API Docs）で markdown が
"Loading..." のみ、Angular の markdown が極少量等の問題が判明。

### Task 14.1: readability フォールバックの改善

- [ ] readability 失敗を検知する基準の明確化（出力が短すぎる、本文なし等）
- [ ] フォールバック戦略: `<main>` → `<article>` → `<body>` の順で本文領域を探索
- [ ] semantic_tree ベースの markdown 生成（ランドマーク構造を活用）
- [ ] テスト: readability 成功/失敗/部分成功の各パターン

### Task 14.2: noscript コンテンツの活用

- [ ] `<noscript>` 内のHTMLをパースしてコンテンツ候補にする
- [ ] JS無効時にnoscriptコンテンツがメインコンテンツになるサイトへの対応
- [ ] noscript と通常コンテンツの優先度判定
- [ ] テスト: Yahoo! Japan 等 noscript に有用なコンテンツがあるサイト

### Task 14.3: SPA 検出と警告

- [ ] 本文が "Loading...", "Please wait", 空の場合にSPA可能性を検出
- [ ] markdown 出力にSPA警告メタデータを付与（`spa_detected: true`）
- [ ] JS実行後のDOMを使ったmarkdown再生成の仕組み
- [ ] テスト: React CSR, Angular, Vue CSR の各パターン

### Task 14.4: markdown 出力の最適化

- [ ] SVG データURI の除去（Svelte等で混入）
- [ ] 過剰な空行・空白の正規化
- [ ] 出力トークン数の上限設定（`--max-tokens` オプション）
- [ ] テスト: 各サイトカテゴリでの品質スコア改善確認

### Task 14.5: Phase 14 Verification

- [ ] 30サイトでの markdown 品質スコア再測定（目標: 平均3.5→4.0以上）
- [ ] SPA サイト5件での markdown 改善確認
- [ ] 日本語サイト10件での markdown 品質確認
- [ ] 既存テストの全パス

---

## Phase 15: 接続安定性とレスポンスサイズ制御

Connection closed エラーの解消と、大規模サイト処理時の安定性向上。

背景: Amazon links, MDN h1 等で散発する Connection closed エラー。
MCP レスポンスが巨大（数百KB）になるケースへの対処。

### Task 15.1: MCP レスポンスサイズ制御

- [ ] レスポンスの最大サイズ設定（デフォルト: 100KB）
- [ ] 超過時の切り詰め（末尾に `[truncated]` 付与）
- [ ] `browse` ツールに `max_length` パラメータ追加
- [ ] テスト: 巨大ページでの切り詰め動作確認

### Task 15.2: query ツールの大量結果対策

- [ ] `query` ツールに `limit` パラメータ追加（デフォルト: 100件）
- [ ] `offset` パラメータでページネーション対応
- [ ] 結果件数のサマリ情報を付与（`total: 1178, returned: 100`）
- [ ] テスト: 1000件超のリンクがあるページでの動作

### Task 15.3: タイムアウトとリトライ

- [ ] MCP ツール実行のタイムアウト設定（デフォルト: 30秒）
- [ ] ネットワークリトライ（最大2回、指数バックオフ）
- [ ] タイムアウト時の部分結果返却（取得済みのDOMを返す）
- [ ] テスト: 遅延サーバー、タイムアウトシナリオ

### Task 15.4: Phase 15 Verification

- [ ] 30サイトで Connection closed エラーが0件になることを確認
- [ ] Amazon, GitHub Trending 等の巨大サイトでの安定動作
- [ ] 既存テストの全パス

---

## Phase 16: Cloudflare 基本対策

Cloudflare Managed Challenge の基本的な回避を試みる。
完全な回避は困難だが、簡易的な JS challenge への対応で一部サイトのアクセスを改善。

背景: Stack Overflow, Reddit, Medium が Cloudflare でブロック。
これらは月間数億アクセスの主要サイトであり、対応の優先度は高い。

### Task 16.1: Cloudflare 検出

- [ ] Cloudflare challenge ページの検出ロジック実装
- [ ] レスポンスヘッダー（`cf-ray`, `cf-challenge`）による検出
- [ ] HTML 内容による検出（"Enable JavaScript and cookies to continue"）
- [ ] 検出結果を MCP レスポンスのメタデータとして返却
- [ ] テスト: Cloudflare チャレンジページの検出

### Task 16.2: Cookie ベースの challenge 対応

- [ ] Cloudflare の `__cf_bm`, `cf_clearance` Cookie の処理
- [ ] 簡易 JS challenge のスクリプト実行（goja で）
- [ ] challenge 成功後の Cookie 保持とリトライ
- [ ] テスト: 簡易 challenge のシミュレーション

### Task 16.3: リクエストパターンの改善

- [ ] 初回アクセス時の振る舞いを一般ブラウザに近づける
  - favicon.ico の自動リクエスト
  - CSS/JS リソースの参照（実際にはダウンロードしない）
- [ ] リファラーヘッダーの適切な設定
- [ ] Connection: keep-alive の維持
- [ ] テスト: リクエストパターンの比較

### Task 16.4: Phase 16 Verification

- [ ] Stack Overflow, Reddit, Medium への接続テスト
- [ ] 改善率の測定（目標: 3サイト中1サイト以上で改善）
- [ ] 副作用確認: 既に動作するサイトへの影響なし
- [ ] 既存テストの全パス
