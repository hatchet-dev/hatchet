import React, { useEffect, useRef, useState } from 'react';
import AnsiToHtml from 'ansi-to-html';
import DOMPurify from 'dompurify';

const convert = new AnsiToHtml({
  newline: true,
  bg: 'transparent',
});

export interface ExtendedLogLine {
  badge?: React.ReactNode;
  /** @format date-time */
  timestamp?: string;
  instance?: string;
  line: string;
}

type LogProps = {
  logs: ExtendedLogLine[];
  onTopReached: () => void;
  onBottomReached: () => void;
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

const LoggingComponent: React.FC<LogProps> = ({
  logs,
  onTopReached,
  onBottomReached,
  autoScroll = true,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [refreshing, setRefreshing] = useState(false);
  const [lastTopCall, setLastTopCall] = useState<number>(0);
  const [lastBottomCall, setLastBottomCall] = useState<number>(0);
  const [firstMount, setFirstMount] = useState<boolean>(true);
  const previousScrollHeightRef = useRef<number>(0);

  const handleScroll = () => {
    if (!containerRef.current) {
      return;
    }
    const { scrollTop, scrollHeight, clientHeight } = containerRef.current;
    previousScrollHeightRef.current = scrollHeight;
    const now = Date.now();

    if (scrollTop === 0 && now - lastTopCall >= 1000) {
      if (logs.length > 0) {
        onTopReached();
      }
      setLastTopCall(now);
    } else if (
      scrollTop + clientHeight >= scrollHeight &&
      now - lastBottomCall >= 1000
    ) {
      if (logs.length > 0) {
        onBottomReached();
      }
      setLastBottomCall(now);
    }
  };

  useEffect(() => {
    setTimeout(() => {
      const container = containerRef.current;

      if (container && container.scrollHeight > container.clientHeight) {
        if (firstMount && autoScroll) {
          container.scrollTo({
            top: container.scrollHeight,
            behavior: 'smooth',
          });

          setFirstMount(false);
        }
      }
    }, 250);
  }, [containerRef, firstMount, autoScroll]);

  useEffect(() => {
    if (refreshing) {
      const timer = setTimeout(() => {
        setRefreshing(false);
      }, 1000);
      return () => clearTimeout(timer);
    }
  }, [refreshing]);

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

  const showLogs =
    logs.length > 0
      ? logs
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
    <div
      className="w-full mx-auto overflow-y-auto p-6 text-indigo-300 font-mono text-xs rounded-md max-h-[25rem] min-h-[25rem] bg-muted"
      ref={containerRef}
      onScroll={handleScroll}
    >
      {refreshing && (
        <div className="absolute top-0 left-0 right-0 bg-gray-800 text-white p-2 text-center">
          Refreshing...
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
            className="pb-2 break-all overflow-x-hidden"
            id={'log' + i}
          >
            {log.badge}
            {log.timestamp && (
              <span className="text-gray-500 mr-2 ml--2">
                {new Date(log.timestamp)
                  .toLocaleString('sv', options)
                  .replace(',', '.')
                  .replace(' ', 'T')}
              </span>
            )}
            {log.instance && (
              <span className="text-foreground dark:text-white mr-2 ml--2">
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

export default LoggingComponent;
