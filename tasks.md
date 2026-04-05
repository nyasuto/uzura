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

- [x] `Text` 構造体（`baseNode` 埋め込み）
- [x] `Comment` 構造体（`baseNode` 埋め込み）
- [x] `NodeName()` → `#text` / `#comment`
- [x] `TextContent()` の取得・設定
- [x] `CloneNode()` の実装
- [x] テスト: 生成、内容get/set、クローン

### Task 1.4: Element Implementation

- [x] `Element` 構造体（`baseNode` 埋め込み）
- [x] `TagName`（大文字）と `LocalName`（小文字）
- [x] 属性: `GetAttribute`, `SetAttribute`, `HasAttribute`, `RemoveAttribute`
- [x] `Id()`, `ClassName()` ショートハンド
- [x] テスト: 属性CRUD、大文字小文字

### Task 1.5: Document Implementation

- [x] `Document` 構造体（`baseNode` 埋め込み）
- [x] `DocumentElement()`, `Head()`, `Body()`, `Title()`
- [x] `CreateElement`, `CreateTextNode`, `CreateComment`
- [x] `GetElementById`, `GetElementsByTagName`, `GetElementsByClassName`
- [x] テスト: 要素生成、id/tag/classでの検索

### Task 1.6: HTML Parser → DOM Tree Conversion

- [x] `internal/html/parser.go` — `golang.org/x/net/html` のアダプター
- [x] `Parse(r io.Reader) (*dom.Document, error)`
- [x] `html.Node` → `dom.Node` の再帰変換
- [x] 不正HTML（暗黙タグ挿入）の処理確認
- [x] テスト: 基本HTML、ネストテーブル、閉じタグ欠落、空ドキュメント

### Task 1.7: DOM Serializer

- [x] `Serialize(node Node) string` — DOM → HTML文字列
- [x] void要素（`<br>`, `<img>`等）、属性エスケープ、テキストエスケープ
- [x] raw text要素（`<script>`, `<style>`）
- [x] `InnerHTML()`, `OuterHTML()`
- [x] ラウンドトリップテスト: parse → serialize → parse → deep-equal

### Task 1.8: CLI `parse` コマンド仕上げ

- [x] stdin / ファイル入力対応
- [x] `--format text` インデント付きツリー表示（デフォルト）
- [x] `--format json` JSON構造
- [x] `--format html` 再シリアライズHTML
- [x] stderr/stdoutの分離、終了コード

### Task 1.9: Phase 1 検証

- [x] `go test ./... -race` 全パス
- [x] `go vet ./...` クリーン
- [x] ベンチマーク: 100KB HTML を 50ms以内でパース（実測: ~0.8ms）
- [x] `curl -s https://example.com | ./uzura parse` 動作確認

### Phase 1 Status: COMPLETE ✅

---

## 開発エコシステム整備

### Task 0.1: Makefile

- [x] `Makefile` を作成
- [x] `make build` — `go build -o uzura ./cmd/uzura`
- [x] `make test` — `go test ./... -race`
- [x] `make bench` — `go test ./... -bench=. -benchmem`
- [x] `make vet` — `go vet ./...`
- [x] `make lint` — `golangci-lint run`（未インストールならスキップ）
- [x] `make fmt` — `gofmt -w .` + `goimports -w .`
- [x] `make clean` — バイナリ・キャッシュ削除
- [x] `make quality` — `fmt` + `vet` + `lint` + `test` を一括実行
- [x] `make all` — `quality` + `build`（デフォルトターゲット）
- [x] `make cover` — カバレッジレポート生成（`go test -coverprofile` + `go tool cover -html`）

### Task 0.2: CI用ヘルパー

- [x] `.golangci.yml` — lintルール設定（unused, errcheck, govet, staticcheck等）
- [x] `.editorconfig` — インデント・改行コード統一

### Task 0.3: Git hooks

- [x] `make install-hooks` — pre-commit hookのインストール（`make quality` を実行）
- [x] `.gitignore` の整備（`uzura` バイナリ、`*.out`, `coverage.html` 等を追加）

---

## Phase 2: HTTP Fetcher + Document Loading

### Task 2.1: Basic HTTP Fetcher

- [x] `internal/network/fetcher.go`
- [x] `Fetch(url string) (*Response, error)` タイムアウト付き
- [x] User-Agent設定、リダイレクト追跡（最大10回）
- [x] `net/http/httptest` でテスト

### Task 2.2: Content-Type and Encoding

