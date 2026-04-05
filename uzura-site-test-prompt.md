# Uzura 実サイト互換性テスト（Claude Code用）

Uzura MCP ツールを使って、以下のサイトを巡回し互換性レポートを作成してください。

## テスト対象サイト（30サイト）

### 日本語サイト（10）
1. https://www3.nhk.or.jp/news/ — NHKニュース（動的コンテンツ）
2. https://qiita.com/ — Qiita（SPA寄り）
3. https://zenn.dev/ — Zenn（Next.js）
4. https://www.yahoo.co.jp/ — Yahoo! Japan（大規模ポータル）
5. https://tabelog.com/ — 食べログ（構造化データ多め）
6. https://suumo.jp/chintai/kanagawa/ — SUUMO（不動産、フォーム多め）
7. https://github.co.jp/ — GitHub Japan
8. https://www.amazon.co.jp/ — Amazon（巨大DOM）
9. https://developer.mozilla.org/ja/ — MDN日本語
10. https://connpass.com/ — connpass（イベントサイト）

### 英語テックサイト（10）
11. https://news.ycombinator.com/ — HN（シンプルHTML）
12. https://github.com/trending — GitHub Trending
13. https://dev.to/ — DEV Community
14. https://stackoverflow.com/ — Stack Overflow
15. https://docs.anthropic.com/ — Anthropic Docs
16. https://go.dev/ — Go公式
17. https://www.rust-lang.org/ — Rust公式
18. https://lightpanda.io/ — Lightpanda（競合）
19. https://servo.org/ — Servo（競合）
20. https://platform.claude.com/docs/ — Claude API Docs

### SPAサイト（JS実行必須）（5）
21. https://react.dev/ — React公式
22. https://vuejs.org/ — Vue公式
23. https://angular.dev/ — Angular公式
24. https://svelte.dev/ — Svelte公式
25. https://vercel.com/ — Vercel

### 大規模・複雑（5）
26. https://en.wikipedia.org/wiki/Web_browser — Wikipedia
27. https://www.reddit.com/ — Reddit（動的）
28. https://medium.com/ — Medium（ペイウォール）
29. https://twitter.com/ — X/Twitter（SPA）
30. https://www.google.com/search?q=headless+browser — Google検索結果

## 各サイトのテスト手順

### Step 1: browse (text)
```
browse url={url} format=text
```
- 成否を記録
- エラーがあればエラー内容を記録

### Step 2: browse (markdown) 
```
browse url={url} format=markdown
```
- markdown出力のトークン数（概算: 文字数÷4）
- タイトル・本文が正しく抽出されたか
- リンクが正しくmarkdown化されたか

### Step 3: semantic_tree
```
semantic_tree url={url}
```
- ランドマーク要素の検出数
- インタラクティブ要素（リンク、ボタン、フォーム）の検出数
- ツリーの深さ

### Step 4: query（特定要素）
```
query url={url} selector="h1"
query url={url} selector="a" attribute="href"
query url={url} selector="form"
```
- h1の取得
- リンク数
- フォーム数

## 出力形式

### サイトごとの結果
```markdown
### {番号}. {サイト名} ({URL})

| テスト | 結果 | 備考 |
|--------|------|------|
| browse (text) | ✅/❌ | {詳細} |
| browse (markdown) | ✅/❌ | ~{N}トークン |
| semantic_tree | ✅/❌ | {ランドマーク数}/{インタラクティブ数} |
| query h1 | ✅/❌ | "{取得したh1テキスト}" |
| query links | ✅/❌ | {リンク数}件 |
```

### サマリ
```markdown
## 互換性サマリ

- テスト総数: 30サイト × 5テスト = 150
- 成功: {N}/150 ({N/150*100}%)
- 失敗: {N}/150
- 日本語サイト成功率: {N}%
- SPAサイト成功率: {N}%
- markdown品質スコア（主観1-5）: 平均{N}

## カテゴリ別結果
| カテゴリ | 成功率 | 主な失敗原因 |
|----------|--------|-------------|
| 日本語 | {N}% | ... |
| 英語テック | {N}% | ... |
| SPA | {N}% | ... |
| 大規模 | {N}% | ... |

## 改善提案
（失敗パターンからの改善優先度リスト）
```

## 注意事項

- レート制限: サイト間に2秒の間隔を取る
- タイムアウト: 各リクエストは30秒以内
- robots.txt: --obey-robots を有効にする
- 失敗時: エラーメッセージを記録して次のサイトへ進む
- 結果ファイル: `site-compatibility-{YYYY-MM-DD}.md` として保存
