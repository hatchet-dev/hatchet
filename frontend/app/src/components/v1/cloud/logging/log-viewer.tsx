import { LogLine } from './log-search/use-logs';
import AnsiToHtml from 'ansi-to-html';
import DOMPurify from 'dompurify';
import { useRef, useEffect } from 'react';

const convert = new AnsiToHtml({
  newline: true,
  bg: 'transparent',
});

const DATE_FORMAT_OPTIONS: Intl.DateTimeFormatOptions = {
  year: 'numeric',
  month: 'numeric',
  day: 'numeric',
  hour: 'numeric',
  minute: 'numeric',
  second: 'numeric',
};

export interface LogViewerProps {
  logs: LogLine[];
  onScroll?: (scrollData: {
    scrollTop: number;
    scrollHeight: number;
    clientHeight: number;
  }) => void;
  emptyMessage?: string;
}

export function LogViewer({
  logs,
  onScroll,
  emptyMessage = 'Waiting for logs...',
}: LogViewerProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const previousScrollHeightRef = useRef<number>(0);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const previousScrollHeight = previousScrollHeightRef.current;
    const currentScrollHeight = container.scrollHeight;
    const { scrollTop, clientHeight } = container;

    const isAtBottom = scrollTop + clientHeight >= previousScrollHeight - 10;

    if (isAtBottom) {
      container.scrollTo({ top: currentScrollHeight, behavior: 'smooth' });
    }

    previousScrollHeightRef.current = currentScrollHeight;
  }, [logs]);

  const handleScroll = () => {
    if (!containerRef.current || !onScroll) return;
    const { scrollTop, scrollHeight, clientHeight } = containerRef.current;
    previousScrollHeightRef.current = scrollHeight;
    onScroll({ scrollTop, scrollHeight, clientHeight });
  };

  const showLogs =
    logs.length > 0
      ? logs
      : [
          {
            line: emptyMessage,
            timestamp: new Date().toISOString(),
            instance: 'Hatchet',
          },
        ];

  return (
    <div
      ref={containerRef}
      onScroll={handleScroll}
      className="scrollbar-thin scrollbar-track-muted scrollbar-thumb-muted-foreground mx-auto max-h-[25rem] min-h-[25rem] w-full overflow-y-auto rounded-md bg-muted p-6 font-mono text-xs text-indigo-300"
    >
      {showLogs.map((log, i) => {
        const sanitizedHtml = DOMPurify.sanitize(
          convert.toHtml(log.line || ''),
          { USE_PROFILES: { html: true } },
        );

        return (
          <p
            key={`${log.timestamp}-${i}`}
            className="overflow-x-hidden whitespace-pre-wrap break-all pb-2"
          >
            {log.timestamp && (
              <span className="mr-2 text-gray-500">
                {new Date(log.timestamp)
                  .toLocaleString('sv', DATE_FORMAT_OPTIONS)
                  .replace(',', '.')
                  .replace(' ', 'T')}
              </span>
            )}
            {log.instance && (
              <span className="mr-2 text-foreground dark:text-white">
                {log.instance}
              </span>
            )}
            <span dangerouslySetInnerHTML={{ __html: sanitizedHtml }} />
          </p>
        );
      })}
    </div>
  );
}