- [x] Content-Typeヘッダーからcharset検出
- [x] `<meta charset>` タグからのcharset検出
- [x] `golang.org/x/text/encoding` でUTF-8変換
- [x] テスト: Shift-JIS, EUC-JP, ISO-8859-1

### Task 2.3: Cookie Jar

- [x] `net/http/cookiejar` 統合
- [x] セッション内のCookie永続化
- [x] テスト: set/get, リダイレクト跨ぎ

### Task 2.4: Document Loading Pipeline

- [x] Fetch → Decode → Parse → Document パイプライン
- [x] エラーハンドリング（ネットワーク、タイムアウト、不正HTML）
- [x] httptest での結合テスト

### Task 2.5: CLI `fetch` コマンド

- [x] `uzura fetch <url>`
- [x] `--format`, `--timeout`, `--user-agent` フラグ
- [x] 動作確認: `uzura fetch https://example.com`

### Task 2.6: robots.txt

- [x] `--obey-robots` フラグ
- [x] robots.txtパースとallow/disallowチェック

### Task 2.7: Phase 2 検証

- [x] テスト全パス、日本語サイトのエンコーディング確認

### Phase 2 Status: COMPLETE ✅

---

## Phase 3: CSS Selector Engine

### Task 3.1: cascadia統合

- [x] `github.com/andybalholm/cascadia` 依存追加
- [x] `internal/css/selector.go` アダプター

### Task 3.2: querySelector / querySelectorAll

- [x] Element/DocumentにquerySelector/querySelectorAllを追加
- [x] テスト: tag, class, id, 属性, 結合子, 擬似クラス

### Task 3.3: matches / closest

- [x] `Element.Matches()`, `Element.Closest()`

### Task 3.4: Phase 3 検証

- [x] 複雑セレクターのテスト、大規模DOMでのベンチマーク

### Phase 3 Status: COMPLETE ✅

---

## Phase 4: DOM API（Web標準準拠）

- [x] Node: appendChild, removeChild, insertBefore, cloneNode（Phase 1で基本実装済み、拡張）
- [x] Element: classList, dataset, innerHTML setter（パーサー連携）
- [x] Document: createDocumentFragment, importNode
- [x] MutationObserver 基本実装
- [x] WPT `dom/nodes/` テストのパス

### Phase 4 Status: COMPLETE ✅

---

## Phase 4.5: アーキテクチャ改善（JS/CDP準備）

Gemini / o3 レビューで指摘された構造的改善。Phase 5（JS）以降で手戻りしないよう先に対処する。

### Task 4.5.1: QueryEngine インターフェース化

- [x] `dom.QueryEngine` インターフェースを定義（QuerySelector, QuerySelectorAll, Matches, Closest）
- [x] `Document` / `Element` 生成時にインターフェースを注入する形に変更
- [x] `query.go` の関数変数パターンを廃止
- [x] `internal/css` が `QueryEngine` を実装
- [x] 既存テストが全パス

### Task 4.5.2: Observer パターン（DOMミューテーション検知）

- [x] `dom.MutationRecord` 型を定義（childList, attributes, characterData）
- [x] `dom.MutationObserver` + `MutationCallback` 型を定義
- [x] `baseNode` の変更メソッド（AppendChild, RemoveChild, SetAttribute等）からイベント発火
- [x] テスト: ノード追加/削除/属性変更でコールバックが呼ばれることを検証（18テスト）
- [x] 将来のCDP（DOM.documentUpdated）やJS EventListener の基盤となる

### Task 4.5.3: オーケストレーション層（internal/page）

- [x] `internal/page` パッケージを新設
- [x] `Page` 構造体: DOM, Network, （将来の）JSContext を統括
- [x] `Page.Navigate(ctx, url)` — Fetch→Decode→Parse→DOM のライフサイクル管理
- [x] `context.Context` を全ブロッキング操作に導入（Fetcher.FetchContext, LoadDocumentContext）
- [x] `network/loader.go` の責務を `page` に移行、CLI を Page 経由に更新
- [x] テスト: httptest.Server を使ったページロードの E2E テスト（8テスト）

### Task 4.5.4: エラー型の整備

- [x] sentinel errors を定義（`ErrRobotsDisallowed`, `ErrTooManyRedirects`, `ErrInvalidSelector`）
- [x] `FetchError` 構造体（StatusCode, URL を保持、Unwrap対応）
- [x] `errors.Is()` / `errors.As()` で分岐可能にする
- [x] 既存のエラーハンドリングをリファクタリング（robots.go, fetcher.go, css/selector.go）

