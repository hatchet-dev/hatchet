import { AnsiLine } from './ansi-line';
import {
  LogLine,
  V1LogLineLevelIncludingEvictionNotice,
} from './log-search/use-logs';
import RelativeDate from '@/components/v1/molecules/relative-date';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/v1/ui/popover';
import {
  PortalTooltip,
  PortalTooltipContent,
  PortalTooltipProvider,
  PortalTooltipTrigger,
} from '@/components/v1/ui/portal-tooltip';
import { V1LogLineLevel, V1TaskStatus } from '@/lib/api';
import { cn } from '@/lib/utils';
import { Link } from '@tanstack/react-router';
import { ExternalLink, XCircle } from 'lucide-react';
import { useMemo, useCallback, useRef, useState } from 'react';

const DATE_FORMAT_OPTIONS: Intl.DateTimeFormatOptions = {
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false,
};

const levelToStyle = (
  level: V1LogLineLevelIncludingEvictionNotice,
): { bg: string; text: string; dot: string; content: string } => {
  if (level == 'EVICTION_NOTICE') {
    return {
      bg: 'bg-indigo-500/20',
      text: 'text-indigo-800 dark:text-indigo-300',
      dot: 'bg-indigo-500',
      content: 'info',
    };
  } else if (level == 'RESTORE_NOTICE') {
    return {
      bg: 'bg-indigo-500/20',
      text: 'text-indigo-800 dark:text-indigo-300',
      dot: 'bg-indigo-500',
      content: 'info',
    };
  } else {
    switch (level) {
      case V1LogLineLevel.ERROR:
        return {
          bg: 'bg-red-500/10',
          text: 'text-red-600 dark:text-red-400',
          dot: 'bg-red-500',
          content: 'error',
        };
      case V1LogLineLevel.WARN:
        return {
          bg: 'bg-yellow-500/10',
          text: 'text-yellow-600 dark:text-yellow-400',
          dot: 'bg-yellow-500',
          content: 'warn',
        };
      case V1LogLineLevel.INFO:
        return {
          bg: 'bg-green-500/10',
          text: 'text-green-600 dark:text-green-400',
          dot: 'bg-green-500',
          content: 'info',
        };
      case V1LogLineLevel.DEBUG:
        return {
          bg: 'bg-gray-500/10',
          text: 'text-gray-500 dark:text-gray-400',
          dot: 'bg-gray-500',
          content: 'debug',
        };
      default:
        const exhaustiveCheck: never = level;
        throw new Error(`Unhandled log level: ${exhaustiveCheck}`);
    }
  }
};

const formatTimestamp = (timestamp: string): string => {
  return new Date(timestamp)
    .toLocaleString('sv', DATE_FORMAT_OPTIONS)
    .replace(',', '.');
};

interface LogViewerProps {
  logs: LogLine[];
  onScrollToBottom?: () => void;
  onScrollToTop?: () => void;
  onAtTopChange?: (atTop: boolean) => void;
  isLoading?: boolean;
  taskStatus?: V1TaskStatus;
  onViewRun?: (taskExternalId: string) => void;
  emptyMessage?: string;
  showAttempt?: boolean;
  showTaskName?: boolean;
}

function getEmptyStateMessage(
  taskStatus?: V1TaskStatus,
  emptyMessage?: string,
): string {
  if (emptyMessage) {
    return emptyMessage;
  }
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

const LevelBadge = ({
  level,
}: {
  level: V1LogLineLevelIncludingEvictionNotice;
}) => {
  const style = levelToStyle(level);

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 rounded-md px-1.5 py-0.5 text-[11px] font-medium leading-none uppercase tracking-wide',
        style.bg,
        style.text,
      )}
    >
      <span className={cn('size-1.5 rounded-full', style.dot)} />
      {style.content}
    </span>
  );
};

function isNonEmpty<TValue>(value: TValue | null | undefined): value is TValue {
  return value !== null && value !== undefined;
}

