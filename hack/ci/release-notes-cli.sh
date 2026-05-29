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

  --prose)
    # returns human-written prose in the top release section (between # Release and <!-- GENERATED:START -->)
    awk '
      /^# Release \[[0-9]+\.[0-9]+\.[0-9]+\]/ {
        found++;
        next;
      }
      found == 1 && /^## Changelog/ {
        exit;
      }
      found == 1 {
        print;
      }
    ' "$FILE"
    ;;

  --from-changelog)
    # returns everything from ## Changelog onwards in the top release section
    awk '
      /^## Changelog/ {
        found = 1;
      }
      found {
        print;
      }
    ' "$FILE"
    ;;

  *)
    echo "Usage: $0 [--latest|--unreleased|--all|--prose|--from-changelog] [CHANGELOG.md]" >&2
    exit 1
    ;;
esac