---

## GitHub Actions エコシステム整備

### Task CI.0: Makefile の `vet` 削除

- [x] `make quality` から `vet` ターゲットを除外（`golangci-lint` に一本化）
- [x] Makefile から `vet` ターゲット自体を削除
- [x] `golangci-lint` を必須化（未インストール時はスキップではなくエラー）
- [x] CLAUDE.md の `go vet ./...` 記載を更新

### Task CI.1: 基本CIワークフロー

- [x] `.github/workflows/ci.yml` を作成
- [x] Push / PR トリガーで lint + test を実行
- [x] Go バージョンマトリクス（1.26、go.mod要件に合わせ）
- [x] キャッシュ設定（`actions/setup-go` + `golangci-lint-action` の自動キャッシュ）
- [x] Readmeを作成
- [x] ステータスバッジを README に追加（README作成時に対応）

### Task CI.2: カバレッジレポート

- [x] `make cover` でカバレッジ生成（既存）
- [x] Codecov に連携（codecov-action@v5）
- [x] PR にカバレッジ差分コメントを自動投稿（Codecov側のPR統合で対応）
- [x] カバレッジバッジを README に追加（README作成時に対応）

### Task CI.3: リリース自動化

- [x] `.github/workflows/release.yml` を作成
- [x] タグプッシュ（`v*`）トリガーで GoReleaser 実行
- [x] マルチプラットフォームビルド（linux/amd64, linux/arm64, darwin/arm64, darwin/amd64）
- [x] GitHub Releases にバイナリ・チェックサムを自動アップロード
- [x] `.goreleaser.yml` の作成

### Task CI.4: Dependabot / セキュリティ

- [x] `.github/dependabot.yml` を作成（Go モジュール + GitHub Actions の自動更新）
- [x] `govulncheck` をCIに追加（既知の脆弱性チェック）

### Task CI.5: PR自動化

- [x] golangci-lint の PR インライン指摘（`only-new-issues` で新規問題のみ表示）
- [x] ベンチマーク結果の PR コメント自動投稿（`benchstat` 差分）

## Phase 5: JavaScript Execution（Goja統合）

### Phase 5.1: VM基盤 + グローバルオブジェクト

- [x] `github.com/dop251/goja` 依存追加
- [x] `internal/js/vm.go` — Goja Runtime のラッパー（生成、実行、リセット）
- [x] サンドボックス: ファイルI/O・ネットワーク等のネイティブアクセスを遮断
- [x] `window` / `globalThis` オブジェクトの基本構造
- [x] `console.log` / `console.warn` / `console.error`（Go の io.Writer に出力）
- [x] テスト: JS式の評価、console出力のキャプチャ、禁止操作のエラー

### Phase 5.2: Document バインディング（読み取り系）

- [x] `document` オブジェクトを goja に登録
- [x] `document.getElementById`, `document.querySelector`, `document.querySelectorAll`
- [x] `document.getElementsByTagName`, `document.getElementsByClassName`
- [x] `document.title`, `document.documentElement`, `document.head`, `document.body`
- [x] Element プロキシ: `tagName`, `id`, `className`, `textContent`, `innerHTML`（getter）
- [x] `element.getAttribute`, `element.hasAttribute`
- [x] `element.querySelector`, `element.querySelectorAll`, `element.matches`, `element.closest`
- [x] NodeList / HTMLCollection の JS 表現（length, index アクセス, forEach）
- [x] テスト: JS からの DOM クエリが Go 側の DOM と一致することを検証

### Phase 5.3: Document バインディング（書き込み系）

- [x] `document.createElement`, `document.createTextNode`, `document.createDocumentFragment`
- [x] `element.setAttribute`, `element.removeAttribute`
- [x] `element.textContent` setter, `element.innerHTML` setter
- [x] `node.appendChild`, `node.removeChild`, `node.insertBefore`, `node.replaceChild`
- [x] `element.classList`（add, remove, toggle, contains）の JS バインディング
- [x] `element.dataset` の JS Proxy バインディング
- [x] テスト: JS で DOM を変更 → Go 側の DOM ツリーに反映されることを検証

### Phase 5.4: イベントシステム

