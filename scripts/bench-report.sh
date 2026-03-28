#!/bin/bash
# bench-report.sh — Run Uzura benchmarks and format output as Markdown.
set -euo pipefail

OUTFILE="${1:-bench-results.txt}"

echo "Running benchmarks..."
go test ./internal/bench/ -bench=. -benchmem -count=3 -benchtime=200ms > "$OUTFILE" 2>&1

echo ""
echo "=== Uzura Benchmark Report ==="
echo ""

# Parse and format results.
awk '
/^Benchmark/ {
    name = $1
    sub(/^Benchmark/, "", name)
    sub(/-[0-9]+$/, "", name)
    gsub(/_/, " / ", name)

    ops = $2
    nsop = $3
    # Find B/op and allocs/op
    bop = ""; aop = ""
    for (i = 1; i <= NF; i++) {
        if ($(i+1) == "B/op") bop = $i
        if ($(i+1) == "allocs/op") aop = $i
    }

    # Format time
    if (nsop + 0 >= 1000000) {
        time = sprintf("%.2f ms", nsop / 1000000)
    } else if (nsop + 0 >= 1000) {
        time = sprintf("%.1f μs", nsop / 1000)
    } else {
        time = sprintf("%.0f ns", nsop)
    }

    printf "| %-40s | %8s | %10s | %10s |\n", name, time, bop " B", aop " allocs"
}
BEGIN {
    printf "| %-40s | %8s | %10s | %10s |\n", "Benchmark", "Time", "Memory", "Allocs"
    printf "|%-42s|%10s|%12s|%12s|\n", ":-", "-:", "-:", "-:"
}
END {
    print ""
}
' "$OUTFILE"

echo "Raw results saved to: $OUTFILE"
