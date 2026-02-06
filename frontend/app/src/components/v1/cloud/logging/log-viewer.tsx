import { LogLine } from './log-search/use-logs';
import { V1TaskStatus } from '@/lib/api';
import { cn } from '@/lib/utils';
import { useMemo, useCallback, useRef } from 'react';

const DATE_FORMAT_OPTIONS: Intl.DateTimeFormatOptions = {
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false,
};

const LEVEL_STYLES: Record<string, { bg: string; text: string; dot: string }> =
  {
    error: {
      bg: 'bg-red-500/10',
      text: 'text-red-600 dark:text-red-400',
      dot: 'bg-red-500',
    },
    warn: {
      bg: 'bg-yellow-500/10',
      text: 'text-yellow-600 dark:text-yellow-400',
      dot: 'bg-yellow-500',
    },
    info: {
      bg: 'bg-blue-500/10',
      text: 'text-blue-600 dark:text-blue-400',
      dot: 'bg-blue-500',
    },
    debug: {
      bg: 'bg-gray-500/10',
      text: 'text-gray-500 dark:text-gray-400',
      dot: 'bg-gray-500',
    },
  };

const formatTimestamp = (timestamp: string): string => {
  return new Date(timestamp)
    .toLocaleString('sv', DATE_FORMAT_OPTIONS)
    .replace(',', '.');
};

export interface LogViewerProps {
  logs: LogLine[];
  onScrollToBottom?: () => void;
  onScrollToTop?: () => void;
  onAtTopChange?: (atTop: boolean) => void;
  isLoading?: boolean;
  taskStatus?: V1TaskStatus;
}

function getEmptyStateMessage(taskStatus?: V1TaskStatus): string {
  switch (taskStatus) {
    case V1TaskStatus.COMPLETED:
      return 'Task completed with no logs.';
    case V1TaskStatus.FAILED:
      return 'Task failed with no logs.';
    case V1TaskStatus.CANCELLED:
      return 'Task was cancelled with no logs.';
    case V1TaskStatus.RUNNING:
    case V1TaskStatus.QUEUED:
      return 'Waiting for logs...';
    default:
      return 'No logs available.';
  }
}

const LevelBadge = ({ level }: { level: string }) => {
  const normalized = level.toLowerCase();
  const style = LEVEL_STYLES[normalized] ?? LEVEL_STYLES.debug;

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 rounded-md px-1.5 py-0.5 text-[11px] font-medium leading-none uppercase tracking-wide',
        style.bg,
        style.text,
      )}
    >
      <span className={cn('size-1.5 rounded-full', style.dot)} />
      {normalized}
    </span>
  );
};