- [x] `EventTarget` インターフェース（addEventListener, removeEventListener, dispatchEvent）
- [x] `Event` オブジェクト（type, target, currentTarget, bubbles, cancelable, preventDefault, stopPropagation）
- [x] イベントバブリング・キャプチャリングの実装
- [x] `document`, `window`, 各 `Element` を EventTarget として登録
- [x] テスト: イベント発火、バブリング、preventDefault の動作検証

### Phase 5.5: タイマーとイベントループ

- [x] イベントループの基本構造（タスクキュー + 実行サイクル）
- [x] `setTimeout` / `clearTimeout`
- [x] `setInterval` / `clearInterval`
- [x] Go goroutine との同期（goja は単一スレッド、タイマーコールバックはループ内で実行）
- [x] テスト: タイマーの順序保証、クリア動作、ネストしたタイマー

### Phase 5.6: `<script>` タグ実行

- [x] HTML パーサーからの `<script>` タグ検出
- [x] インラインスクリプトの実行（document 解析順）
- [x] `defer` 属性のセマンティクス（defer スクリプトは後回し実行）
- [x] スクリプトエラー時のハンドリング（他のスクリプトは続行）
- [x] テスト: 複数スクリプトの実行順序、defer の動作、エラー継続

### Phase 5.7: WPT テスト + 検証

- [x] イベントシステムの包括的テスト（バブリング、キャプチャ、preventDefault、stopPropagation）
- [x] `go test ./... -race` 全パス
- [x] JS 実行のベンチマーク（単純式: ~1μs, DOMクエリ: ~9μs, DOM変更: ~32μs）

### Phase 5 Status: COMPLETE ✅

---

## Phase 6: CDP WebSocket Server

### Phase 6.1: WebSocket 基盤 + ディスカバリー

- [x] `nhooyr.io/websocket` 依存追加
- [x] `internal/cdp/server.go` — WebSocket サーバー起動（`uzura serve --port 9222`）
- [x] CDP メッセージのJSON-RPC ディスパッチャー（id, method, params → result/error）
- [x] `/json/version` エンドポイント（ブラウザメタ情報）
- [x] `/json/list` エンドポイント（ターゲット一覧）
- [x] `/json/protocol` エンドポイント（サポートドメイン一覧）
- [x] テスト: WebSocket 接続、JSON-RPC の送受信、ディスカバリーの HTTP レスポンス

### Phase 6.2: Page ドメイン

- [x] `Page.enable` — ページイベントの有効化
- [x] `Page.navigate` — URL ナビゲーション（internal/page と連携）
- [x] `Page.getFrameTree` — フレームツリー取得（単一フレーム）
- [x] `Page.loadEventFired` イベント送信
- [x] `Page.domContentEventFired` イベント送信
- [x] テスト: ナビゲーション → ロード完了イベントの一連の流れ

### Phase 6.3: DOM ドメイン

- [x] `DOM.enable` — DOM イベントの有効化
- [x] `DOM.getDocument` — ルートノードの返却（depth 制御）
- [x] `DOM.querySelector` / `DOM.querySelectorAll`
- [x] `DOM.getOuterHTML` / `DOM.setOuterHTML`
- [x] `DOM.getAttributes` / `DOM.setAttributeValue` / `DOM.removeAttribute`
- [x] `DOM.requestChildNodes` — 子ノードのストリーミング取得
- [x] `DOM.documentUpdated` イベント（MutationObserver と連携）
- [x] nodeId の管理（Go DOM ノード ↔ CDP nodeId のマッピング）
- [x] テスト: DOM クエリ・変更が CDP 経由で正しく動作

### Phase 6.4: Runtime ドメイン

- [x] `Runtime.enable` — ランタイムイベントの有効化
- [x] `Runtime.evaluate` — JS 式の評価（Phase 5 の VM と連携）
- [x] `Runtime.callFunctionOn` — リモートオブジェクトへの関数呼び出し
- [x] RemoteObject のシリアライズ（type, value, objectId）
- [x] `Runtime.consoleAPICalled` イベント（console.log 等の転送）
- [x] `Runtime.exceptionThrown` イベント
- [x] テスト: JS 評価、戻り値のシリアライズ、エラーハンドリング

### Phase 6.5: Network ドメイン

- [x] `Network.enable` — ネットワークイベントの有効化
- [x] `Network.requestWillBeSent` イベント
- [x] `Network.responseReceived` イベント
- [x] `Network.loadingFinished` / `Network.loadingFailed` イベント
- [x] `Network.getResponseBody` — レスポンスボディの取得
- [x] リクエスト ID の管理とイベントの時系列保証
- [x] テスト: ページロード中のネットワークイベント発火の検証

