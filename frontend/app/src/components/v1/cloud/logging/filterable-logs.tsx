import { LogSearchInput } from './log-search/log-search-input';
import { useLogSearch } from './log-search/use-log-search';
import { LogLine } from '@/lib/api/generated/cloud/data-contracts';
import { ListCloudLogsQuery } from '@/lib/api/queries';
import AnsiToHtml from 'ansi-to-html';
import DOMPurify from 'dompurify';
import React, { useEffect, useRef, useMemo } from 'react';

const convert = new AnsiToHtml({
  newline: true,
  bg: 'transparent',
});

type LogProps = {
  logs: LogLine[];
  onTopReached: () => void;
  onBottomReached: () => void;
  onInfiniteScroll?: (scrollMetrics: {
    scrollTop: number;
    scrollHeight: number;
    clientHeight: number;
  }) => void;
  onSearchChange?: (query: ListCloudLogsQuery) => void;
  autoScroll?: boolean;
};

const options: Intl.DateTimeFormatOptions = {
  year: 'numeric',
  month: 'numeric',
  day: 'numeric',
  hour: 'numeric',
  minute: 'numeric',
  second: 'numeric',
};

const FilterableLogs: React.FC<LogProps> = ({
  logs,
  onTopReached,
  onBottomReached,
  onInfiniteScroll,
  onSearchChange,
  autoScroll = true,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const refreshingRef = useRef(false);
  const lastTopCallRef = useRef<number>(0);
  const lastBottomCallRef = useRef<number>(0);
  const firstMountRef = useRef<boolean>(true);
  const previousScrollHeightRef = useRef<number>(0);
  const lastInfiniteScrollCallRef = useRef<number>(0);

  const { queryString, setQueryString, parsedQuery, apiQueryParams } =
    useLogSearch();

  useEffect(() => {
    onSearchChange?.(apiQueryParams);
  }, [apiQueryParams, onSearchChange]);
  const handleScroll = () => {
    if (!containerRef.current) {
      return;
    }
    const { scrollTop, scrollHeight, clientHeight } = containerRef.current;
    previousScrollHeightRef.current = scrollHeight;
    const now = Date.now();

    if (
      onInfiniteScroll &&
      logs.length > 0 &&
      now - lastInfiniteScrollCallRef.current >= 100
    ) {
      onInfiniteScroll({
        scrollTop,
        scrollHeight,
        clientHeight,
      });
      lastInfiniteScrollCallRef.current = now;
      return;
    }

    if (scrollTop === 0 && now - lastTopCallRef.current >= 1000) {
      if (logs.length > 0) {
        onTopReached();
      }
      lastTopCallRef.current = now;
    } else if (
      scrollTop + clientHeight >= scrollHeight &&
      now - lastBottomCallRef.current >= 1000
    ) {
      if (logs.length > 0) {
        onBottomReached();
      }
      lastBottomCallRef.current = now;
    }
  };

  useEffect(() => {
    setTimeout(() => {
      const container = containerRef.current;

      if (container && container.scrollHeight > container.clientHeight) {
        if (firstMountRef.current && autoScroll) {
          container.scrollTo({
            top: container.scrollHeight,
            behavior: 'smooth',
          });

          firstMountRef.current = false;
        }
      }
    }, 250);
  }, [autoScroll]);

  useEffect(() => {
    if (!autoScroll) {
      return;
    }

    const container = containerRef.current;
    if (!container) {
      return;
    }

    const previousScrollHeight = previousScrollHeightRef.current;
    const currentScrollHeight = container.scrollHeight;
    const { scrollTop, clientHeight } = container;

    const isAtBottom = scrollTop + clientHeight >= previousScrollHeight;

    if (!isAtBottom) {
      const newScrollTop =
        scrollTop + (currentScrollHeight - previousScrollHeight);
      container.scrollTo({ top: newScrollTop });
    } else {
      container.scrollTo({ top: currentScrollHeight, behavior: 'smooth' });
    }
  }, [logs, autoScroll]);

  // todo: remove this when we implement server-side filtering
  const filteredLogs = useMemo(() => {
    if (!parsedQuery || !queryString.trim()) {
      return logs;
    }

    return logs.filter((log, index) => {
      const rawLog = logs[index];

      if (parsedQuery.search) {
        const searchLower = parsedQuery.search.toLowerCase();
        const lineMatches = log.line?.toLowerCase().includes(searchLower);
        const instanceMatches = log.instance
          ?.toLowerCase()
          .includes(searchLower);
        if (!lineMatches && !instanceMatches) {
          return false;
        }
      }

      if (parsedQuery.level && rawLog) {
        const levelLower = parsedQuery.level.toLowerCase();
        const rawLevel = rawLog.level?.toLowerCase();
        const lineHasLevel = log.line?.toLowerCase().includes(levelLower);
        if (rawLevel !== levelLower && !lineHasLevel) {
          return false;
        }
      }

      return true;
    });
  }, [logs, parsedQuery, queryString]);

  const showLogs =
    filteredLogs.length > 0
      ? filteredLogs
      : logs.length > 0
        ? [] // Show nothing if filters match nothing
        : [
            {
              line: 'Waiting for logs...',
              timestamp: new Date().toISOString(),
              instance: 'Hatchet',
            },
          ];

  const sortedLogs = [...showLogs].sort((a, b) => {
    if (!a.timestamp || !b.timestamp) {
      return 0;
    }

    return new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime();
  });

  return (
    <div className="flex flex-col justify-center gap-y-2">
      <LogSearchInput value={queryString} onChange={setQueryString} />
      <div
        className="scrollbar-thin scrollbar-track-muted scrollbar-thumb-muted-foreground mx-auto max-h-[25rem] min-h-[25rem] w-full overflow-y-auto rounded-md bg-muted p-6 font-mono text-xs text-indigo-300"
        ref={containerRef}
        onScroll={handleScroll}
      >
        {refreshingRef.current && (
          <div className="absolute left-0 right-0 top-0 bg-gray-800 p-2 text-center text-white">
            Refreshing...
          </div>
        )}
        {sortedLogs.length === 0 && queryString.trim() && (
          <div className="text-muted-foreground">
            No logs match your search criteria
          </div>
        )}
        {sortedLogs.map((log, i) => {
          const sanitizedHtml = DOMPurify.sanitize(
            convert.toHtml(log.line || ''),
            {
              USE_PROFILES: { html: true },
            },
          );

          const logHash = log.timestamp + generateHash(log.line);

          return (
            <p
              key={logHash}
              className="overflow-x-hidden whitespace-pre-wrap break-all pb-2"
              id={'log' + i}
            >
              {log.timestamp && (
                <span className="ml--2 mr-2 text-gray-500">
                  {new Date(log.timestamp)
                    .toLocaleString('sv', options)
                    .replace(',', '.')
                    .replace(' ', 'T')}
                </span>
              )}
              {log.instance && (
                <span className="ml--2 mr-2 text-foreground dark:text-white">
                  {log.instance}
                </span>
              )}
              <span
                dangerouslySetInnerHTML={{
                  __html: sanitizedHtml,
                }}
              />
            </p>
          );
        })}
      </div>
    </div>
  );
};

const generateHash = (input: string | undefined): string => {
  if (!input) {
    return Math.random().toString(36).substring(2, 15);
  }
  const trimmedInput = input.substring(0, 50);
  return cyrb53(trimmedInput) + '';
};

// source: https://github.com/bryc/code/blob/master/jshash/experimental/cyrb53.js
const cyrb53 = function (str: string, seed = 0) {
  let h1 = 0xdeadbeef ^ seed,
    h2 = 0x41c6ce57 ^ seed;
  for (let i = 0, ch; i < str.length; i++) {
    ch = str.charCodeAt(i);
    h1 = Math.imul(h1 ^ ch, 2654435761);
    h2 = Math.imul(h2 ^ ch, 1597334677);
  }
  h1 = Math.imul(h1 ^ (h1 >>> 16), 2246822507);
  h1 ^= Math.imul(h2 ^ (h2 >>> 13), 3266489909);
  h2 = Math.imul(h2 ^ (h2 >>> 16), 2246822507);
  h2 ^= Math.imul(h1 ^ (h1 >>> 13), 3266489909);
  return 4294967296 * (2097151 & h2) + (h1 >>> 0);
};

export default FilterableLogs;
