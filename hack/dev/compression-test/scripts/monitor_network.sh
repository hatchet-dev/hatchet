#!/bin/bash
# monitor_network.sh - Monitors network traffic for a Docker container

set -e

CONTAINER_NAME=$1
DURATION=${2:-60}
OUTPUT_FILE=${3:-/tmp/network_stats_${CONTAINER_NAME}.log}

if [ -z "$CONTAINER_NAME" ]; then
    echo "Usage: $0 <container_name> [duration_seconds] [output_file]"
    exit 1
fi

echo "Monitoring network for container: $CONTAINER_NAME"
echo "Duration: ${DURATION}s"
echo "Output: $OUTPUT_FILE"

# Clear output file
> "$OUTPUT_FILE"

# Wait a moment for container to be ready
sleep 1

# Helper function to parse stats and convert to bytes
parse_stats_to_bytes() {
    local stats_line="$1"
    local rx_bytes=0
    local tx_bytes=0

    # Strip any remaining ANSI codes and clean up
    stats_line=$(echo "$stats_line" | sed 's/\x1b\[[0-9;]*[a-zA-Z]//g' | sed 's/\x1b\[[0-9;]*m//g' | tr -d '\r' | xargs)

    # Docker stats format: "1.2MB / 3.4MB" or "1.2KB / 3.4KB" or "2.95kB / 2.35kB" etc.
    # Try to match the pattern - handle both uppercase and lowercase 'k'
    if [[ $stats_line =~ ([0-9.]+)([kmgtKMG]?B)\ */\ *([0-9.]+)([kmgtKMG]?B) ]]; then
        RX_VAL=${BASH_REMATCH[1]}
        RX_UNIT=${BASH_REMATCH[2]}
        TX_VAL=${BASH_REMATCH[3]}
        TX_UNIT=${BASH_REMATCH[4]}

        # Normalize units to uppercase
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
    elif [[ $stats_line =~ ([0-9.]+)([KMGT]?iB)\ */\ *([0-9.]+)([KMGT]?iB) ]]; then
        # Handle KiB, MiB, GiB, TiB format
        RX_VAL=${BASH_REMATCH[1]}
        RX_UNIT=${BASH_REMATCH[2]}
        TX_VAL=${BASH_REMATCH[3]}
        TX_UNIT=${BASH_REMATCH[4]}

        rx_bytes=$(awk -v val="$RX_VAL" -v unit="$RX_UNIT" 'BEGIN {
            if (unit == "KiB") print int(val * 1024)
            else if (unit == "MiB") print int(val * 1024 * 1024)
            else if (unit == "GiB") print int(val * 1024 * 1024 * 1024)
            else if (unit == "TiB") print int(val * 1024 * 1024 * 1024 * 1024)
            else print int(val)
        }')

        tx_bytes=$(awk -v val="$TX_VAL" -v unit="$TX_UNIT" 'BEGIN {
            if (unit == "KiB") print int(val * 1024)
            else if (unit == "MiB") print int(val * 1024 * 1024)
            else if (unit == "GiB") print int(val * 1024 * 1024 * 1024)
            else if (unit == "TiB") print int(val * 1024 * 1024 * 1024 * 1024)
            else print int(val)
        }')
    fi

    echo "$rx_bytes $tx_bytes"
}

# Helper function to strip ANSI escape codes
strip_ansi() {
    echo "$1" | sed 's/\x1b\[[0-9;]*[a-zA-Z]//g' | sed 's/\x1b\[[0-9;]*m//g' | tr -d '\r'
}

