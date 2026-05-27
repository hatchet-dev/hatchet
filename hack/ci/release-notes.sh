#!/bin/sh

# HACK(gregfurman): While we don't have proper conventional commits & sufficient version tagging
# history, let's bootstrap the commit history between 2 releases of each SDK.

get_pr_history() {
  local since_hash="$1" until_hash="$2"
  shift 2
  git log "${since_hash}..${until_hash}" --pretty=format:'%s' -- "$@" \
    | grep -oE '#[0-9]+' | tr -d '#' | sort -u \
    | while read -r pr; do
        gh pr view "$pr" \
          --repo hatchet-dev/hatchet \
          --json number,title,author,labels \
          --jq '"- \(.title) (#\(.number)) by @\(.author.login)"'
      done
}

get_latest_entry() {
  local sdk="$1"
  awk '
  /^## \[Unreleased\]/ {
      release++;
      print "";
      print "NOTE: This is a release candidate.";
      next;
  }
  /^## \[[0-9]+\.[0-9]+\.[0-9]+\]/ {
      release++;
      next;
  }
  {
      if (release == 1) print;
      if (release > 1) exit;
  }' "$sdk/CHANGELOG.md"
}


get_instructions() {
  local sdk="$1"
  awk '
  /^## (Installation|Documentation)/ {
    printing=1
  }
  /^## / && printing && !/^## (Installation|Documentation)/ {
    printing=0
  }
  printing
  ' "$sdk/README.md"
}

generate_release_notes() {
  local sdk="$1"
  local file
  local pattern
  local sdk_dir
  case "$sdk" in
    python)
      file="sdks/python/pyproject.toml"
      pattern='^version ='
      sdk_dir="sdks/python/"
      ;;
    typescript)
      file="sdks/typescript/package.json"
      pattern='"version"'
      sdk_dir="sdks/typescript/"
      ;;
    ruby)
      file="sdks/ruby/src/lib/hatchet/version.rb"
      pattern='VERSION ='
      sdk_dir="sdks/ruby/"
      ;;
    *)
      echo "Unknown SDK: $sdk" >&2
      return 1
      ;;
  esac

  get_latest_entry "$(dirname $file)"
  get_instructions "$(dirname $file)"

  local line
  line=$(grep -n "$pattern" "$file" | head -1 | cut -d: -f1)

  local hashes
  hashes=$(git log -2 -L "$line,$line:$file" --pretty=format:"%h" -s)

  local until_hash since_hash
  until_hash=$(echo "$hashes" | sed -n '1p')
  since_hash=$(echo "$hashes" | sed -n '2p')

  echo "## What's Changed?"
  echo

  get_pr_history "$sdk_dir" "$since_hash" "$until_hash"
}


echo "## Overview"

generate_release_notes "$1"
