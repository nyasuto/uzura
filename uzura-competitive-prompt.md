# Uzura 競合調査プロンプト（Claude Code用）

以下のタスクを Uzura MCP ツールを使って実行してください。

## 調査対象

1. **Lightpanda** — https://github.com/lightpanda-io/browser
2. **Lightpanda gomcp** — https://github.com/lightpanda-io/gomcp
3. **Servo** — https://github.com/servo/servo
4. **Gost-DOM** — https://github.com/gost-dom/browser
5. **mizchi/tui-poc** — https://github.com/mizchi/tui-poc
6. **chromedp** — https://github.com/nickmassaro/chromedp (CDPクライアント参考)

## 各リポジトリから収集する情報

各URLに対して `browse` ツール（format: markdown）で以下を収集：

### GitHub READMEから
- Stars数（ページ上部）
- 言語構成
- 最終コミット日
- ライセンス
- 主要依存ライブラリ
- サポートするプロトコル（CDP, MCP等）
- 対応プラットフォーム

### リリースページから（{repo_url}/releases）
- 最新リリースバージョンと日付
- リリース頻度（月次/不定期等）

### Issuesから（{repo_url}/issues のカウント）
- オープンイシュー数
- クローズ済みイシュー数

## 追加調査

### Lightpanda のベンチマーク
- `browse` で https://lightpanda.io/ を取得
- パフォーマンス数値（実行速度、メモリ使用量）を抽出

### Servo のWPTパス率
- `browse` で https://servo.org/wpt を取得（取得可能なら）
- 最新のWPTパス率数値を抽出

### Gost-DOM の記事
- `browse` で https://dev.to/stroiman/series/29492 を取得
- 開発ブログの最新投稿タイトルと日付を一覧化

## 出力形式

収集した情報を以下の形式でMarkdownファイルに出力してください。
ファイル名: `competitive-analysis-{YYYY-MM-DD}.md`

```markdown
# Uzura 競合分析レポート — {日付}

## サマリ

（3-5文で全体的な競合状況を要約）

## 詳細比較表

| 項目 | Uzura | Lightpanda | Servo | Gost-DOM |
|------|-------|-----------|-------|---------|
| Stars | ... | ... | ... | ... |
| 言語 | ... | ... | ... | ... |
| （以下続く） |

## 注目すべき変化

（前回との差分、新機能、新リリース等）

## Uzuraへの示唆

（競合の動きからUzuraが取るべきアクション）
```

## 実行のヒント

- GitHub のリポジトリページは `query` ツールで特定要素を取得すると効率的
  - Stars: `query` selector `.Counter` or `#repo-stars-counter-star`
  - About: `query` selector `.f4.my-3`
- 大量ページの場合は `browse` → `semantic_tree` の順で構造を把握してから `query` で詳細取得
- レート制限に注意: 各リクエスト間に1-2秒のインターバルを取る