### Phase 6.6: Puppeteer 接続テスト

- [x] Puppeteer (`puppeteer-core`) からの接続確認スクリプト作成
- [x] `puppeteer.connect({ browserWSEndpoint })` での接続
- [x] 基本操作の E2E テスト（navigate → querySelector → getOuterHTML）
- [x] Playwright からの接続テスト（CDP モード）
- [x] 非対応メソッド呼び出し時のエラーレスポンス確認

### Phase 6 Status: COMPLETE ✅

---

## Phase 7: Network Interception + Event System

### Phase 7.1: CDP Fetch ドメイン基盤

- [x] `Fetch.enable` — リクエストインターセプトの有効化（パターン指定）
- [x] `Fetch.requestPaused` イベント — マッチしたリクエストを一時停止
- [x] `Fetch.continueRequest` — リクエストをそのまま続行
- [x] `Fetch.failRequest` — リクエストを任意のエラーで失敗させる
- [x] テスト: 特定 URL パターンのリクエストが pause → continue で正常完了

### Phase 7.2: リクエスト/レスポンス書き換え

- [x] `Fetch.continueRequest` でヘッダー・URL の書き換え
- [x] `Fetch.fulfillRequest` — モックレスポンスの返却（status, headers, body）
- [x] `Fetch.getResponseBody` — pause 中のレスポンスボディ取得
- [x] `Fetch.continueResponse` — レスポンスヘッダーの書き換え
- [x] テスト: ヘッダー注入、レスポンスの差し替え、リダイレクトの書き換え

### Phase 7.3: Go API でのインターセプト

- [x] `page.OnRequest(handler)` — Go コールバックでリクエストをフック
- [x] `page.OnResponse(handler)` — Go コールバックでレスポンスをフック
- [x] `Request.Continue()`, `Request.Abort()`, `Request.Fulfill()` メソッド
- [x] CDP 経由と Go API 経由の両方で同じインターセプト基盤を共有
- [x] テスト: Go API での広告ブロック・認証ヘッダー注入のシナリオ

### Phase 7 Status: COMPLETE ✅

## Phase 8: Multi-Page + Browser Context

### Phase 8.1: Browser / BrowserContext 構造

- [x] `internal/browser/browser.go` — Browser 構造体（プロセス全体の管理）
- [x] `internal/browser/context.go` — BrowserContext 構造体（セッション分離）
- [x] BrowserContext ごとの Cookie Jar 分離
- [x] デフォルト BrowserContext の自動生成
- [x] `Browser.NewContext()`, `Context.Close()` のライフサイクル
- [x] テスト: 異なる Context 間で Cookie が共有されないことを検証

### Phase 8.2: 複数タブ（Page）管理

- [x] `Context.NewPage()` — 新規 Page の生成
- [x] `Context.Pages()` — アクティブな Page 一覧
- [x] `Page.Close()` — Page のクリーンアップ（VM、DOM、ネットワーク）
- [x] Page 間のリソース独立性の保証
- [x] CDP Target の動的追加・削除（`Target.targetCreated` / `Target.targetDestroyed`）
- [x] テスト: 複数 Page の並行ナビゲーション

### Phase 8.3: 並行処理とリソース管理

- [x] Page ごとの goroutine 管理（context.Context によるキャンセル）
- [x] Browser 全体の graceful shutdown（全 Page → 全 Context → Browser）
- [x] メモリリーク防止: Page/Context クローズ時のリソース解放検証
- [x] 同時ページ数の上限設定（`BrowserOptions.MaxPages`）
- [x] テスト: 大量 Page 生成 → 一斉クローズの race condition テスト

### Phase 8.4: CDP Target ドメイン統合

- [x] `Target.createTarget` — 新規タブの作成
- [x] `Target.closeTarget` — タブの削除
- [x] `Target.attachToTarget` — 特定ターゲットへの CDP セッション確立
- [x] `Target.detachFromTarget` — セッション切断
- [x] セッション多重化（1 WebSocket 接続で複数ターゲット操作）
- [x] テスト: Puppeteer からの複数タブ操作シナリオ

---

## Phase 9: WPT Integration + Benchmark

### Phase 9.1: WPT テストランナー

- [x] WPT リポジトリのサブモジュール or ダウンロード戦略の決定
- [x] `internal/wpt/runner.go` — テストハーネス（testharness.js の実行基盤）
- [x] WPT テストの実行 → 結果の JSON 出力
- [x] `make wpt` コマンドで指定ディレクトリのテスト実行
- [x] テストスキップリストの管理（既知の未実装機能）

