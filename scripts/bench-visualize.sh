#!/bin/bash
# bench-visualize.sh — Generate an HTML benchmark report from Go benchmark output.
set -euo pipefail

INPUT="${1:-bench-results.txt}"
OUTPUT="${2:-bench-report.html}"

if [ ! -f "$INPUT" ]; then
  echo "Usage: $0 <bench-results.txt> [output.html]"
  echo "Run 'make bench-report' first to generate bench-results.txt"
  exit 1
fi

cat > "$OUTPUT" <<'HEADER'
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Uzura Benchmark Report</title>
<style>
  body { font-family: -apple-system, sans-serif; max-width: 900px; margin: 2rem auto; padding: 0 1rem; color: #333; }
  h1 { border-bottom: 2px solid #2563eb; padding-bottom: 0.5rem; }
  h2 { color: #2563eb; margin-top: 2rem; }
  table { width: 100%; border-collapse: collapse; margin: 1rem 0; }
  th { background: #f1f5f9; text-align: left; padding: 0.5rem; border: 1px solid #e2e8f0; }
  td { padding: 0.5rem; border: 1px solid #e2e8f0; }
  tr:nth-child(even) { background: #f8fafc; }
  .bar { height: 20px; background: #3b82f6; border-radius: 3px; min-width: 2px; }
  .bar-cell { width: 200px; }
  .num { text-align: right; font-variant-numeric: tabular-nums; }
  .category { background: #dbeafe; font-weight: 600; }
  footer { margin-top: 2rem; color: #94a3b8; font-size: 0.875rem; }
</style>
</head>
<body>
<h1>Uzura Benchmark Report</h1>
HEADER

# Parse benchmarks and generate table rows.
awk '
# First pass: find max ns/op for scaling.
NR == FNR && /^Benchmark/ {
    ns = $3 + 0
    if (ns > max_ns) max_ns = ns
    next
}
# Second pass: emit HTML rows.
/^Benchmark/ {
    name = $1
    sub(/^Benchmark/, "", name)
    sub(/-[0-9]+$/, "", name)

    split(name, parts, "_")
    category = parts[1]
    specific = ""
    for (i = 2; i <= length(parts); i++) {
        if (specific != "") specific = specific " / "
        specific = specific parts[i]
    }
    if (specific == "") specific = category

    nsop = $3 + 0
    bop = 0; aop = 0
    for (i = 1; i <= NF; i++) {
        if ($(i+1) == "B/op") bop = $i + 0
        if ($(i+1) == "allocs/op") aop = $i + 0
    }

    if (category != prev_cat) {
        printf "<tr class=\"category\"><td colspan=\"5\">%s</td></tr>\n", category
        prev_cat = category
    }

    if (nsop >= 1000000) time = sprintf("%.2f ms", nsop / 1000000)
    else if (nsop >= 1000) time = sprintf("%.1f μs", nsop / 1000)
    else time = sprintf("%.0f ns", nsop)

    if (bop >= 1048576) memstr = sprintf("%.1f MB", bop / 1048576)
    else if (bop >= 1024) memstr = sprintf("%.1f KB", bop / 1024)
    else memstr = sprintf("%d B", bop)

    bar_pct = 1
    if (nsop > 0 && max_ns > 0) {
        bar_pct = (log(nsop) / log(max_ns)) * 100
        if (bar_pct < 1) bar_pct = 1
    }

    printf "<tr><td>%s</td><td class=\"num\">%s</td>", specific, time
    printf "<td class=\"bar-cell\"><div class=\"bar\" style=\"width:%d%%\"></div></td>", bar_pct
    printf "<td class=\"num\">%s</td><td class=\"num\">%d</td></tr>\n", memstr, aop
}
BEGIN {
    print "<table>"
    print "<tr><th>Benchmark</th><th class=\"num\">Time</th><th class=\"bar-cell\">Relative</th><th class=\"num\">Memory</th><th class=\"num\">Allocs</th></tr>"
}
END {
    print "</table>"
}
' "$INPUT" "$INPUT" >> "$OUTPUT"

cat >> "$OUTPUT" <<FOOTER
<footer>
  Generated on $(date -u +"%Y-%m-%d %H:%M UTC") by Uzura bench-visualize.sh
</footer>
</body>
</html>
FOOTER

echo "HTML report: $OUTPUT"
