#!/bin/bash

# Trap SIGTERM and SIGINT signals to gracefully shut down
trap 'shutdown' SIGTERM SIGINT

# Function to handle shutdown
shutdown() {
  echo "Gracefully shutting down hatchet-api..."
  kill -SIGTERM "$HATCHET_API_PID"

  # Wait for hatchet-api to exit
  wait "$HATCHET_API_PID"

  echo "Shutting down NGINX..."
  nginx -s quit

  # Exit the script
  exit 0
}

# Start hatchet-api with any passed command line arguments in the background
./hatchet-api "$@" &
HATCHET_API_PID=$!

# Override the template-style {{ .BasePath }} with the envar $BASE_PATH
: "${BASE_PATH:=/}"
sed -i "s|{{ .BasePath }}|${BASE_PATH}|g" /usr/share/nginx/html/index.html

# Start NGINX in the foreground
nginx -g "daemon off;"