### Phase 9.2: パス率トラッキング

- [x] テスト結果の集計（pass / fail / skip / total）
- [x] ドメイン別パス率の出力（dom/, html/, css/ 等）
- [x] 前回結果との差分表示（regression 検出）
- [x] 結果の JSON/CSV エクスポート

### Phase 9.3: ベンチマーク基盤

- [x] ベンチマークスイートの定義（ページロード、DOM操作、JS実行、セレクタ）
- [x] 実サイトの HTML スナップショットを使ったベンチマーク
- [x] メモリ使用量の計測（`runtime.MemStats`）
- [x] `make bench-report` で結果をフォーマット出力

### Phase 9.4: 比較ベンチマーク + ダッシュボード

- [x] Headless Chrome との比較スクリプト（同一ページの処理時間）
- [x] Lightpanda との比較（利用可能な場合）
- [x] ベンチマーク結果の可視化（Markdown テーブル or HTML レポート）
- [x] CI でのベンチマーク自動実行 + 結果アーカイブ

### Phase 9 Status: COMPLETE ✅

---

# Uzura — Backlog

フェーズ完了時、次フェーズの内容を tasks.md にコピーしてループを再開する。

## Phase 10: MCPサーバー内蔵

UzuraをClaude Code / Claude DesktopのMCPツールとして使えるようにする。
`uzura mcp` でstdioモードのMCPサーバーが起動し、シングルバイナリ内で完結する。

### Task 10.1: MCP プロトコル基盤

- [x] MCP JSON-RPC メッセージ型の定義（Request, Response, Notification）
- [x] `internal/mcp/protocol.go`: JSON-RPC 2.0 のパース・シリアライズ
- [x] `initialize` / `initialized` ハンドシェイク実装
- [x] `ping` / `pong` 実装
- [x] テスト: メッセージのラウンドトリップ、不正JSONのエラー処理

### Task 10.2: stdio トランスポート

- [x] `internal/mcp/transport.go`: stdin/stdout での行区切りJSON-RPC読み書き
- [x] バッファリングと改行処理（MCP仕様: Content-Length or newline-delimited）
- [x] stderr へのログ出力（stdoutはMCPプロトコル専用）
- [x] テスト: パイプ経由での双方向通信

### Task 10.3: ツール定義 — `browse`

- [x] `internal/mcp/tools.go`: MCPツール登録の仕組み（名前、説明、inputSchema）
- [x] `tools/list` レスポンスの実装
- [x] `browse` ツール定義:
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
- [x] テスト: ツール一覧の正しいスキーマ出力

### Task 10.4: `browse` ツール実行

- [x] `tools/call` ハンドラの実装
- [x] `browse` 呼び出し時の処理: URL取得 → DOM構築 → JS実行 → フォーマット出力
- [x] Phase 2 の Fetcher + Phase 1 の Parser を結合して呼ぶ
- [x] エラーハンドリング: ネットワークエラー、タイムアウト、不正URL
- [x] テスト: httptest.Server を使ったブラウズ→結果返却

### Task 10.5: ツール定義 — `evaluate`

- [x] `evaluate` ツール定義:
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
- [x] `tools/call` に `evaluate` ハンドラ追加
- [x] Goja VM で式を評価し、結果を文字列化して返却
- [x] テスト: DOM操作スクリプトの実行、エラースクリプトの処理

### Task 10.6: ツール定義 — `query`

- [x] `query` ツール定義:
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
- [x] マッチした要素のリストを返す（テキスト、属性値、outerHTML）
- [x] テスト: 複数要素のマッチ、属性取得、マッチなしの場合

### Task 10.7: ツール定義 — `interact`

- [x] `interact` ツール定義:
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
- [x] `click`: 要素のclickイベント発火
- [x] `fill`: input/textarea/selectの値設定 + input/changeイベント発火
- [x] テスト: フォーム入力→JS読み取り、クリック→イベントハンドラ実行

### Task 10.8: CLI `mcp` サブコマンド

- [x] `uzura mcp` でstdioモードMCPサーバーを起動
- [x] `--log-level` フラグ（stderrへ出力）
- [x] Ctrl+C / EOF でのグレースフルシャットダウン
- [x] テスト: プロセス起動→initialize→tools/list→終了の一連の流れ

### Task 10.9: Claude Code 統合テスト

