#!/bin/bash
# Recalculate network summary from an existing log file

LOG_FILE=$1
if [ -z "$LOG_FILE" ]; then
    echo "Usage: $0 <log_file>"
    exit 1
fi

# Helper function to strip ANSI escape codes
strip_ansi() {
    echo "$1" | sed 's/\x1b\[[0-9;]*[a-zA-Z]//g' | sed 's/\x1b\[[0-9;]*m//g' | tr -d '\r'
}

# Helper function to parse stats and convert to bytes
parse_stats_to_bytes() {
    local stats_line="$1"
    local rx_bytes=0
    local tx_bytes=0

    stats_line=$(echo "$stats_line" | sed 's/\x1b\[[0-9;]*[a-zA-Z]//g' | sed 's/\x1b\[[0-9;]*m//g' | tr -d '\r' | xargs)

    if [[ $stats_line =~ ([0-9.]+)([kmgtKMG]?B)\ */\ *([0-9.]+)([kmgtKMG]?B) ]]; then
        RX_VAL=${BASH_REMATCH[1]}
        RX_UNIT=${BASH_REMATCH[2]}
        TX_VAL=${BASH_REMATCH[3]}
        TX_UNIT=${BASH_REMATCH[4]}

        RX_UNIT=$(echo "$RX_UNIT" | tr '[:lower:]' '[:upper:]')
        TX_UNIT=$(echo "$TX_UNIT" | tr '[:lower:]' '[:upper:]')

        rx_bytes=$(awk -v val="$RX_VAL" -v unit="$RX_UNIT" 'BEGIN {
            if (unit == "B") print int(val)
            else if (unit == "KB") print int(val * 1024)
            else if (unit == "MB") print int(val * 1024 * 1024)
            else if (unit == "GB") print int(val * 1024 * 1024 * 1024)
            else if (unit == "TB") print int(val * 1024 * 1024 * 1024 * 1024)
            else print int(val)
        }')

        tx_bytes=$(awk -v val="$TX_VAL" -v unit="$TX_UNIT" 'BEGIN {
            if (unit == "B") print int(val)
            else if (unit == "KB") print int(val * 1024)
            else if (unit == "MB") print int(val * 1024 * 1024)
            else if (unit == "GB") print int(val * 1024 * 1024 * 1024)
            else if (unit == "TB") print int(val * 1024 * 1024 * 1024 * 1024)
            else print int(val)
        }')
    fi

    echo "$rx_bytes $tx_bytes"
}
# Extract initial and final stats from log file
# First, strip ANSI codes and get clean lines
CLEAN_LOG=$(mktemp)
sed 's/\x1b\[[0-9;]*[a-zA-Z]//g' "$LOG_FILE" | sed 's/\x1b\[[0-9;]*m//g' | tr -d '\r' > "$CLEAN_LOG"

# Try to find Initial line
INITIAL_LINE=$(grep "^Initial:" "$CLEAN_LOG" | head -1 | sed 's/^Initial: *//' | xargs)

# Try to find Final line
FINAL_LINE=$(grep "^Final:" "$CLEAN_LOG" | tail -1 | sed 's/^Final: *//' | xargs)

# If no Final line, try to get the last valid stats line
if [ -z "$FINAL_LINE" ]; then
    FINAL_LINE=$(grep -E "^[0-9]+\.[0-9]+[kmgtKMG]?B / [0-9]+\.[0-9]+[kmgtKMG]?B$" "$CLEAN_LOG" | tail -1 | xargs)
fi

# If no Initial line, try to get the first valid stats line
if [ -z "$INITIAL_LINE" ]; then
    INITIAL_LINE=$(grep -E "^[0-9]+\.[0-9]+[kmgtKMG]?B / [0-9]+\.[0-9]+[kmgtKMG]?B$" "$CLEAN_LOG" | head -1 | xargs)
fi

rm -f "$CLEAN_LOG"

if [ -z "$INITIAL_LINE" ] || [ -z "$FINAL_LINE" ]; then
    echo "Error: Could not find Initial or Final stats in log file"
    echo "Initial: '$INITIAL_LINE'"
    echo "Final: '$FINAL_LINE'"
    exit 1
fi

INITIAL_STATS=$(strip_ansi "$INITIAL_LINE")
FINAL_STATS=$(strip_ansi "$FINAL_LINE")

INITIAL_BYTES=$(parse_stats_to_bytes "$INITIAL_STATS")
FINAL_BYTES=$(parse_stats_to_bytes "$FINAL_STATS")

INITIAL_RX=$(echo $INITIAL_BYTES | awk '{print $1}')
INITIAL_TX=$(echo $INITIAL_BYTES | awk '{print $2}')
FINAL_RX=$(echo $FINAL_BYTES | awk '{print $1}')
FINAL_TX=$(echo $FINAL_BYTES | awk '{print $2}')

TOTAL_RX=$((FINAL_RX - INITIAL_RX))
TOTAL_TX=$((FINAL_TX - INITIAL_TX))
TOTAL_BYTES=$((TOTAL_RX + TOTAL_TX))

# Helper function to format bytes in human-readable format
format_bytes() {
    local bytes=$1
    if [ $bytes -ge 1099511627776 ]; then
        awk "BEGIN {printf \"%.2f TB\", $bytes / 1099511627776}"
    elif [ $bytes -ge 1073741824 ]; then
        awk "BEGIN {printf \"%.2f GB\", $bytes / 1073741824}"
    elif [ $bytes -ge 1048576 ]; then
        awk "BEGIN {printf \"%.2f MB\", $bytes / 1048576}"
    elif [ $bytes -ge 1024 ]; then
        awk "BEGIN {printf \"%.2f KB\", $bytes / 1024}"
    else
        echo "${bytes} B"
    fi
}

RX_FORMATTED=$(format_bytes $TOTAL_RX)
TX_FORMATTED=$(format_bytes $TOTAL_TX)
TOTAL_FORMATTED=$(format_bytes $TOTAL_BYTES)

echo "Initial: $INITIAL_STATS ($INITIAL_RX bytes RX, $INITIAL_TX bytes TX)"
echo "Final: $FINAL_STATS ($FINAL_RX bytes RX, $FINAL_TX bytes TX)"
echo ""
echo "=== Network Summary ==="
echo "Total Received: $RX_FORMATTED"
echo "Total Sent: $TX_FORMATTED"
echo "Total: $TOTAL_FORMATTED"
echo ""
echo "RX_BYTES=$TOTAL_RX"
echo "TX_BYTES=$TOTAL_TX"
echo "TOTAL_BYTES=$TOTAL_BYTES"

# Save to summary file
{
    echo "RX_BYTES=$TOTAL_RX"
    echo "TX_BYTES=$TOTAL_TX"
    echo "TOTAL_BYTES=$TOTAL_BYTES"
} > "${LOG_FILE}.summary"

echo ""
echo "Summary saved to: ${LOG_FILE}.summary"
