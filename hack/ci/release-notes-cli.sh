#!/bin/sh
set -eu

PREV="$(gh release view --json tagName --jq '.tagName' 2>/dev/null || echo "")"
CURRENT="${GITHUB_REF_NAME}"

# Extracts the relevant section of the RELEASE_NOTES.md

awk '
/^## \[Unreleased\]/ {
    print "NOTE: This is a release candidate.";
    exit 1;
}
/^## Release v[0-9]+\.[0-9]+\.[0-9]+ /{
    release++;
    next;
}
{
    if (release == 1) print;
    if (release > 1) exit;
}' "RELEASE_NOTES.md"

echo ""
echo "## What's Changed?"
echo ""

# Extracts the sections/entries of the CHANGELOG.md between the previous release and latest tag.

awk -v cur="${CURRENT#v}" -v prev="${PREV#v}" '
$0 ~ "^## \\[" cur "\\]" {
    release++;
    print;
    next;
}
$0 ~ "^## \\[" prev "\\]" {
    exit;
}
{
    if (release == 1) print;
}' CHANGELOG.md