- [x] `.claude.json` 用の設定例を README に記載
- [x] 手動テスト: Claude Code から `browse` ツールを呼び出し
- [x] 手動テスト: Claude Code から `query` + `evaluate` の組み合わせ
- [x] ページセッション管理: 同一URLへの連続呼び出しでDOM再利用（キャッシュ）

### Task 10.10: Phase 10 Verification

- [x] `go test ./... -race` 全パス
- [x] MCPプロトコルの仕様準拠確認（JSON-RPCエラーコード等）
- [x] `uzura mcp` の起動時間が100ms以内（実測: user+sys ~18ms）
- [x] README に MCP セットアップ手順を記載

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

- [x] `internal/semantic/tree.go`: SemanticNode 構造体
  ```go
  type SemanticNode struct {
      Role     string          // "navigation", "main", "form", "link", "button", "input", "heading", "text", "list", "image"
      Name     string          // 要素の識別名（テキスト、label、aria-label等）
      NodeID   int             // DOM上の要素ID（interact時の参照用）
      Value    string          // input の現在値、link の href 等
      Children []*SemanticNode
  }
  ```
- [x] テスト: 構造体の生成とJSON化

### Task 12.2: DOM → Semantic Tree 変換（ランドマーク）

- [x] `<header>` → role: "banner"
- [x] `<nav>` → role: "navigation"
- [x] `<main>` → role: "main"
- [x] `<aside>` → role: "complementary"
- [x] `<footer>` → role: "contentinfo"
- [x] `<article>` → role: "article"
- [x] `<section>` → role: "region"
- [x] ARIA `role` 属性の優先適用
- [x] テスト: ランドマーク要素のあるページ、ないページ

### Task 12.3: DOM → Semantic Tree 変換（インタラクティブ要素）

- [x] `<a href>` → role: "link", value: href, name: テキスト
- [x] `<button>` → role: "button", name: テキスト
- [x] `<input type="text">` → role: "textbox", name: label/placeholder
- [x] `<input type="checkbox">` → role: "checkbox", value: checked状態
- [x] `<input type="radio">` → role: "radio", value: checked状態
- [x] `<select>` → role: "combobox", value: 選択中のoption
- [x] `<textarea>` → role: "textbox", name: label/placeholder
- [x] `<input type="submit">` / `<button type="submit">` → role: "button"
- [x] label要素との紐付け（for属性、ラッピング）
- [x] テスト: 各input型、label紐付け、ネストしたフォーム

### Task 12.4: DOM → Semantic Tree 変換（コンテンツ要素）

- [x] `<h1>`-`<h6>` → role: "heading", name: テキスト
- [x] 連続テキストノード → role: "text", name: 結合テキスト（100文字で切る）
- [x] `<ul>/<ol>` → role: "list", children に各 `<li>`
- [x] `<img>` → role: "image", name: alt属性
- [x] `<table>` → role: "table"（行数・列数をnameに含める）
- [x] テスト: 各コンテンツ要素の変換

### Task 12.5: ツリーの圧縮・ノイズ除去

- [x] テキストのみの中間ノード（`<div>`, `<span>`）をスキップして子を昇格
- [x] 空テキストノードの除去
- [x] hidden要素、aria-hidden="true" の除去
- [x] 同一roleの連続ノードの折りたたみ（テキストブロックの結合）
- [x] 最大深さ制限（デフォルト10）超えたら子を省略
- [x] テスト: 圧縮前後のツリーサイズ比較

### Task 12.6: MCPツール `semantic_tree`

- [x] `semantic_tree` ツール定義:
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
- [x] 出力フォーマット: インデント付きテキスト
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
- [x] テスト: MCP経由でのsemantic_tree出力

### Task 12.7: `interact` ツールとの連携

- [x] semantic_tree の NodeID を `interact` ツールの selector として使用可能にする
- [x] NodeID → DOM要素 のマッピングテーブル管理
- [x] ワークフロー: semantic_tree で構造把握 → interact でNodeID指定して操作
- [x] テスト: semantic_tree取得 → フォーム入力 → 結果確認

### Task 12.8: CLIへの `--format semantic` 追加

- [x] `uzura fetch <url> --format semantic` で動作
- [x] インタラクティブ要素の数をサマリ表示
- [x] `--semantic-depth N` オプション

### Task 12.9: Phase 12 Verification