export function LogViewer({
  logs,
  onScrollToBottom,
  onScrollToTop,
  onAtTopChange,
  isLoading,
  taskStatus,
}: LogViewerProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const lastScrollTopRef = useRef(0);
  const wasAtTopRef = useRef(true);
  const wasInTopRegionRef = useRef(false);
  const wasInBottomRegionRef = useRef(false);

  const sortedLogs = useMemo(() => {
    if (logs.length === 0) {
      return [];
    }

    return [...logs].sort((a, b) => {
      if (!a.timestamp || !b.timestamp) {
        return 0;
      }
      return new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime();
    });
  }, [logs]);

  const { hasInstance, hasAttempt } = useMemo(() => {
    let hasInstance = false;
    let hasAttempt = false;
    for (const log of sortedLogs) {
      if (log.instance) {
        hasInstance = true;
      }
      if (log.attempt !== undefined) {
        hasAttempt = true;
      }
      if (hasInstance && hasAttempt) {
        break;
      }
    }
    return { hasInstance, hasAttempt };
  }, [sortedLogs]);

  const handleScroll = useCallback(() => {
    const el = scrollRef.current;
    if (!el) {
      return;
    }

    const { scrollTop, scrollHeight, clientHeight } = el;
    const scrollableHeight = scrollHeight - clientHeight;
    if (scrollableHeight <= 0) {
      return;
    }

    const scrollPercentage = scrollTop / scrollableHeight;
    const isScrollingUp = scrollTop < lastScrollTopRef.current;
    const isScrollingDown = scrollTop > lastScrollTopRef.current;

    const isAtTop = scrollPercentage < 0.05;
    if (onAtTopChange && isAtTop !== wasAtTopRef.current) {
      wasAtTopRef.current = isAtTop;
      onAtTopChange(isAtTop);
    }

    const isInTopRegion = isScrollingUp && scrollPercentage < 0.3;
    if (isInTopRegion && !wasInTopRegionRef.current && onScrollToTop) {
      onScrollToTop();
    }
    wasInTopRegionRef.current = isInTopRegion;

    const isInBottomRegion = isScrollingDown && scrollPercentage > 0.7;
    if (isInBottomRegion && !wasInBottomRegionRef.current && onScrollToBottom) {
      onScrollToBottom();
    }
    wasInBottomRegionRef.current = isInBottomRegion;

    lastScrollTopRef.current = scrollTop;
  }, [onScrollToTop, onScrollToBottom, onAtTopChange]);

  const isRunning = taskStatus === V1TaskStatus.RUNNING;

  // Build dynamic grid-template-columns
  const gridCols = [
    '160px', // timestamp
    '72px', // level
    hasInstance && 'minmax(100px, 200px)',
    hasAttempt && '60px',
    'minmax(0, 1fr)', // message
  ]
    .filter(Boolean)
    .join(' ');

  if (isLoading) {
    return (
      <div className="max-h-[25rem] min-h-[25rem] rounded-lg border bg-background flex items-center justify-center">
        <span className="text-sm text-muted-foreground">Loading logs...</span>
      </div>
    );
  }

  const isEmpty = logs.length === 0;
  if (isEmpty && taskStatus !== undefined) {
    return (
      <div className="max-h-[25rem] min-h-[25rem] rounded-lg border bg-background flex items-center justify-center">
        <span className="text-sm text-muted-foreground">
          {getEmptyStateMessage(taskStatus)}
        </span>
      </div>
    );
  }

  // Column count for subgrid span
  const colCount = 3 + (hasInstance ? 1 : 0) + (hasAttempt ? 1 : 0);

  return (
    <div className="relative rounded-lg border bg-background overflow-hidden">
      {isRunning && (
        <div className="absolute top-2 right-4 z-20 flex items-center gap-2 text-xs text-muted-foreground bg-background/80 px-2 py-1 rounded-md">
          <span className="relative flex h-2 w-2">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75" />
            <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500" />
          </span>
          <span>Live</span>
        </div>
      )}

      <div
        className="grid max-h-[25rem] min-h-[25rem] overflow-y-auto"
        style={{
          gridTemplateColumns: gridCols,
          alignContent: 'start',
        }}
        ref={scrollRef}
        onScroll={handleScroll}
      >
        {/* Header row */}
        <div
          className="col-span-full grid grid-cols-subgrid sticky top-0 z-10 bg-muted/50 backdrop-blur-sm border-b"
          style={{ gridColumn: `1 / span ${colCount}` }}
        >
          <div className="px-3 py-2 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
            Timestamp
          </div>
          <div className="px-3 py-2 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
            Level
          </div>
          {hasInstance && (
            <div className="px-3 py-2 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
              Instance
            </div>
          )}
          {hasAttempt && (
            <div className="px-3 py-2 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
              Attempt
            </div>
          )}
          <div className="px-3 py-2 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
            Message
          </div>
        </div>

        {/* Data rows */}
        {sortedLogs.map((log, idx) => (
          <div
            key={`${log.timestamp}-${idx}`}
            className="col-span-full items-baseline grid grid-cols-subgrid border-b border-border/40 hover:bg-muted/30 transition-colors group"
            style={{ gridColumn: `1 / span ${colCount}` }}
          >
            <div className="px-3 py-1.5 font-mono text-xs text-muted-foreground whitespace-nowrap tabular-nums">
              {log.timestamp ? formatTimestamp(log.timestamp) : '—'}
            </div>
            <div className="px-3 py-1.5 flex items-center">
              {log.level ? (
                <LevelBadge level={log.level} />
              ) : (
                <span className="text-xs text-muted-foreground/50">—</span>
              )}
            </div>
            {hasInstance && (
              <div className="px-3 py-1.5 font-mono text-xs text-muted-foreground truncate">
                {log.instance || '—'}
              </div>
            )}
            {hasAttempt && (
              <div className="px-3 py-1.5 font-mono text-xs text-muted-foreground text-center tabular-nums">
                {log.attempt ?? '—'}
              </div>
            )}
            <div className="px-3 py-1.5 font-mono text-xs text-foreground truncate group-hover:whitespace-normal group-hover:break-words">
              {log.line || ''}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
