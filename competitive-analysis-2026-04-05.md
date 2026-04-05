# Uzura 競合分析レポート — 2026-04-05

## サマリ

AIエージェント向けヘッドレスブラウザ市場は急速に進化している。**Lightpanda**（27k Stars）がZig製で11倍高速・9倍省メモリを実現し、MCP対応を積極的に進めている最大の競合。**Servo**（36k Stars）はRust製の本格的Webエンジンで月次リリースとWPT 63.3%パス率を達成。**Gost-DOM**（263 Stars）はGoで最も近い競合だがV8依存（cgo必須）でTDD特化。Lightpandaの外部MCPブリッジ（gomcp）がアーカイブされ、ブラウザ本体にMCPが統合される流れが確認された。Uzuraの「pure Go・cgo不要・AIエージェント特化」というポジションは依然として差別化要因だが、MCP対応とパフォーマンス実証が急務。

## 詳細比較表

| 項目 | Uzura | Lightpanda | Servo | Gost-DOM | mizchi/tui-poc |
|------|-------|-----------|-------|----------|---------------|
| **Stars** | — | 27,166 | 36,195 | 263 | 1 |
| **言語** | Go (pure, no cgo) | Zig + Rust | Rust | Go + cgo (V8) | Rust |
| **ライセンス** | — | AGPL-3.0 | MPL-2.0 | MIT | MIT |
| **最終コミット** | 2026-04-05 | 2026-04-04 | 2026-04-05 | 2026-02-19 | 2025-07-17 |
| **最新リリース** | — | v0.2.8 (2025-04-02) | v0.0.6 (2026-03-31) | v0.11.0 (2026-02-19) | — |
| **リリース頻度** | — | 1-2週間毎 | 月次 (4-6週) | 不定期 (月2回程度) | — |
| **JSエンジン** | goja (pure Go) | V8 | SpiderMonkey系(独自) | V8 | QuickJS |
| **HTMLパーサ** | golang.org/x/net/html | html5ever (Rust) | 独自実装 | 独自 | html5ever |
| **CDP対応** | Phase 6+ | ✅ | WebDriver | ✗ | ✅ |
| **MCP対応** | — | ✅ (ネイティブ) | ✗ | ✗ | ✗ |
| **cgo依存** | なし | N/A (Zig) | N/A (Rust) | あり (V8) | N/A (Rust) |
| **対応OS** | クロスプラットフォーム | Linux, macOS, WSL2 | Linux, macOS, Win, Android, OpenHarmony | Linux, macOS | Linux, macOS |
| **Open Issues** | — | 92 | 3,065 | 66 | 0 |
| **Closed Issues** | — | 279 | 12,618 | 35 | 0 |
| **コントリビュータ** | — | 多数 | 多数 | 4 | 1 |

## Lightpanda gomcp（アーカイブ済み）

| 項目 | 値 |
|------|-----|
| Stars | 63 |
| 言語 | Go (94.8%) |
| ライセンス | Apache-2.0 |
| 状態 | **アーカイブ（2026-03-13）** |
| 理由 | Lightpanda本体にMCPサーバーがネイティブ統合されたため不要に |
| 主要依存 | chromedp, html-to-markdown, goquery, cascadia |

**示唆**: 外部MCPブリッジではなくブラウザ本体にMCPを統合する流れが主流化。

## chromedp (nickmassaro/chromedp)

指定されたURL `github.com/nickmassaro/chromedp` は404。正しいリポジトリは `github.com/chromedp/chromedp`（Go製CDPクライアント）の可能性が高い。

## パフォーマンスベンチマーク（Lightpanda）

AWS EC2 m5.large上、Puppeteerで100ページをリクエスト（933ページでテスト）:

| 指標 | Lightpanda | Chrome Headless | 倍率 |
|------|-----------|----------------|------|
| 実行時間 | 2.3秒 | 25.2秒 | **11x 高速** |
| ピークメモリ | 24 MB | 207 MB | **9x 省メモリ** |

## Servo WPTパス率

2026年最新（Linux 22.04で毎日実行）:

| テストスイート | パス率 | サブテスト通過率 |
|---------------|--------|-----------------|
| **全WPTテスト** | **63.3%** | 92.8% |
| /css/ (全体) | 69.4% | 63.9% |
| /css/CSS2/ | 92.7% | 92.2% |
| /css/css-flexbox/ | 80.4% | 63.2% |
| /css/css-grid/ | 36.3% | 44.0% |
| /WebCryptoAPI/ | 89.5% | 94.1% |
| /webdriver/ | 79.6% | 89.6% |

## Gost-DOM 開発ブログ

dev.to シリーズ「Building a Headless Browser in Go」（著者: Peter Stroiman）:

| # | タイトル | 日付 |
|---|---------|------|
| 1 | "Go-DOM - A headless browser written in Go." | 2024-11-06 |
| 2 | "Go-DOM - 1st major milestone" | 2024-11-18 |

2記事のみで更新停止。

## 注目すべき変化

- **Lightpanda MCP統合**: 2026-04-04のコミットで `hover`, `press`, `selectOption`, `setChecked` のMCPアクションツールを追加。外部ブリッジ（gomcp）をアーカイブし、本体統合に一本化。
- **Servo月次リリース定着**: v0.0.2（2025-11）→ v0.0.6（2026-03）で安定した月次リリースサイクルを確立。
- **Gost-DOM活動低下**: 最終リリースが2026-02-19（v0.11.0）、約2ヶ月間新リリースなし。
- **gomcpのアーカイブ**: MCPブリッジパターンの終焉を示唆。ブラウザ本体にMCPを組み込む方がアーキテクチャ的に優位。

## Uzuraへの示唆

### 1. MCP対応を最優先に
LightpandaがネイティブMCP対応を急速に進めている。Uzuraも早期にMCPサーバー機能を本体に組み込むべき。gomcpのアーカイブは「外部ブリッジではなくネイティブ統合」が正解であることを示している。

### 2. Pure Go・no cgoは強力な差別化
Gost-DOMはV8依存（cgo必須）で、ビルドの複雑さやクロスコンパイルの困難さが課題。Uzuraのgoja採用によるpure Goアプローチは、デプロイの容易さとポータビリティで明確に優位。

### 3. パフォーマンスベンチマークの公開
Lightpandaは「Chrome比11x高速」を前面に押し出している。Uzuraも同様のベンチマーク（vs Chrome headless, vs Lightpanda）を公開し、特にAIエージェントユースケースでの優位性を数値で示すべき。

### 4. ライセンス優位性
LightpandaのAGPL-3.0は商用利用にハードル。Uzuraがより許容的なライセンス（MIT/Apache-2.0）を採用すれば、企業ユーザーへの訴求力で差別化できる。

### 5. Windows ネイティブ対応
LightpandaはWSL2のみ。Uzuraはpure Goのため、Windows含む全プラットフォームでネイティブ動作可能。これは実用上の大きなアドバンテージ。

### 6. AIエージェント特化の深堀り
Servo・Lightpandaは汎用ブラウザ寄り。Uzuraは「AIエージェントに最適化」（semantic tree出力、トークン効率、構造化データ抽出）の方向をさらに深め、ニッチでの圧倒的優位を確立すべき。