# Get initial stats (retry a few times if container not ready)
INITIAL_STATS="0B / 0B"
for i in {1..10}; do
    RAW_STATS=$(docker stats --no-stream --format "{{.NetIO}}" "$CONTAINER_NAME" 2>/dev/null || echo "0B / 0B")
    INITIAL_STATS=$(strip_ansi "$RAW_STATS")
    # Check if we got valid stats (not empty and not just "0B / 0B")
    if [ -n "$INITIAL_STATS" ] && [ "$INITIAL_STATS" != "0B / 0B" ] && [ "$INITIAL_STATS" != "-- / --" ] && [ "$INITIAL_STATS" != "" ]; then
        break
    fi
    if [ "$i" -eq 10 ]; then
        echo "Warning: Could not get initial stats after 10 attempts, using 0B / 0B" >> "$OUTPUT_FILE"
        INITIAL_STATS="0B / 0B"
        break
    fi
    sleep 1
done
echo "Initial: $INITIAL_STATS" >> "$OUTPUT_FILE"

INITIAL_BYTES=$(parse_stats_to_bytes "$INITIAL_STATS")
INITIAL_RX=$(echo $INITIAL_BYTES | awk '{print $1}')
INITIAL_TX=$(echo $INITIAL_BYTES | awk '{print $2}')

# Set up trap to write summary if interrupted (after INITIAL_STATS is set)
handle_exit() {
    # Ensure summary is written before exiting
    if [ -n "$OUTPUT_FILE" ] && [ -n "$INITIAL_STATS" ] && [ ! -f "${OUTPUT_FILE}.summary" ]; then
        # Get final stats one more time, or use last valid stats
        FINAL_STATS="0B / 0B"
        for i in {1..5}; do
            if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
                RAW_STATS=$(docker stats --no-stream --format "{{.NetIO}}" "$CONTAINER_NAME" 2>/dev/null || echo "0B / 0B")
                FINAL_STATS=$(strip_ansi "$RAW_STATS")
                if [ -n "$FINAL_STATS" ] && [ "$FINAL_STATS" != "-- / --" ] && [ "$FINAL_STATS" != "" ] && [ "$FINAL_STATS" != "0B / 0B" ]; then
                    LAST_VALID_STATS="$FINAL_STATS"
                    break
                fi
            fi
            sleep 0.2
        done

        # If final stats are 0B / 0B but we have a last valid stats, use that
        if [ "$FINAL_STATS" = "0B / 0B" ] && [ -n "$LAST_VALID_STATS" ] && [ "$LAST_VALID_STATS" != "0B / 0B" ]; then
            FINAL_STATS="$LAST_VALID_STATS"
        fi

        if [ -n "$FINAL_STATS" ] && [ "$FINAL_STATS" != "0B / 0B" ]; then
            echo "Final: $FINAL_STATS" >> "$OUTPUT_FILE"
            FINAL_BYTES=$(parse_stats_to_bytes "$FINAL_STATS")
            FINAL_RX=$(echo $FINAL_BYTES | awk '{print $1}')
            FINAL_TX=$(echo $FINAL_BYTES | awk '{print $2}')
            TOTAL_RX=$((FINAL_RX - INITIAL_RX))
            TOTAL_TX=$((FINAL_TX - INITIAL_TX))
            TOTAL_BYTES=$((TOTAL_RX + TOTAL_TX))

            {
                echo "RX_BYTES=$TOTAL_RX"
                echo "TX_BYTES=$TOTAL_TX"
                echo "TOTAL_BYTES=$TOTAL_BYTES"
            } > "${OUTPUT_FILE}.summary"
        fi
    fi
    exit 0
}
trap 'handle_exit' TERM INT

# Monitor periodically and capture stats
INTERVAL=5  # Check every 5 seconds
ITERATIONS=$((DURATION / INTERVAL))
LAST_VALID_STATS="0B / 0B"
for i in $(seq 1 $ITERATIONS); do
    # Check if container still exists - if not, break early but still write summary
    if ! docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        break
    fi
    RAW_STATS=$(docker stats --no-stream --format "{{.NetIO}}" "$CONTAINER_NAME" 2>/dev/null || echo "0B / 0B")
    STATS=$(strip_ansi "$RAW_STATS")
    echo "$STATS" >> "$OUTPUT_FILE"
    # Keep track of last valid (non-zero) stats
    if [ -n "$STATS" ] && [ "$STATS" != "0B / 0B" ] && [ "$STATS" != "-- / --" ] && [ "$STATS" != "" ]; then
        LAST_VALID_STATS="$STATS"
    fi
    sleep "$INTERVAL"
