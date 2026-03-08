#!/usr/bin/env bash
# Installs system deps needed for Pillow when building from source (e.g. Python 3.14).
# Pillow requires libjpeg and zlib headers; wheels are not always available for newer Python.
set -e

case "$(uname -s)" in
  Linux)
    if command -v apt-get >/dev/null 2>&1; then
      sudo apt-get update
      sudo apt-get install -y libjpeg-dev zlib1g-dev
    else
      echo "Unsupported Linux package manager. Install libjpeg and zlib dev packages manually."
      exit 1
    fi
    ;;
  Darwin)
    if command -v brew >/dev/null 2>&1; then
      brew install libjpeg zlib
    else
      echo "macOS: Install Homebrew and run: brew install libjpeg zlib"
      exit 1
    fi
    ;;
  *)
    echo "Unsupported OS: $(uname -s). Install libjpeg and zlib dev packages for Pillow."
    exit 1
    ;;
esac
