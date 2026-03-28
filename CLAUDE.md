# Uzura — Claude Code Memory

## Project Overview
Uzura is a minimal headless browser in Go, optimized for AI agents.
- Repo: github.com/nyasuto/uzura
- Binary: `uzura`
- Language: Go (pure, no cgo)

## Build & Test
```bash
go build ./cmd/uzura          # build binary
go test ./... -race            # run all tests with race detector
go test ./... -bench=.         # run benchmarks
golangci-lint run              # lint (includes vet)
```
## 外部AI相談ルール

以下の状況では、gemini MCPツールを使って相談・検索すること：

- デバッグで2回以上同じエラーの修正に失敗したとき
- 最新のライブラリ仕様やAPIの変更点を確認したいとき
- アーキテクチャの設計判断で迷ったとき（セカンドオピニオン）
- 自分の知識に自信がないニッチな技術トピックのとき

使い分け：
- gemini-search: ウェブ検索が必要な最新情報の調査
- gemini-query: 設計相談やセカンドオピニオン
- gemini-brainstorm: 複数の選択肢を検討したいとき

注意: 単純なコーディングや既知の問題には使わない（トークンの無駄）
## Architecture
- `cmd/uzura/` — CLI entry point only. No logic here.
- `internal/dom/` — DOM tree types (Node, Element, Text, Document)
- `internal/html/` — HTML parser adapter (wraps golang.org/x/net/html)
- `internal/css/` — CSS selector adapter (wraps cascadia) — Phase 3+
- `internal/js/` — JavaScript engine (wraps goja) — Phase 5+
- `internal/cdp/` — CDP WebSocket server — Phase 6+
- `internal/network/` — HTTP fetcher — Phase 2+
- `internal/browser/` — Browser/Page/Context — Phase 8+

## Conventions
- One type or concern per file, max 300 lines
- All exported names have godoc comments
- Tests use table-driven style
- Commit format: `phase{N}.{T}: {description}`
- Errors returned, never panic

## Current Focus
See tasks.md for the active phase and pending tasks.