# Uzura 競合ベンチマーク — スクラッチビルドブラウザ比較

最終更新: 2026-04-05

## 対象プロジェクト

| | Uzura | Lightpanda | Servo | Gost-DOM |
|---|---|---|---|---|
| **リポジトリ** | nyasuto/uzura | lightpanda-io/browser | servo/servo | gost-dom/browser |
| **言語** | Go (pure, cgo不要) | Zig + V8 + Rust (html5ever) | Rust | Go (V8 + Goja移行中) |
| **GitHub Stars** | — | ~18,000 | ~35,900 | ~200 |
| **開始年** | 2026 | 2022 | 2012 (2023再起動) | 2024 |
| **ライセンス** | MIT | AGPL-3.0 | MPL-2.0 | MIT |
| **ステータス** | Phase 12完了 | Beta (~95%互換) | v0.0.4 (実験的) | 開発中 |

## 設計思想

| | Uzura | Lightpanda | Servo | Gost-DOM |
|---|---|---|---|---|
| **ターゲット** | AIエージェント/MCP | AIエージェント/スクレイピング | 汎用エンジン/組み込み | Go TDDワークフロー |
| **レンダリング** | なし（DOM+JSのみ） | なし（DOM+JSのみ） | フル（CSS+GPU） | なし（DOM+JSのみ） |
| **ビルド** | `go build` 1コマンド | Zig + Rust + C依存 | Rust + 大量依存 | `go build` (cgo必要) |
| **バイナリサイズ** | シングル、~15MB想定 | ~24MB | ~100MB+ | シングル (cgo) |
| **cgo依存** | なし | N/A (Zig) | N/A (Rust) | あり (V8) → Goja移行中 |

## 機能比較

| 機能 | Uzura | Lightpanda | Servo | Gost-DOM |
|---|---|---|---|---|
| **HTML5パーサー** | ✅ golang.org/x/net/html | ✅ html5ever (Rust) | ✅ html5ever | ✅ golang.org/x/net/html |
| **DOMツリー** | ✅ WHATWG準拠 | ✅ 独自Zig実装 | ✅ WHATWG準拠 | ✅ Web IDL codegen |
| **CSS解析** | ❌ 意図的に除外 | ❌ 意図的に除外 | ✅ Stylo (Firefox共有) | ❌ |
| **CSSレイアウト** | ❌ | ❌ | ✅ Layout 2020 | ❌ |
| **GPUレンダリング** | ❌ | ❌ | ✅ WebGPU対応 | ❌ |
| **JavaScript** | ✅ Goja (pure Go, ES6 ~80%) | ✅ V8 (フルES2024) | ✅ SpiderMonkey | ✅ V8 / Goja (移行中) |
| **CDP** | ✅ Page/DOM/Runtime/Network/Fetch | ✅ フル対応 | ❌ | ❌ |
| **Puppeteer互換** | ✅ | ✅ | ❌ | ❌ |
| **Playwright互換** | 🔶 CDP経由で限定的 | 🔶 対応中 (注意事項あり) | ❌ | ❌ |
| **MCP** | ✅ 内蔵 (uzura mcp) | ✅ gomcp (別バイナリ) | ❌ | ❌ |
| **Markdown出力** | ✅ readability統合 | ✅ ネイティブ対応 | ❌ | ❌ |
| **Semantic Tree** | ✅ アクセシビリティツリー | ❌ | 🔶 部分的 | ❌ |
| **ネットワークインターセプト** | ✅ Fetch/fulfill/fail | ✅ | ❌ | ✅ Go http.Handler直結 |
| **マルチページ** | ✅ BrowserContext分離 | 🔶 1接続1ページ制限 | ✅ | ❌ |
| **Cookie管理** | ✅ | ✅ | ✅ | ❌ |
| **robots.txt遵守** | ✅ | ✅ | ❌ | N/A |
| **WPTテスト** | ✅ ランナー内蔵 | ✅ 外部Goランナー | ✅ (内部ダッシュボード) | ❌ |
| **スクリーンショット** | ❌ | ❌ | ✅ | ❌ |
| **イベントシステム** | ✅ バブリング+キャプチャ | ✅ | ✅ | ✅ |
| **MutationObserver** | ✅ | ❌ 不明 | ✅ | ❌ |

## パフォーマンス特性

| | Uzura | Lightpanda | Servo | Gost-DOM |
|---|---|---|---|---|
| **起動時間** | ~18ms | <100ms | 数秒 | ~50ms (V8初期化) |
| **メモリ/インスタンス** | ~10-20MB想定 | ~24MB | ~200MB+ | ~50MB (V8) |
| **並行インスタンス (8GB)** | 200+想定 | ~140 | ~15 | ~50 |
| **100ページフェッチ** | 未計測 | ~2.3s (ローカル) | N/A | 未計測 |

## AI統合の深さ

| | Uzura | Lightpanda | Servo | Gost-DOM |
|---|---|---|---|---|
| **MCP内蔵** | ✅ 同一バイナリ | ❌ 別バイナリ (gomcp) | ❌ | ❌ |
| **MCP stdio** | ✅ | ✅ (gomcp経由) | ❌ | ❌ |
| **MCP SSE** | ❌ | ✅ (gomcp経由) | ❌ | ❌ |
| **LLM向け出力** | ✅ markdown/semantic | ✅ markdown | ❌ | ❌ |
| **ツール数** | 5 (browse/evaluate/query/interact/semantic_tree) | 6+ | ❌ | ❌ |
| **Claude Code統合** | ✅ .claude.json 1行 | ✅ claude_desktop_config | ❌ | ❌ |

## Uzura固有の強み

1. **Pure Go / ゼロcgo**: Goがビルドできる全環境で動く。クロスコンパイルも `GOOS=linux go build` のみ
2. **MCP内蔵**: Lightpandaはgomcp（Go別バイナリ）+ Lightpanda本体（Zig）の2プロセス構成。Uzuraは1バイナリ1プロセス
3. **Semantic Tree**: Lightpandaにない独自機能。AIエージェントがページの操作可能要素を把握するのに最適
4. **教育的構造**: 12フェーズの段階的設計で、ブラウザの内部構造を学びながら拡張可能

## Uzuraの弱み・改善ポイント

1. **JS互換性**: Goja (ES6 ~80%) vs V8 (ES2024完全対応)。最新JSフレームワークのSPA対応に限界
2. **Web API網羅性**: Lightpandaは主要Web APIを継続的に追加中。Uzuraは必要最小限のサブセット
3. **コミュニティ**: Lightpandaの18,000 starsに対し、個人プロジェクト
4. **実サイト互換性**: Lightpandaの~95%に対し、未計測

## 参考リンク

- Lightpanda: https://github.com/lightpanda-io/browser
- Lightpanda gomcp: https://github.com/lightpanda-io/gomcp
- Servo: https://github.com/servo/servo
- Servo WPT Dashboard: https://servo.org/wpt
- Gost-DOM: https://github.com/gost-dom/browser
- mizchi/tui-poc: https://github.com/mizchi/tui-poc
- Web Platform Tests: https://github.com/web-platform-tests/wpt
