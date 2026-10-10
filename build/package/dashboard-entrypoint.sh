#!/bin/sh

# Trap SIGTERM and SIGINT signals to gracefully shut down
trap 'shutdown' TERM INT

# Function to handle shutdown
shutdown() {
  echo "Gracefully shutting down hatchet-api..."
  kill -TERM "$HATCHET_API_PID"

  # Wait for hatchet-api to exit
  wait "$HATCHET_API_PID"

  echo "Shutting down static file server..."
  kill -TERM "$STATIC_PID"
  wait "$STATIC_PID"

  # Exit the script
  exit 0
}

# Start hatchet-api with any passed command line arguments in the background
./hatchet-api "$@" &
HATCHET_API_PID=$!

./hatchet-staticfileserver -port 80 -static-asset-dir ./html -api-proxy "http://localhost:${SERVER_PORT:-8080}" &
STATIC_PID=$!

while kill -0 "$HATCHET_API_PID" 2>/dev/null && kill -0 "$STATIC_PID" 2>/dev/null; do
  sleep 1
done

if ! kill -0 "$HATCHET_API_PID" 2>/dev/null; then
  wait "$HATCHET_API_PID"; rc=$?
  echo "hatchet-api exited with status $rc"
  kill -TERM "$STATIC_PID" 2>/dev/null; wait "$STATIC_PID"
else
  wait "$STATIC_PID"; rc=$?
  echo "static file server exited with status $rc"
  kill -TERM "$HATCHET_API_PID" 2>/dev/null; wait "$HATCHET_API_PID"
fi

exit "$rc"