- [x] 複雑なフォームページ（ログイン、検索、入力フォーム）でのsemantic_tree品質
- [x] SPAサイト（JS実行後）のsemantic_tree出力
- [x] semantic_tree → interact のE2Eワークフローテスト
- [x] 出力トークン数: 一般的なページで500-2000トークン以内
- [x] Claude Code からの実際のワークフローテスト:
      「このサイトにログインして」→ semantic_tree → interact の流れ


## Phase 13: text出力ノイズ除去 & User-Agent改善

browse format=text の出力から不要な script/style コンテンツを除去し、
User-Agent を改善してボット検知を軽減する。
小さな変更で全サイトの出力品質を大幅に向上させるクイックウィン。

背景: 30サイト互換性テスト（2026-04-05）で、text出力に大量のJS/CSSが
混入する問題が判明。NHK(852KB), Amazon(752KB), Vercel(527KB) 等で
出力の大半がノイズ。また Stack Overflow, Reddit, Medium が
Cloudflare/ボット検知でブロックされた。

### Task 13.1: text出力の script/style 除去

- [x] `internal/dom/` のテキスト抽出で `<script>`, `<style>` ノード内テキストをスキップ
- [x] `<noscript>` ノードの扱いを決定（有用なコンテンツを含む場合がある）
- [x] markdown変換（`internal/markdown/`）で既に実装済みの除去ロジックを参考に統一
- [x] テスト: script/style混在HTMLでのtext出力がクリーンになることを確認
- [x] ベンチマーク: 除去前後の出力サイズ比較

### Task 13.2: hidden要素・メタデータの除去

- [x] `hidden` 属性を持つ要素のテキスト除去
- [x] `aria-hidden="true"` 要素のテキスト除去
- [x] `display:none` インラインスタイル要素の除去
- [x] `<template>` 要素の除去
- [x] テスト: 各hidden パターンでの除去確認

### Task 13.3: User-Agent の改善

- [x] 現在のデフォルト User-Agent を確認
- [x] 一般的な Chrome User-Agent 文字列に変更
  （例: `Mozilla/5.0 ... Chrome/130.0.0.0 Safari/537.36`）
- [x] `Accept`, `Accept-Language`, `Accept-Encoding` ヘッダーの追加
- [x] Sec-Fetch-* ヘッダー群の追加（Sec-Fetch-Mode, Sec-Fetch-Site, Sec-Fetch-Dest）
- [x] テスト: httptest.Server でヘッダーの送信を確認

### Task 13.4: TLS フィンガープリントの改善

- [x] `crypto/tls` の ClientHello 設定を Chrome 相当に調整
- [x] TLS 拡張の順序と内容を一般的なブラウザに合わせる
- [x] HTTP/2 対応の確認（ALPN ネゴシエーション）
- [x] テスト: TLS 接続が一般ブラウザと同等のフィンガープリントになることを確認

### Task 13.5: Phase 13 Verification
30サイトテスト内容は uzura-site-test-prompt.md を参照

- [ ] 30サイト互換性テストの再実行（MCPサーバー再起動後に実施）
- [x] text出力サイズの削減率を測定（99.9%削減確認、目標50%超を大幅達成）
- [ ] User-Agent 改善後のボット検知回避率を確認（MCPサーバー再起動後に実施）
- [ ] Stack Overflow, Reddit, Medium への接続改善を確認（MCPサーバー再起動後に実施）
- [x] 既存テストの全パス（`go test ./... -race`）

---

## Phase 14: Markdown品質の安定化

readability 失敗時のフォールバック戦略を改善し、
SPA サイトや構造が特殊なサイトでの markdown 出力品質を向上させる。

背景: SPA系サイト（Anthropic Docs, Claude API Docs）で markdown が
"Loading..." のみ、Angular の markdown が極少量等の問題が判明。

### Task 14.1: readability フォールバックの改善

- [x] readability 失敗を検知する基準の明確化（出力が短すぎる、本文なし等）
- [x] フォールバック戦略: `<main>` → `<article>` → `<body>` の順で本文領域を探索
- [x] semantic_tree ベースの markdown 生成（ランドマーク構造を活用）
- [x] テスト: readability 成功/失敗/部分成功の各パターン

### Task 14.2: noscript コンテンツの活用

- [x] `<noscript>` 内のHTMLをパースしてコンテンツ候補にする
- [x] JS無効時にnoscriptコンテンツがメインコンテンツになるサイトへの対応
- [x] noscript と通常コンテンツの優先度判定
- [x] テスト: Yahoo! Japan 等 noscript に有用なコンテンツがあるサイト

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