done

# Get final stats (retry a few times, even if container stopped)
# Use last valid stats if container stopped and returns 0B / 0B
FINAL_STATS="0B / 0B"
for i in {1..10}; do
    # Check if container exists (running or stopped)
    if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        RAW_STATS=$(docker stats --no-stream --format "{{.NetIO}}" "$CONTAINER_NAME" 2>/dev/null || echo "0B / 0B")
        FINAL_STATS=$(strip_ansi "$RAW_STATS")
        if [ -n "$FINAL_STATS" ] && [ "$FINAL_STATS" != "-- / --" ] && [ "$FINAL_STATS" != "" ] && [ "$FINAL_STATS" != "0B / 0B" ]; then
            LAST_VALID_STATS="$FINAL_STATS"
            break
        fi
    fi
    sleep 0.5
done

# If final stats are 0B / 0B but we have a last valid stats, use that
if [ "$FINAL_STATS" = "0B / 0B" ] && [ -n "$LAST_VALID_STATS" ] && [ "$LAST_VALID_STATS" != "0B / 0B" ]; then
    FINAL_STATS="$LAST_VALID_STATS"
fi

echo "Final: $FINAL_STATS" >> "$OUTPUT_FILE"

FINAL_BYTES=$(parse_stats_to_bytes "$FINAL_STATS")
FINAL_RX=$(echo $FINAL_BYTES | awk '{print $1}')
FINAL_TX=$(echo $FINAL_BYTES | awk '{print $2}')

# Calculate difference
TOTAL_RX=$((FINAL_RX - INITIAL_RX))
TOTAL_TX=$((FINAL_TX - INITIAL_TX))

echo "Monitoring complete. Results saved to: $OUTPUT_FILE"

# Check if bc is available, if not use awk for calculations
if command -v bc >/dev/null 2>&1; then
    USE_BC=true
else
    USE_BC=false
    echo "Warning: bc not found, using awk for calculations (may be less precise)"
fi

# TOTAL_RX and TOTAL_TX are already calculated above as the difference
# between initial and final stats

# Helper function to format bytes in human-readable format
format_bytes() {
    local bytes=$1
    if [ "$USE_BC" = true ]; then
        if [ $(echo "$bytes >= 1099511627776" | bc) -eq 1 ]; then
            echo "$(echo "scale=2; $bytes / 1099511627776" | bc) TB"
        elif [ $(echo "$bytes >= 1073741824" | bc) -eq 1 ]; then
            echo "$(echo "scale=2; $bytes / 1073741824" | bc) GB"
        elif [ $(echo "$bytes >= 1048576" | bc) -eq 1 ]; then
            echo "$(echo "scale=2; $bytes / 1048576" | bc) MB"
        elif [ $(echo "$bytes >= 1024" | bc) -eq 1 ]; then
            echo "$(echo "scale=2; $bytes / 1024" | bc) KB"
        else
            echo "${bytes} B"
        fi
    else
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
    fi
}

# Output summary
echo "=== Network Summary ==="
TOTAL_BYTES=$((TOTAL_RX + TOTAL_TX))
RX_FORMATTED=$(format_bytes $TOTAL_RX)
TX_FORMATTED=$(format_bytes $TOTAL_TX)
TOTAL_FORMATTED=$(format_bytes $TOTAL_BYTES)
echo "Total Received: $RX_FORMATTED"
echo "Total Sent: $TX_FORMATTED"
echo "Total: $TOTAL_FORMATTED"

# Save summary to file
{
    echo "RX_BYTES=$TOTAL_RX"
    echo "TX_BYTES=$TOTAL_TX"
    echo "TOTAL_BYTES=$TOTAL_BYTES"
} > "${OUTPUT_FILE}.summary"
