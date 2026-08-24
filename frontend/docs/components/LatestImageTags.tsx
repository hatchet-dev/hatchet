"use client";

import { useEffect } from "react";

const SWAPS: { guard: string; pattern: RegExp; replacement: string }[] = [
  {
    guard: "ghcr.io/hatchet-dev/hatchet/",
    pattern: /(ghcr\.io\/hatchet-dev\/hatchet\/hatchet-[a-z-]+:)latest\b/g,
    replacement: "$1{tag}",
  },
  {
    guard: "vX.Y.Z",
    pattern: /\bvX\.Y\.Z\b/g,
    replacement: "{tag}",
  },
];

const latestTags = new Map<string, Promise<string | undefined>>();

function fetchLatestTag(versionRepo?: string) {
  const key = versionRepo ?? "hatchet";
  if (!latestTags.has(key)) {
    latestTags.set(
      key,
      fetch(`/api/latest-version${versionRepo ? `?repo=${versionRepo}` : ""}`)
        .then((res) => (res.ok ? res.json() : undefined))
        .then((release) =>
          /^v\d+\.\d+\.\d+(-\d+)?$/.test(release?.tag_name)
            ? (release.tag_name as string)
            : undefined,
        )
        .catch(() => undefined)
        .then((tag) => {
          if (!tag) latestTags.delete(key);
          return tag;
        }),
    );
  }
  return latestTags.get(key)!;
}

function swapTags(root: Node, tag: string) {
  const swap = (node: Node) => {
    let value = node.nodeValue;
    if (!value) return;
    for (const { guard, pattern, replacement } of SWAPS) {
      if (!value.includes(guard)) continue;
      value = value.replace(pattern, replacement.replace("{tag}", tag));
    }
    if (value !== node.nodeValue) {
      node.nodeValue = value;
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

/**
 * Rewrites release-version placeholders on the page to the latest public
 * release tag: `ghcr.io/hatchet-dev/hatchet/*:latest` image references and the
 * literal `vX.Y.Z` placeholder. Falls back to the original text if the GitHub
 * API is unreachable.
 */
export default function LatestImageTags({
  versionRepo,
}: {
  versionRepo?: "hatchet-embedded";
}) {
  useEffect(() => {
    let cancelled = false;
    let observer: MutationObserver | undefined;
    fetchLatestTag(versionRepo).then((tag) => {
      if (!tag || cancelled) return;
      swapTags(document.body, tag);
      // Re-renders (hydration recovery, tab switches) can restore or remount
      // text with the original placeholder, so keep watching for it.
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
  }, [versionRepo]);

  return null;
}
