#!/usr/bin/env bash
#
# ralph.sh — Ralph Wiggum Loop for Uzura
#
# Usage:
#   ./ralph.sh              # 1イテレーションずつ確認
#   ./ralph.sh --auto       # 自動継続（Ctrl+Cで停止）
#   ./ralph.sh --auto -n 10 # 最大10イテレーション
#
# フェーズ移行:
#   1. ループが全タスク完了で停止する
#   2. 成果物を確認する
#   3. backlog.md から次フェーズの内容を tasks.md にコピーする
#   4. ./ralph.sh で再開
#

set -euo pipefail

PROMPT_FILE="PROMPT.md"
TASKS_FILE="tasks.md"
MAX_ITERATIONS=0
AUTO_MODE=false
SLEEP_BETWEEN=3

while [[ $# -gt 0 ]]; do
  case $1 in
    --auto)   AUTO_MODE=true; shift ;;
    -n)       MAX_ITERATIONS="$2"; shift 2 ;;
    --help)
      echo "Usage: $0 [--auto] [-n max_iterations]"
      exit 0
      ;;
    *)        echo "Unknown option: $1"; exit 1 ;;
  esac
done

if ! command -v claude &>/dev/null; then
  echo "❌ claude コマンドが見つかりません"; exit 1
fi
for f in "$PROMPT_FILE" "$TASKS_FILE"; do
  [[ -f "$f" ]] || { echo "❌ $f が見つかりません"; exit 1; }
done

count_remaining() { grep -c '^- \[ \]' "$TASKS_FILE" 2>/dev/null || echo 0; }
count_completed() { grep -c '^- \[x\]' "$TASKS_FILE" 2>/dev/null || echo 0; }

iteration=0

echo "🐦 Uzura Ralph Loop"
echo "📋 残り: $(count_remaining) / 完了: $(count_completed)"
echo ""

while true; do
  iteration=$((iteration + 1))
  remaining=$(count_remaining)

  if [[ "$remaining" -eq 0 ]]; then
    echo "✅ 全タスク完了！ ($(count_completed) tasks)"
    echo "👉 backlog.md から次フェーズを tasks.md にコピーして再開"
    break
  fi

  if [[ "$MAX_ITERATIONS" -gt 0 && "$iteration" -gt "$MAX_ITERATIONS" ]]; then
    echo "⏸️  最大 $MAX_ITERATIONS 回に到達。停止。"
    break
  fi

  echo "─── #$iteration $(date '+%H:%M:%S') ─── 残り: $remaining ───"

  claude --print --dangerously-skip-permissions "$(cat "$PROMPT_FILE")"

  echo ""
  echo "残り: $(count_remaining) / 完了: $(count_completed)"

  if $AUTO_MODE; then
    sleep "$SLEEP_BETWEEN"
  else
    read -rp "ENTER=次, q=終了: " input
    [[ "$input" == "q" ]] && break
  fi
done