#!/bin/sh

awk '
  /^## \[Unreleased\]/ {
      release++;
      next;
  }
  /^## \[[0-9]+\.[0-9]+\.[0-9]+\]/ {
      release++;
      next;
  }
  {
      if (release == 1) print;
      if (release > 1) exit;
  }' "CHANGELOG.md"