function ErrorPopover({ error }: { error: string }) {
  return (
    <Popover>
      <PopoverTrigger
        onClick={(e) => e.stopPropagation()}
        className="ml-2 shrink-0 inline-flex items-center rounded border border-red-500/30 px-1.5 py-0.5 font-mono text-[10px] text-red-500 dark:text-red-400 hover:border-red-500/60 hover:bg-red-500/5 transition-colors cursor-pointer"
      >
        View Error
      </PopoverTrigger>
      <PopoverContent
        className="w-[300px] max-w-[90vw] border-border bg-popover p-0 shadow-lg sm:w-[400px] md:w-[500px] lg:w-[600px]"
        align="start"
      >
        <div className="w-[300px] max-w-[90vw] p-4 sm:w-[400px] md:w-[500px] lg:w-[600px]">
          <div className="space-y-3">
            <div className="flex items-center gap-2 border-b border-border pb-2">
              <XCircle className="h-5 w-5 text-destructive" />
              <h3 className="font-medium text-foreground">Error Details</h3>
            </div>
            <div className="h-[400px] overflow-hidden rounded-md border border-border bg-muted/50">
              <div className="scrollbar-thin scrollbar-track-muted scrollbar-thumb-muted-foreground h-full overflow-x-hidden overflow-y-scroll p-4 font-mono text-sm text-foreground">
                <pre className="min-h-[500px] whitespace-pre-wrap break-words">
                  {error || 'No error message found'}
                </pre>
              </div>
            </div>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}

export function LogViewer({
  logs,
  onScrollToBottom,
  onScrollToTop,
  onAtTopChange,
  isLoading,
  taskStatus,
  onViewRun,
  emptyMessage,
  showAttempt = true,
  showTaskName = false,
}: LogViewerProps) {
  const [showRelativeTime, setShowRelativeTime] = useState(true);
  const scrollRef = useRef<HTMLDivElement>(null);
  const lastScrollTopRef = useRef(0);
  const wasAtTopRef = useRef(true);
  const wasInTopRegionRef = useRef(false);
  const wasInBottomRegionRef = useRef(false);
  const [selectedLogIndex, setSelectedLogIndex] = useState<number>();

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
      if (showAttempt && log.attempt !== undefined) {
        hasAttempt = true;
      }
      if (hasInstance && hasAttempt) {
        break;
      }
    }
    return { hasInstance, hasAttempt };
  }, [sortedLogs, showAttempt]);

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

  const heightClass = 'flex-1 min-h-0';

  // Build dynamic grid-template-columns
  const gridCols = [
    'auto', // timestamp
    '72px', // level
    hasInstance && 'minmax(100px, 200px)',
    hasAttempt && 'auto',
    showTaskName && 'minmax(100px, 200px)',
    'minmax(0, 1fr)', // message
  ]
    .filter(Boolean)
    .join(' ');

  if (isLoading) {
    return (
      <div
        className={cn(
          heightClass,
          'rounded-lg border bg-background flex items-center justify-center',
        )}
      >
        <span className="text-sm text-muted-foreground">Loading logs...</span>
      </div>
    );
  }

  const isEmpty = logs.length === 0;
  if (isEmpty) {
    return (
      <div
        className={cn(
          heightClass,
          'rounded-lg border bg-background flex items-center justify-center',
        )}
      >
        <span className="text-sm text-muted-foreground">
          {getEmptyStateMessage(taskStatus, emptyMessage)}
        </span>
      </div>
    );
  }

  // Column count for subgrid span
  const colCount =
    3 + (hasInstance ? 1 : 0) + (hasAttempt ? 1 : 0) + (showTaskName ? 1 : 0);

  return (
    <div className="relative rounded-lg border bg-background overflow-hidden flex flex-col flex-1 min-h-0">
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
        className={cn('grid overflow-y-auto', heightClass)}
        style={{
          gridTemplateColumns: gridCols,
          alignContent: 'start',
        }}
        ref={scrollRef}
        onScroll={handleScroll}
      >
        {/* Header row */}
        <div
          className="col-span-full grid grid-cols-subgrid sticky top-0 z-10 bg-gradient-to-b from-background/95 to-background/0 from-75%"
          style={{ gridColumn: `1 / span ${colCount}` }}
        >
          <div
            className="px-2 py-3 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground cursor-pointer select-none hover:text-foreground transition-colors"
            onClick={() => setShowRelativeTime((prev) => !prev)}
          >
            Timestamp
          </div>
          <div className="px-2 py-3 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
            Level
          </div>
          {hasInstance && (
            <div className="px-2 py-3 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
              Instance
            </div>
          )}
          {hasAttempt && (
            <div className="px-2 py-3 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
              Attempt
            </div>
          )}
          {showTaskName && (
            <div className="px-2 py-3 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
              Task
            </div>
          )}
          <div className="px-2 py-3 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
            Message
          </div>
        </div>

        {/* Data rows */}
        {sortedLogs
          .filter((log) => isNonEmpty(log.line))
          .map((log, ix) => (
            <div
              key={`${log.timestamp}-${log.instance ?? ''}-${log.attempt ?? ''}`}
              className="col-span-full items-baseline grid grid-cols-subgrid border-b border-border/40 hover:bg-muted/30 transition-colors group"
              style={{ gridColumn: `1 / span ${colCount}` }}
            >
              <div className="px-3 py-1.5 font-mono text-xs text-muted-foreground whitespace-nowrap tabular-nums">
                {log.timestamp ? (
                  showRelativeTime ? (
                    <RelativeDate date={log.timestamp} />
                  ) : (
                    formatTimestamp(log.timestamp)
                  )
                ) : (
                  '—'
                )}
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
              {showTaskName && (
                <div
                  className={cn(
                    'px-3 py-1.5 font-mono text-xs text-muted-foreground truncate',
                    onViewRun &&
                      log.taskExternalId &&
                      'cursor-pointer hover:text-foreground hover:underline',
                  )}
                  onClick={
                    onViewRun && log.taskExternalId
                      ? () => onViewRun(log.taskExternalId!)
                      : undefined
                  }
                >
                  {log.taskDisplayName || '—'}
                </div>
              )}
              <div
                className={cn(
                  'px-3 py-1.5 font-mono text-xs text-foreground flex flex-row items-baseline gap-x-1 min-w-0',
                )}
                onClick={() => {
                  setSelectedLogIndex((prev) => (prev === ix ? undefined : ix));
                }}
                onMouseEnter={(e) => {
                  const el = e.currentTarget;
                  if (el.scrollWidth > el.clientWidth) {
                    el.style.cursor = 'pointer';
                  } else {
                    el.style.cursor = 'default';
                  }
                }}
              >
                <span
                  className={cn(
                    'min-w-0',
                    selectedLogIndex === ix
                      ? 'whitespace-normal break-words'
                      : 'truncate',
                    log.linkTo &&
                      selectedLogIndex !== ix &&
                      'max-w-[calc(100%-1rem)]',
                  )}
                >
                  {/* fixme: figure out how to use the type guard properly here */}
                  <AnsiLine text={log.line as string} />
                </span>
                {log.error && <ErrorPopover error={log.error} />}
                {log.linkTo && (
                  <PortalTooltipProvider>
                    <PortalTooltip>
                      <PortalTooltipTrigger asChild>
                        <Link
                          to={log.linkTo.destination}
                          params={log.linkTo.params as Record<string, string>}
                          className="ml-1 shrink-0 text-muted-foreground hover:text-foreground transition-colors"
                          onClick={(e) => {
                            e.stopPropagation();
                          }}
                        >
                          <ExternalLink className="size-3" />
                        </Link>
                      </PortalTooltipTrigger>
                      <PortalTooltipContent>
                        {log.linkTo.hoverText}
                      </PortalTooltipContent>
                    </PortalTooltip>
                  </PortalTooltipProvider>
                )}
              </div>
            </div>
          ))}
      </div>
    </div>
  );
}
