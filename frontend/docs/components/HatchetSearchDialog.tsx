"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import MiniSearch, { type SearchResult } from "minisearch";
import posthog from "posthog-js";
import {
  SearchDialog,
  SearchDialogClose,
  SearchDialogContent,
  SearchDialogHeader,
  SearchDialogIcon,
  SearchDialogInput,
  SearchDialogList,
  SearchDialogOverlay,
  type SearchItemType,
  type SharedProps,
} from "fumadocs-ui/components/dialog/search";
import {
  MINISEARCH_OPTIONS,
  SEARCH_OPTIONS,
  rerankResults,
  expandSynonyms,
} from "@/lib/search-config";

let indexPromise: Promise<MiniSearch> | null = null;

function loadIndex(): Promise<MiniSearch> {
  if (indexPromise === null) {
    indexPromise = fetch("/llms-search-index.json")
      .then((res) => {
        if (!res.ok)
          throw new Error(`Failed to load search index: ${res.status}`);
        return res.text();
      })
      .then((json) => MiniSearch.loadJSON(json, MINISEARCH_OPTIONS));
  }
  return indexPromise;
}

function getContentSnippet(
  content: string | undefined,
  query: string,
  maxLen = 120,
): string {
  if (!content) return "";
  const plain = content
    .replace(/^#{1,6}\s+.*$/gm, "")
    .replace(/```[\s\S]*?```/g, "")
    .replace(/`[^`]*`/g, "")
    .replace(/^\|[-|\s:]+\|$/gm, "")
    .replace(/^\|.*\|$/gm, (row) =>
      row
        .replace(/^\||\|$/g, "")
        .split("|")
        .map((c) => c.trim())
        .filter(Boolean)
        .join(", "),
    )
    .replace(/\[([^\]]*)\]\([^)]*\)/g, "$1")
    .replace(/[*_~]+/g, "")
    .replace(/\n+/g, " ")
    .replace(/\s+/g, " ")
    .trim();
  if (!plain) return "";

  const words = query
    .trim()
    .split(/\s+/)
    .filter((w) => w.length > 1);
  if (words.length > 0) {
    const escaped = words
      .map((w) => w.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"))
      .join("|");
    const re = new RegExp(escaped, "i");
    const matchIdx = plain.search(re);
    if (matchIdx >= 0) {
      const start = Math.max(0, matchIdx - 40);
      const end = Math.min(plain.length, start + maxLen);
      let snippet = plain.slice(start, end).trim();
      if (start > 0) snippet = "…" + snippet;
      if (end < plain.length) snippet = snippet + "…";
      return snippet;
    }
  }

  if (plain.length <= maxLen) return plain;
  return plain.slice(0, maxLen).trim() + "…";
}

function idToRoute(id: string): string {
  return (
    "/" +
    id
      .replace("hatchet://docs/", "")
      .replace(/\/index$/, "")
      .replace(/\/index#/, "#")
  );
}

function getPageRoute(result: SearchResult): string {
  return (result.pageRoute as string) || result.id.replace(/#.*$/, "");
}

function getPageTitle(result: SearchResult): string {
  return (result.pageTitle as string) || (result.title as string) || result.id;
}

function toItems(results: SearchResult[], query: string): SearchItemType[] {
  const items: SearchItemType[] = [];
  const seenPages = new Set<string>();

  for (const r of results) {
    const pageRoute = idToRoute(getPageRoute(r));
    const route = idToRoute(r.id);

    if (!seenPages.has(pageRoute)) {
      seenPages.add(pageRoute);
      items.push({
        id: `page:${pageRoute}`,
        url: pageRoute,
        type: "page",
        content: getPageTitle(r),
      });
    }

    const title = (r.title as string) || getPageTitle(r);
    const isPageItself = route === pageRoute && title === getPageTitle(r);
    if (!isPageItself) {
      items.push({
        id: r.id,
        url: route,
        type: "heading",
        content: title,
      });
    }

    const snippet = getContentSnippet(r.content as string | undefined, query);
    if (snippet) {
      items.push({
        id: `${r.id}:snippet`,
        url: route,
        type: "text",
        content: snippet,
      });
    }
  }

  return items;
}

export default function HatchetSearchDialog(props: SharedProps) {
  const [search, setSearch] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [isLoading, setLoading] = useState(false);

  const sessionRef = useRef({ query: "", resultCount: 0, clicked: false });
  const prevOpenRef = useRef(false);

  useEffect(() => {
    sessionRef.current.query = search;
  }, [search]);

  useEffect(() => {
    if (prevOpenRef.current && !props.open) {
      const { query, resultCount, clicked } = sessionRef.current;
      const trimmed = query.trim();
      if (trimmed) {
        if (resultCount === 0) {
          posthog.capture("docs_search_no_results", { query: trimmed });
        } else if (!clicked) {
          posthog.capture("docs_search_abandoned", {
            query: trimmed,
            result_count: resultCount,
          });
        }
      }
      sessionRef.current = { query: "", resultCount: 0, clicked: false };
      setSearch("");
      setResults([]);
    }
    prevOpenRef.current = props.open;
  }, [props.open]);

  useEffect(() => {
    if (props.open) loadIndex().catch(() => {});
  }, [props.open]);

  useEffect(() => {
    if (!search.trim()) {
      setResults([]);
      return;
    }
    let cancelled = false;
    setLoading(true);
    loadIndex()
      .then((idx) => {
        if (cancelled) return;
        try {
          const expanded = expandSynonyms(search);
          const raw = idx.search(expanded, SEARCH_OPTIONS);
          const reranked = rerankResults(raw, search).slice(0, 20);
          setResults(reranked);
          sessionRef.current.resultCount = reranked.length;
        } catch {
          setResults([]);
          sessionRef.current.resultCount = 0;
        }
      })
      .catch(() => {})
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [search]);

  const items = useMemo(
    () => (search.trim() ? toItems(results, search) : null),
    [results, search],
  );

  return (
    <SearchDialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      search={search}
      onSearchChange={setSearch}
      isLoading={isLoading}
      onSelect={(item) => {
        sessionRef.current.clicked = true;
        posthog.capture("docs_search_result_clicked", {
          query: search.trim(),
          result_id: String(item.id),
        });
      }}
    >
      <SearchDialogOverlay />
      <SearchDialogContent>
        <SearchDialogHeader>
          <SearchDialogIcon />
          <SearchDialogInput placeholder="Search documentation…" />
          <SearchDialogClose />
        </SearchDialogHeader>
        <SearchDialogList items={items} />
      </SearchDialogContent>
    </SearchDialog>
  );
}
