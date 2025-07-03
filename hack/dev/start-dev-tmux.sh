#!/bin/bash

set -e

SESSION_NAME="hatchet-dev"

# Check if tmux is installed
if ! command -v tmux &> /dev/null; then
    echo "tmux is not installed. Please install tmux first."
    exit 1
fi

# Kill existing session if it exists
if tmux has-session -t $SESSION_NAME 2>/dev/null; then
    echo "Killing existing session: $SESSION_NAME"
    tmux kill-session -t $SESSION_NAME
fi

# Start database first
echo "Starting database..."
docker compose up -d
sleep 3

# Create new session with main window
echo "Creating new tmux session: $SESSION_NAME"
tmux new-session -d -s $SESSION_NAME -n "hatchet-dev"

# Enable pane titles
tmux set-option -t $SESSION_NAME pane-border-status top
tmux set-option -t $SESSION_NAME pane-border-format "#{pane_index}: #{pane_title}"

# Start API in the first pane (left half)
tmux select-pane -t $SESSION_NAME:hatchet-dev.0 -T "API"
tmux send-keys -t $SESSION_NAME:hatchet-dev "task start-api" C-m

# Split horizontally to create right half for engine
tmux split-window -h -t $SESSION_NAME:hatchet-dev
tmux select-pane -t $SESSION_NAME:hatchet-dev.1 -T "Engine"
tmux send-keys -t $SESSION_NAME:hatchet-dev "task start-engine" C-m

# Split the right pane vertically to create a smaller bottom pane for frontend
tmux split-window -v -t $SESSION_NAME:hatchet-dev.1
tmux select-pane -t $SESSION_NAME:hatchet-dev.2 -T "Frontend"
tmux send-keys -t $SESSION_NAME:hatchet-dev "task start-frontend" C-m

# Resize panes to make frontend smaller (30% of right side)
tmux resize-pane -t $SESSION_NAME:hatchet-dev.2 -y 30%

# Select the first pane (API)
tmux select-pane -t $SESSION_NAME:hatchet-dev.0

echo "Development environment started in tmux session: $SESSION_NAME"
echo "Attaching to session..."
echo "To detach from the session, press: Ctrl-b d"
echo "To kill the session, run: tmux kill-session -t $SESSION_NAME"

# Attach to the session
tmux attach-session -t $SESSION_NAME