#!/bin/bash
# collect_results.sh - Collect and aggregate results from all test runs

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
RESULTS_DIR="$TEST_DIR/results"

if [ ! -d "$RESULTS_DIR/disabled" ] || [ ! -d "$RESULTS_DIR/enabled" ]; then
    echo "Error: Results directories not found"
    echo "Please run tests first: ./scripts/run_all_tests.sh disabled && ./scripts/run_all_tests.sh enabled"
    exit 1
fi

echo "Collecting results..."

# Function to parse bytes from summary file
parse_bytes() {
    local file=$1
    if [ -f "$file" ]; then
        source "$file"
        echo "$TOTAL_BYTES"
    else
        echo "0"
    fi
}

# Collect results for each SDK
declare -A baseline_results
declare -A compressed_results

for SDK in go typescript python; do
    baseline_file="$RESULTS_DIR/disabled/${SDK}_network.log.summary"
    compressed_file="$RESULTS_DIR/enabled/${SDK}_network.log.summary"

    baseline_results[$SDK]=$(parse_bytes "$baseline_file")
    compressed_results[$SDK]=$(parse_bytes "$compressed_file")
done

# Calculate totals
baseline_total=0
compressed_total=0

for SDK in go typescript python; do
    baseline_total=$(echo "$baseline_total + ${baseline_results[$SDK]}" | bc)
    compressed_total=$(echo "$compressed_total + ${compressed_results[$SDK]}" | bc)
done

# Save aggregated results
{
    echo "BASELINE_TOTAL=$baseline_total"
    echo "COMPRESSED_TOTAL=$compressed_total"
    echo "GO_BASELINE=${baseline_results[go]}"
    echo "GO_COMPRESSED=${compressed_results[go]}"
    echo "TYPESCRIPT_BASELINE=${baseline_results[typescript]}"
    echo "TYPESCRIPT_COMPRESSED=${compressed_results[typescript]}"
    echo "PYTHON_BASELINE=${baseline_results[python]}"
    echo "PYTHON_COMPRESSED=${compressed_results[python]}"
} > "$RESULTS_DIR/aggregated_results.txt"

echo "Results collected and saved to: $RESULTS_DIR/aggregated_results.txt"
