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
- [ ] Readmeを作成
- [ ] ステータスバッジを README に追加（README作成時に対応）

### Task CI.2: カバレッジレポート

- [x] `make cover` でカバレッジ生成（既存）
- [x] Codecov に連携（codecov-action@v5）
- [x] PR にカバレッジ差分コメントを自動投稿（Codecov側のPR統合で対応）
- [ ] カバレッジバッジを README に追加（README作成時に対応）

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
- [ ] `Fetch.enable` — リクエストインターセプトの有効化（パターン指定）
- [ ] `Fetch.requestPaused` イベント — マッチしたリクエストを一時停止
- [ ] `Fetch.continueRequest` — リクエストをそのまま続行
- [ ] `Fetch.failRequest` — リクエストを任意のエラーで失敗させる
- [ ] テスト: 特定 URL パターンのリクエストが pause → continue で正常完了

### Phase 7.2: リクエスト/レスポンス書き換え
- [ ] `Fetch.continueRequest` でヘッダー・URL の書き換え
- [ ] `Fetch.fulfillRequest` — モックレスポンスの返却（status, headers, body）
- [ ] `Fetch.getResponseBody` — pause 中のレスポンスボディ取得
- [ ] `Fetch.continueResponse` — レスポンスヘッダーの書き換え
- [ ] テスト: ヘッダー注入、レスポンスの差し替え、リダイレクトの書き換え

### Phase 7.3: Go API でのインターセプト
- [ ] `page.OnRequest(handler)` — Go コールバックでリクエストをフック
- [ ] `page.OnResponse(handler)` — Go コールバックでレスポンスをフック
- [ ] `Request.Continue()`, `Request.Abort()`, `Request.Fulfill()` メソッド
- [ ] CDP 経由と Go API 経由の両方で同じインターセプト基盤を共有
- [ ] テスト: Go API での広告ブロック・認証ヘッダー注入のシナリオ


## Phase 8: Multi-Page + Browser Context

### Phase 8.1: Browser / BrowserContext 構造
- [ ] `internal/browser/browser.go` — Browser 構造体（プロセス全体の管理）
- [ ] `internal/browser/context.go` — BrowserContext 構造体（セッション分離）
- [ ] BrowserContext ごとの Cookie Jar 分離
- [ ] デフォルト BrowserContext の自動生成
- [ ] `Browser.NewContext()`, `Context.Close()` のライフサイクル
- [ ] テスト: 異なる Context 間で Cookie が共有されないことを検証

### Phase 8.2: 複数タブ（Page）管理
- [ ] `Context.NewPage()` — 新規 Page の生成
- [ ] `Context.Pages()` — アクティブな Page 一覧
- [ ] `Page.Close()` — Page のクリーンアップ（VM、DOM、ネットワーク）
- [ ] Page 間のリソース独立性の保証
- [ ] CDP Target の動的追加・削除（`Target.targetCreated` / `Target.targetDestroyed`）
- [ ] テスト: 複数 Page の並行ナビゲーション

### Phase 8.3: 並行処理とリソース管理
- [ ] Page ごとの goroutine 管理（context.Context によるキャンセル）
- [ ] Browser 全体の graceful shutdown（全 Page → 全 Context → Browser）
- [ ] メモリリーク防止: Page/Context クローズ時のリソース解放検証
- [ ] 同時ページ数の上限設定（`BrowserOptions.MaxPages`）
- [ ] テスト: 大量 Page 生成 → 一斉クローズの race condition テスト

### Phase 8.4: CDP Target ドメイン統合
- [ ] `Target.createTarget` — 新規タブの作成
- [ ] `Target.closeTarget` — タブの削除
- [ ] `Target.attachToTarget` — 特定ターゲットへの CDP セッション確立
- [ ] `Target.detachFromTarget` — セッション切断
- [ ] セッション多重化（1 WebSocket 接続で複数ターゲット操作）
- [ ] テスト: Puppeteer からの複数タブ操作シナリオ

---

## Phase 9: WPT Integration + Benchmark

### Phase 9.1: WPT テストランナー
- [ ] WPT リポジトリのサブモジュール or ダウンロード戦略の決定
- [ ] `internal/wpt/runner.go` — テストハーネス（testharness.js の実行基盤）
- [ ] WPT テストの実行 → 結果の JSON 出力
- [ ] `make wpt` コマンドで指定ディレクトリのテスト実行
- [ ] テストスキップリストの管理（既知の未実装機能）

### Phase 9.2: パス率トラッキング
- [ ] テスト結果の集計（pass / fail / skip / total）
- [ ] ドメイン別パス率の出力（dom/, html/, css/ 等）
- [ ] 前回結果との差分表示（regression 検出）
- [ ] 結果の JSON/CSV エクスポート

### Phase 9.3: ベンチマーク基盤
- [ ] ベンチマークスイートの定義（ページロード、DOM操作、JS実行、セレクタ）
- [ ] 実サイトの HTML スナップショットを使ったベンチマーク
- [ ] メモリ使用量の計測（`runtime.MemStats`）
- [ ] `make bench-report` で結果をフォーマット出力

### Phase 9.4: 比較ベンチマーク + ダッシュボード
- [ ] Headless Chrome との比較スクリプト（同一ページの処理時間）
- [ ] Lightpanda との比較（利用可能な場合）
- [ ] ベンチマーク結果の可視化（Markdown テーブル or HTML レポート）
- [ ] CI でのベンチマーク自動実行 + 結果アーカイブ

---
