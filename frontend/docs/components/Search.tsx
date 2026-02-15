import React, {
  useCallback,
  useEffect,
  useRef,
  useState,
  type KeyboardEvent,
} from "react";
import { createPortal } from "react-dom";
import { useRouter } from "next/router";
import MiniSearch, { type SearchResult } from "minisearch";
import posthog from "posthog-js";
import {
  MINISEARCH_OPTIONS,
  SEARCH_OPTIONS,
  rerankResults,
  expandSynonyms,
} from "@/lib/search-config";

// ---------------------------------------------------------------------------
// Lazy singleton for the search index
// ---------------------------------------------------------------------------
let indexPromise: Promise<MiniSearch> | null = null;

function loadIndex(): Promise<MiniSearch> {
  if (!indexPromise) {
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

/** Convert a MiniSearch doc id to a Next.js route. */
function idToRoute(id: string): string {
  return "/" + id.replace("hatchet://docs/", "");
}

/** Extract the page route (without anchor) from a result. */
function getPageRoute(result: SearchResult): string {
  return (result.pageRoute as string) || result.id.replace(/#.*$/, "");
}

/** Get the page title from a result. */
function getPageTitle(result: SearchResult): string {
  return (result.pageTitle as string) || (result.title as string) || result.id;
}

/** Group results by page, maintaining overall order by first appearance. */
function groupByPage(
  results: SearchResult[],
): Array<{ pageRoute: string; pageTitle: string; items: SearchResult[] }> {
  const groups: Array<{
    pageRoute: string;
    pageTitle: string;
    items: SearchResult[];
  }> = [];
  const seen = new Map<string, number>();

  for (const r of results) {
    const route = getPageRoute(r);
    const idx = seen.get(route);
    if (idx !== undefined) {
      groups[idx].items.push(r);
    } else {
      seen.set(route, groups.length);
      groups.push({
        pageRoute: route,
        pageTitle: getPageTitle(r),
        items: [r],
      });
    }
  }

  return groups;
}

// ---------------------------------------------------------------------------
// Detect Mac for keyboard shortcut display
// ---------------------------------------------------------------------------
function useIsMac() {
  const [isMac, setIsMac] = useState(false);
  useEffect(() => {
    setIsMac(/(Mac|iPhone|iPod|iPad)/i.test(navigator.platform));
  }, []);
  return isMac;
}

// ---------------------------------------------------------------------------
// Highlight matches in text
// ---------------------------------------------------------------------------
function HighlightMatches({ text, query }: { text: string; query: string }) {
  if (!query.trim()) return <>{text}</>;

  try {
    // Build regex from individual query words for better highlighting
    const words = query
      .trim()
      .split(/\s+/)
      .filter((w) => w.length > 1)
      .map((w) => w.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"));
    if (words.length === 0) return <>{text}</>;

    const re = new RegExp(`(${words.join("|")})`, "ig");
    const parts = text.split(re);
    return (
      <>
        {parts.map((part, i) =>
          re.test(part) ? (
            <span key={i} className="_text-primary-600">
              {part}
            </span>
          ) : (
            <React.Fragment key={i}>{part}</React.Fragment>
          ),
        )}
      </>
    );
  } catch {
    return <>{text}</>;
  }
}

// ---------------------------------------------------------------------------
// Spinner icon (matches Nextra's loading spinner)
// ---------------------------------------------------------------------------
function SpinnerIcon() {
  return (
    <svg
      className="_size-5 _animate-spin _text-gray-400"
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
    >
      <circle
        className="_opacity-25"
        cx="12"
        cy="12"
        r="10"
        stroke="currentColor"
        strokeWidth="4"
      />
      <path
        className="_opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
      />
    </svg>
  );
}

// ---------------------------------------------------------------------------
// Search component
// ---------------------------------------------------------------------------
export default function Search({ className }: { className?: string }) {
  const router = useRouter();
  const isMac = useIsMac();
  const inputRef = useRef<HTMLInputElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const listRef = useRef<HTMLUListElement>(null);

  const [focused, setFocused] = useState(false);
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [isOpen, setIsOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(-1);
  const [indexReady, setIndexReady] = useState(false);
  const [loading, setLoading] = useState(false);
  const [dropdownPos, setDropdownPos] = useState<{
    top: number;
    right: number;
    width: number;
  } | null>(null);

  // ---------------------------------------------------------------------------
  // PostHog search-miss tracking
  // ---------------------------------------------------------------------------
  // Mutable ref tracks the current search session without triggering re-renders.
  // We capture events when the dropdown closes (isOpen → false).
  const searchSessionRef = useRef({
    query: "",
    resultCount: 0,
    clicked: false,
  });
  const prevIsOpenRef = useRef(false);

  // Fire PostHog events when the search dropdown closes
  useEffect(() => {
    if (prevIsOpenRef.current && !isOpen) {
      const { query: q, resultCount, clicked } = searchSessionRef.current;
      const trimmed = q.trim();
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
      searchSessionRef.current = { query: "", resultCount: 0, clicked: false };
    }
    prevIsOpenRef.current = isOpen;
  }, [isOpen]);

  // Eagerly start loading the index when the component mounts
  useEffect(() => {
    loadIndex().then(() => setIndexReady(true));
  }, []);

  // Run the search when the query changes
  useEffect(() => {
    if (!query.trim()) {
      setResults([]);
      return;
    }

    function runSearch(idx: MiniSearch) {
      try {
        const expanded = expandSynonyms(query);
        const raw = idx.search(expanded, SEARCH_OPTIONS);
        // Rerank against the original query so title matching is accurate
        const reranked = rerankResults(raw, query).slice(0, 20);
        setResults(reranked);
        searchSessionRef.current.resultCount = reranked.length;
      } catch {
        // Gracefully handle invalid queries (e.g. punctuation-only input)
        setResults([]);
        searchSessionRef.current.resultCount = 0;
      }
    }

    if (!indexReady) {
      setLoading(true);
      loadIndex()
        .then((idx) => {
          setIndexReady(true);
          setLoading(false);
          runSearch(idx);
        })
        .catch(() => setLoading(false));
      return;
    }

    loadIndex()
      .then(runSearch)
      .catch(() => {});
  }, [query, indexReady]);

  // Global keyboard shortcut: / or Cmd/Ctrl+K
  useEffect(() => {
    function onKeyDown(e: globalThis.KeyboardEvent) {
      if (
        e.key === "/" &&
        !e.metaKey &&
        !e.ctrlKey &&
        !["INPUT", "TEXTAREA"].includes(
          (e.target as HTMLElement)?.tagName || "",
        )
      ) {
        e.preventDefault();
        inputRef.current?.focus();
      }
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        inputRef.current?.focus();
      }
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, []);

  // Close on outside click
  useEffect(() => {
    function onClick(e: MouseEvent) {
      const target = e.target as Node;
      if (
        containerRef.current &&
        !containerRef.current.contains(target) &&
        listRef.current &&
        !listRef.current.contains(target)
      ) {
        setIsOpen(false);
      } else if (
        containerRef.current &&
        !containerRef.current.contains(target) &&
        !listRef.current
      ) {
        setIsOpen(false);
      }
    }
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, []);

  // Close on route change
  useEffect(() => {
    const handleRouteChange = () => {
      setIsOpen(false);
      setQuery("");
      inputRef.current?.blur();
    };
    router.events.on("routeChangeComplete", handleRouteChange);
    return () => router.events.off("routeChangeComplete", handleRouteChange);
  }, [router]);

  // Scroll active item into view
  useEffect(() => {
    if (activeIndex >= 0 && listRef.current) {
      const item = listRef.current.children[activeIndex] as HTMLElement;
      item?.scrollIntoView({ block: "nearest" });
    }
  }, [activeIndex]);

  const showDropdown = isOpen && query.trim().length > 0;
  const hasResults = results.length > 0;
  const grouped = hasResults ? groupByPage(results) : [];

  // Build a flat list of items for keyboard navigation
  const flatItems: SearchResult[] = grouped.flatMap((g) => g.items);

  // Compute dropdown position based on input bounding rect
  useEffect(() => {
    if (!showDropdown || !containerRef.current) {
      setDropdownPos(null);
      return;
    }
    const updatePos = () => {
      const rect = containerRef.current?.getBoundingClientRect();
      if (rect) {
        setDropdownPos({
          top: rect.bottom + 8,
          right: window.innerWidth - rect.right,
          width: Math.max(rect.width, 576),
        });
      }
    };
    updatePos();
    window.addEventListener("scroll", updatePos, true);
    window.addEventListener("resize", updatePos);
    return () => {
      window.removeEventListener("scroll", updatePos, true);
      window.removeEventListener("resize", updatePos);
    };
  }, [showDropdown]);

  const navigate = useCallback(
    (id: string) => {
      const route = idToRoute(id);
      searchSessionRef.current.clicked = true;
      posthog.capture("docs_search_result_clicked", {
        query: searchSessionRef.current.query.trim(),
        result_id: id,
        result_route: route,
      });
      setIsOpen(false);
      setQuery("");
      router.push(route);
    },
    [router],
  );

  const onKeyDown = useCallback(
    (e: KeyboardEvent<HTMLInputElement>) => {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setActiveIndex((i) => Math.min(i + 1, flatItems.length - 1));
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setActiveIndex((i) => Math.max(i - 1, 0));
      } else if (e.key === "Enter") {
        e.preventDefault();
        const idx = activeIndex >= 0 ? activeIndex : 0;
        if (flatItems[idx]) navigate(flatItems[idx].id);
      } else if (e.key === "Escape") {
        setIsOpen(false);
        inputRef.current?.blur();
      }
    },
    [flatItems, activeIndex, navigate],
  );

  return (
    <div
      ref={containerRef}
      className={`_not-prose _relative _flex _items-center _text-gray-900 dark:_text-gray-300 ${className || ""}`}
    >
      {/* Search input */}
      <input
        ref={inputRef}
        type="search"
        autoComplete="off"
        spellCheck={false}
        placeholder="Search documentation…"
        value={query}
        onChange={(e) => {
          const val = e.target.value;
          setQuery(val);
          searchSessionRef.current.query = val;
          setIsOpen(true);
          setActiveIndex(-1);
        }}
        onFocus={() => {
          setFocused(true);
          if (query.trim()) setIsOpen(true);
        }}
        onBlur={() => setFocused(false)}
        onKeyDown={onKeyDown}
        className={[
          "_rounded-lg _px-3 _py-2 _transition-colors",
          "_w-full md:_w-64",
          "_text-base _leading-tight md:_text-sm",
          focused
            ? "_bg-transparent nextra-focusable"
            : "_bg-black/[.05] dark:_bg-gray-50/10",
          "placeholder:_text-gray-500 dark:placeholder:_text-gray-400",
          "contrast-more:_border contrast-more:_border-current",
          "[&::-webkit-search-cancel-button]:_appearance-none",
        ].join(" ")}
      />

      {/* Keyboard shortcut indicator */}
      {!focused && !query && (
        <kbd
          className={[
            "_absolute _my-1.5 _select-none ltr:_right-1.5 rtl:_left-1.5",
            "_h-5 _rounded _bg-white _px-1.5 _font-mono _text-[11px] _font-medium _text-gray-500",
            "_border dark:_border-gray-100/20 dark:_bg-black/50",
            "contrast-more:_border-current contrast-more:_text-current contrast-more:dark:_border-current",
            "_pointer-events-none _flex _items-center _gap-1",
            "max-sm:_hidden",
          ].join(" ")}
        >
          {isMac ? (
            <>
              <span className="_text-xs">⌘</span> K
            </>
          ) : (
            "CTRL K"
          )}
        </kbd>
      )}

      {/* Results dropdown (portaled to body to escape overflow:hidden ancestors) */}
      {showDropdown &&
        dropdownPos &&
        typeof document !== "undefined" &&
        createPortal(
          <ul
            ref={listRef}
            style={{
              position: "fixed",
              top: dropdownPos.top,
              right: dropdownPos.right,
              width: dropdownPos.width,
              zIndex: 50,
            }}
            className={[
              "nextra-search-results nextra-scrollbar",
              "_rounded-xl _py-2.5 _shadow-xl",
              "_border _border-gray-200 dark:_border-neutral-800",
              "_backdrop-blur-lg _bg-[rgb(var(--nextra-bg),.8)]",
              "_max-h-[min(calc(100vh-5rem),400px)]",
              "_overflow-y-auto",
              "contrast-more:_border contrast-more:_border-gray-900 contrast-more:dark:_border-gray-50",
              "_transition-opacity _opacity-100",
            ].join(" ")}
          >
            {loading && (
              <li className="_flex _select-none _justify-center _gap-2 _p-8 _text-center _text-sm _text-gray-400">
                <SpinnerIcon />
                Loading…
              </li>
            )}

            {!loading && !hasResults && (
              <li className="_flex _select-none _justify-center _gap-2 _p-8 _text-center _text-sm _text-gray-400">
                No results for &ldquo;{query}&rdquo;
              </li>
            )}

            {(() => {
              let flatIdx = 0;
              return grouped.map((group) => (
                <li key={group.pageRoute} className="_mt-1 first:_mt-0">
                  {/* Page title header */}
                  <div className="_mx-2.5 _mb-1 _mt-2 _select-none _border-b _border-black/10 _px-2.5 _pb-1.5 _text-xs _font-semibold _uppercase _tracking-wider _text-gray-500 dark:_border-white/20 dark:_text-gray-400 first:_mt-0">
                    <HighlightMatches text={group.pageTitle} query={query} />
                  </div>
                  {/* Section items */}
                  <ul>
                    {group.items.map((result) => {
                      const idx = flatIdx++;
                      return (
                        <li key={result.id}>
                          <a
                            href={idToRoute(result.id)}
                            onClick={(e) => {
                              e.preventDefault();
                              navigate(result.id);
                            }}
                            onMouseEnter={() => setActiveIndex(idx)}
                            className={[
                              "_mx-2.5 _break-words _rounded-md",
                              "_block _scroll-m-12 _px-2.5 _py-2",
                              "_cursor-pointer",
                              "contrast-more:_border",
                              idx === activeIndex
                                ? "_text-primary-600 contrast-more:_border-current _bg-primary-500/10"
                                : "_text-gray-800 dark:_text-gray-300 contrast-more:_border-transparent",
                            ].join(" ")}
                          >
                            <div className="_text-base _font-semibold _leading-5">
                              <HighlightMatches
                                text={(result.title as string) || result.id}
                                query={query}
                              />
                            </div>
                            <div className="excerpt _mt-1 _text-sm _leading-[1.35rem] _text-gray-600 dark:_text-gray-400 contrast-more:dark:_text-gray-50">
                              {idToRoute(result.id)}
                            </div>
                          </a>
                        </li>
                      );
                    })}
                  </ul>
                </li>
              ));
            })()}
          </ul>,
          document.body,
        )}
    </div>
  );
}
