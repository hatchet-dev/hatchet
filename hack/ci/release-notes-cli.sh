#!/bin/sh

MODE="${1:-latest}"
FILE="${2:-CHANGELOG.md}"

case "$MODE" in
  --latest)
    # returns the latest Released section
    awk '
      /^# Release \[[0-9]+\.[0-9]+\.[0-9]+\]/ {
        release++;
        print;
        next;
      }
      {
        if (release == 1) print;
        if (release > 1) exit;
      }
    ' "$FILE"
    ;;
  --unreleased)
    # returns the [Unreleased] section
    awk '
      /^# \[Unreleased\]/ {
        in_section = 1;
        print;
        next;
      }
      /^# Release \[[0-9]+\.[0-9]+\.[0-9]+\]/ {
        if (in_section) exit;
      }
      {
        if (in_section) print;
      }
    ' "$FILE"
    ;;

  --all)
    # returns all released sections
    awk '
      /^# Release \[[0-9]+\.[0-9]+\.[0-9]+\]/ {
        release++;
        print;
        next;
      }
      {
        if (release >= 1) print
      }
    ' "$FILE"
    ;;

  *)
    echo "Usage: $0 [--latest|--unreleased|--all] [CHANGELOG.md]" >&2
    exit 1
    ;;
esac
