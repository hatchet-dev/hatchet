"use client";

import { useEffect } from "react";

const IMAGE_TAG_REGEX =
  /(ghcr\.io\/hatchet-dev\/hatchet\/hatchet-[a-z-]+:)latest\b/g;

let latestTag: Promise<string | undefined> | undefined;

function fetchLatestTag() {
  latestTag ??= fetch("/api/latest-version")
    .then((res) => (res.ok ? res.json() : undefined))
    .then((release) =>
      /^v\d+\.\d+\.\d+$/.test(release?.tag_name)
        ? (release.tag_name as string)
        : undefined,
    )
    .catch(() => undefined)
    .then((tag) => {
      if (!tag) latestTag = undefined;
      return tag;
    });
  return latestTag;
}

/**
 * Rewrites `ghcr.io/hatchet-dev/hatchet/*:latest` image references on the page
 * to the latest public release tag. Falls back to `:latest` if the GitHub API
 * is unreachable.
 */
function swapTags(root: Node, tag: string) {
  const swap = (node: Node) => {
    if (!node.nodeValue?.includes("ghcr.io/hatchet-dev/hatchet/")) return;
    const swapped = node.nodeValue.replace(IMAGE_TAG_REGEX, `$1${tag}`);
    if (swapped !== node.nodeValue) {
      node.nodeValue = swapped;
    }
  };
  if (root.nodeType === Node.TEXT_NODE) {
    swap(root);
    return;
  }
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
  for (let node = walker.nextNode(); node; node = walker.nextNode()) {
    swap(node);
  }
}

export default function LatestImageTags() {
  useEffect(() => {
    let cancelled = false;
    let observer: MutationObserver | undefined;
    fetchLatestTag().then((tag) => {
      if (!tag || cancelled) return;
      swapTags(document.body, tag);
      // Re-renders (hydration recovery, tab switches) can restore or remount
      // text with the original `:latest`, so keep watching for it.
      observer = new MutationObserver((mutations) => {
        for (const mutation of mutations) {
          if (mutation.type === "characterData") {
            swapTags(mutation.target, tag);
          } else {
            mutation.addedNodes.forEach((node) => swapTags(node, tag));
          }
        }
      });
      observer.observe(document.body, {
        subtree: true,
        childList: true,
        characterData: true,
      });
    });
    return () => {
      cancelled = true;
      observer?.disconnect();
    };
  }, []);

  return null;
}
