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
go vet ./...                   # static analysis
```

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