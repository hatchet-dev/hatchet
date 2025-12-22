#!/bin/bash
# generate_report.sh - Generate comparison report

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
RESULTS_DIR="$TEST_DIR/results"

# Collect results first
"$SCRIPT_DIR/collect_results.sh"

# Load aggregated results
if [ ! -f "$RESULTS_DIR/aggregated_results.txt" ]; then
    echo "Error: Aggregated results not found. Run collect_results.sh first."
    exit 1
fi

source "$RESULTS_DIR/aggregated_results.txt"

# Helper function to format bytes
format_bytes() {
    local bytes=$1
    if [ $(echo "$bytes > 1073741824" | bc) -eq 1 ]; then
        echo "$(echo "scale=2; $bytes / 1073741824" | bc) GB"
    elif [ $(echo "$bytes > 1048576" | bc) -eq 1 ]; then
        echo "$(echo "scale=2; $bytes / 1048576" | bc) MB"
    elif [ $(echo "$bytes > 1024" | bc) -eq 1 ]; then
        echo "$(echo "scale=2; $bytes / 1024" | bc) KB"
    else
        echo "${bytes} B"
    fi
}

# Helper function to calculate percentage reduction
calc_reduction() {
    local baseline=$1
    local compressed=$2
    if [ $(echo "$baseline > 0" | bc) -eq 1 ]; then
        local diff=$(echo "$baseline - $compressed" | bc)
        local percent=$(echo "scale=1; ($diff / $baseline) * 100" | bc)
        echo "$percent"
    else
        echo "0"
    fi
}

# Calculate reductions
go_reduction=$(calc_reduction "$GO_BASELINE" "$GO_COMPRESSED")
ts_reduction=$(calc_reduction "$TYPESCRIPT_BASELINE" "$TYPESCRIPT_COMPRESSED")
python_reduction=$(calc_reduction "$PYTHON_BASELINE" "$PYTHON_COMPRESSED")
total_reduction=$(calc_reduction "$BASELINE_TOTAL" "$COMPRESSED_TOTAL")

# Generate report
REPORT_FILE="$RESULTS_DIR/compression_report.txt"

cat > "$REPORT_FILE" <<EOF
========================================
Compression Test Results
========================================

Baseline (No Compression):
  Go SDK:        $(format_bytes "$GO_BASELINE")
  TypeScript SDK: $(format_bytes "$TYPESCRIPT_BASELINE")
  Python SDK:    $(format_bytes "$PYTHON_BASELINE")
  Total:         $(format_bytes "$BASELINE_TOTAL")

With Compression:
  Go SDK:        $(format_bytes "$GO_COMPRESSED") ($go_reduction% reduction)
  TypeScript SDK: $(format_bytes "$TYPESCRIPT_COMPRESSED") ($ts_reduction% reduction)
  Python SDK:    $(format_bytes "$PYTHON_COMPRESSED") ($python_reduction% reduction)
  Total:         $(format_bytes "$COMPRESSED_TOTAL") ($total_reduction% reduction)

Bandwidth Savings:
  Total Saved:   $(format_bytes "$(echo "$BASELINE_TOTAL - $COMPRESSED_TOTAL" | bc)")
  Reduction:     $total_reduction%

========================================
Detailed Breakdown
========================================

Go SDK:
  Baseline:   $(format_bytes "$GO_BASELINE")
  Compressed: $(format_bytes "$GO_COMPRESSED")
  Savings:   $(format_bytes "$(echo "$GO_BASELINE - $GO_COMPRESSED" | bc)") ($go_reduction%)

TypeScript SDK:
  Baseline:   $(format_bytes "$TYPESCRIPT_BASELINE")
  Compressed: $(format_bytes "$TYPESCRIPT_COMPRESSED")
  Savings:   $(format_bytes "$(echo "$TYPESCRIPT_BASELINE - $TYPESCRIPT_COMPRESSED" | bc)") ($ts_reduction%)

Python SDK:
  Baseline:   $(format_bytes "$PYTHON_BASELINE")
  Compressed: $(format_bytes "$PYTHON_COMPRESSED")
  Savings:   $(format_bytes "$(echo "$PYTHON_BASELINE - $PYTHON_COMPRESSED" | bc)") ($python_reduction%)

========================================
EOF

cat "$REPORT_FILE"
echo ""
echo "Full report saved to: $REPORT_FILE"
