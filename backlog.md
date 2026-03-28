# Uzura — Backlog

フェーズ完了時、次フェーズの内容を tasks.md にコピーしてループを再開する。

---


## Phase 6: CDP WebSocket Server

### Phase 6.1: WebSocket 基盤 + ディスカバリー
- [ ] `nhooyr.io/websocket` 依存追加
- [ ] `internal/cdp/server.go` — WebSocket サーバー起動（`uzura serve --port 9222`）
- [ ] CDP メッセージのJSON-RPC ディスパッチャー（id, method, params → result/error）
- [ ] `/json/version` エンドポイント（ブラウザメタ情報）
- [ ] `/json/list` エンドポイント（ターゲット一覧）
- [ ] `/json/protocol` エンドポイント（サポートドメイン一覧）
- [ ] テスト: WebSocket 接続、JSON-RPC の送受信、ディスカバリーの HTTP レスポンス

### Phase 6.2: Page ドメイン
- [ ] `Page.enable` — ページイベントの有効化
- [ ] `Page.navigate` — URL ナビゲーション（internal/page と連携）
- [ ] `Page.getFrameTree` — フレームツリー取得（単一フレーム）
- [ ] `Page.loadEventFired` イベント送信
- [ ] `Page.domContentEventFired` イベント送信
- [ ] テスト: ナビゲーション → ロード完了イベントの一連の流れ

### Phase 6.3: DOM ドメイン
- [ ] `DOM.enable` — DOM イベントの有効化
- [ ] `DOM.getDocument` — ルートノードの返却（depth 制御）
- [ ] `DOM.querySelector` / `DOM.querySelectorAll`
- [ ] `DOM.getOuterHTML` / `DOM.setOuterHTML`
- [ ] `DOM.getAttributes` / `DOM.setAttributeValue` / `DOM.removeAttribute`
- [ ] `DOM.requestChildNodes` — 子ノードのストリーミング取得
- [ ] `DOM.documentUpdated` イベント（MutationObserver と連携）
- [ ] nodeId の管理（Go DOM ノード ↔ CDP nodeId のマッピング）
- [ ] テスト: DOM クエリ・変更が CDP 経由で正しく動作

### Phase 6.4: Runtime ドメイン
- [ ] `Runtime.enable` — ランタイムイベントの有効化
- [ ] `Runtime.evaluate` — JS 式の評価（Phase 5 の VM と連携）
- [ ] `Runtime.callFunctionOn` — リモートオブジェクトへの関数呼び出し
- [ ] RemoteObject のシリアライズ（type, value, objectId）
- [ ] `Runtime.consoleAPICalled` イベント（console.log 等の転送）
- [ ] `Runtime.exceptionThrown` イベント
- [ ] テスト: JS 評価、戻り値のシリアライズ、エラーハンドリング

### Phase 6.5: Network ドメイン
- [ ] `Network.enable` — ネットワークイベントの有効化
- [ ] `Network.requestWillBeSent` イベント
- [ ] `Network.responseReceived` イベント
- [ ] `Network.loadingFinished` / `Network.loadingFailed` イベント
- [ ] `Network.getResponseBody` — レスポンスボディの取得
- [ ] リクエスト ID の管理とイベントの時系列保証
- [ ] テスト: ページロード中のネットワークイベント発火の検証

### Phase 6.6: Puppeteer 接続テスト
- [ ] Puppeteer (`puppeteer-core`) からの接続確認スクリプト作成
- [ ] `puppeteer.connect({ browserWSEndpoint })` での接続
- [ ] 基本操作の E2E テスト（navigate → querySelector → getOuterHTML）
- [ ] Playwright からの接続テスト（CDP モード）
- [ ] 非対応メソッド呼び出し時のエラーレスポンス確認

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

---

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
