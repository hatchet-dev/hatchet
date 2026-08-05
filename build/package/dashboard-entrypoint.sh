#!/bin/sh

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

: "${BASE_PATH:=/}"
case "$BASE_PATH" in /*) ;; *) BASE_PATH="/$BASE_PATH" ;; esac
case "$BASE_PATH" in */) ;; *) BASE_PATH="$BASE_PATH/" ;; esac

sed -i "s|{{ .BasePath }}|${BASE_PATH}|g" /usr/share/nginx/html/index.html

if [ "$BASE_PATH" = "/" ]; then
  cat > /etc/nginx/conf.d/hatchet-app.conf <<EOF
location / {
    try_files \$uri /index.html;
}
EOF
else
  cat > /etc/nginx/conf.d/hatchet-app.conf <<EOF
location = ${BASE_PATH%/} {
    return 302 ${BASE_PATH};
}
location ${BASE_PATH} {
    rewrite ^${BASE_PATH}(.*)\$ /\$1 break;
    try_files \$uri /index.html;
}
location = /index.html {
    internal;
}
location / {
    return 404;
}
EOF
fi

# Start NGINX in the foreground
nginx -g "daemon off;"
