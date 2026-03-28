#!/usr/bin/env bash
# wpt-fetch.sh — Download Web Platform Tests (sparse checkout)
#
# Usage: scripts/wpt-fetch.sh [target_dir]
#   target_dir defaults to testdata/wpt

set -euo pipefail

WPT_REPO="https://github.com/web-platform-tests/wpt.git"
TARGET_DIR="${1:-testdata/wpt}"

# Directories to fetch (only what Uzura can test)
SPARSE_DIRS=(
    "resources"
    "dom/nodes"
    "dom/traversal"
    "dom/collections"
    "html/dom"
    "html/semantics"
)

main() {
    if [ -d "$TARGET_DIR/.git" ]; then
        echo "WPT already cloned at $TARGET_DIR, updating..."
        cd "$TARGET_DIR"
        git pull --depth=1
        exit 0
    fi

    echo "Cloning WPT (sparse checkout)..."
    git clone --depth=1 --filter=blob:none --sparse \
        "$WPT_REPO" "$TARGET_DIR"

    cd "$TARGET_DIR"

    echo "Setting up sparse checkout..."
    git sparse-checkout init --cone
    git sparse-checkout set "${SPARSE_DIRS[@]}"

    echo "WPT downloaded to $TARGET_DIR"
    echo "Directories:"
    for d in "${SPARSE_DIRS[@]}"; do
        if [ -d "$d" ]; then
            count=$(find "$d" -name '*.html' | wc -l | tr -d ' ')
            echo "  $d/ ($count HTML files)"
        else
            echo "  $d/ (not found)"
        fi
    done
}

main
