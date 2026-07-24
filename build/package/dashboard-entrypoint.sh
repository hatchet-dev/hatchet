#!/bin/sh

# Trap SIGTERM and SIGINT signals to gracefully shut down
trap 'shutdown' SIGTERM SIGINT

# Function to handle shutdown
shutdown() {
  echo "Gracefully shutting down hatchet-api..."
  kill -SIGTERM "$HATCHET_API_PID" 2>/dev/null
  kill -SIGTERM "$NGINX_PID" 2>/dev/null

  # Wait for child processes to exit
  wait "$HATCHET_API_PID" 2>/dev/null
  wait "$NGINX_PID" 2>/dev/null

  echo "Shutting down NGINX..."
  nginx -s quit

  # Exit the script
  exit 0
}

HATCHET_API_STATUS_FILE="/tmp/hatchet-api-status-$$"
NGINX_STATUS_FILE="/tmp/nginx-status-$$"

(
  trap 'kill -SIGTERM "$HATCHET_API_CHILD_PID" 2>/dev/null; wait "$HATCHET_API_CHILD_PID" 2>/dev/null; exit 0' SIGTERM SIGINT

  # Start hatchet-api with any passed command line arguments in the background
  ./hatchet-api "$@" &
  HATCHET_API_CHILD_PID=$!

  wait "$HATCHET_API_CHILD_PID"
  echo "$?" > "$HATCHET_API_STATUS_FILE"
) &
HATCHET_API_PID=$!

(
  trap 'nginx -s quit; exit 0' SIGTERM SIGINT
  nginx -g "daemon off;"
  echo "$?" > "$NGINX_STATUS_FILE"
) &
NGINX_PID=$!

while true; do
  if [ -f "$HATCHET_API_STATUS_FILE" ]; then
    EXIT_STATUS=$(cat "$HATCHET_API_STATUS_FILE")
    echo "hatchet-api exited with status $EXIT_STATUS; shutting down NGINX..."
    nginx -s quit || kill -SIGTERM "$NGINX_PID" 2>/dev/null
    wait "$NGINX_PID" 2>/dev/null
    rm -f "$HATCHET_API_STATUS_FILE" "$NGINX_STATUS_FILE"
    exit "$EXIT_STATUS"
  fi

  if [ -f "$NGINX_STATUS_FILE" ]; then
    EXIT_STATUS=$(cat "$NGINX_STATUS_FILE")
    echo "NGINX exited with status $EXIT_STATUS; shutting down hatchet-api..."
    kill -SIGTERM "$HATCHET_API_PID" 2>/dev/null
    wait "$HATCHET_API_PID" 2>/dev/null
    rm -f "$HATCHET_API_STATUS_FILE" "$NGINX_STATUS_FILE"
    exit "$EXIT_STATUS"
  fi

  sleep 1
done
