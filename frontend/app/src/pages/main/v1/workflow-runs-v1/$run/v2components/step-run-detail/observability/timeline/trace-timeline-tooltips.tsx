import { formatDuration, formatTimestamp } from '../utils/format-utils';
import {
  getSpanColor,
  effectiveStatusLabel,
  isEngineSpan,
  isQueuedEngineSpan,
  isQueuedOnlyRoot,
} from '../utils/span-tree-utils';
import type { FlatSpanRow, SpanGroupInfo } from './trace-timeline-utils';
import { cn } from '@/lib/utils';

export const TOOLTIP_MAX_WIDTH = 420;
const TOOLTIP_OVERFLOW_BUFFER = 20;
export const TOOLTIP_EDGE_LIMIT = TOOLTIP_MAX_WIDTH + TOOLTIP_OVERFLOW_BUFFER;

function TooltipShell({
  title,
  subtitle,
  style,
  children,
}: {
  title: string;
  subtitle?: string;
  style: React.CSSProperties;
  children: React.ReactNode;
}) {
  return (
    <div
      className="pointer-events-none z-50 overflow-hidden rounded-lg border border-border bg-popover shadow-lg"
      style={{ maxWidth: TOOLTIP_MAX_WIDTH, ...style }}
    >
      <div className="border-b border-border px-3 py-2">
        <div className="font-mono text-sm font-medium text-foreground">
          {title}
        </div>
        {subtitle && (
          <div className="mt-0.5 truncate font-mono text-xs text-muted-foreground">
            {subtitle}
          </div>
        )}
      </div>
      <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5 px-3 py-2 text-xs">
        {children}
      </div>
    </div>
  );
}

export function SpanTooltip({
  row,
  now,
  style,
}: {
  row: FlatSpanRow;
  now: number;
  style: React.CSSProperties;
}) {
  const span = row.span;
  const startMs = new Date(span.createdAt).getTime();
  const queuedOnly = isQueuedOnlyRoot(span) && span.durationNs <= 0;

  const durationMs = span.inProgress
    ? Math.max(0, now - startMs)
    : span.durationNs / 1_000_000;

  const displayName = span.spanName;
  const started = formatTimestamp(span.createdAt, { ms: true });
  const q = span.queuedPhase;

  let queueMs = 0;
  if (q) {
    const qStartMs = new Date(q.createdAt).getTime();
    queueMs = queuedOnly ? Math.max(0, now - qStartMs) : q.durationNs / 1e6;
  }

  return (
    <TooltipShell
      title={displayName}
      subtitle={displayName !== span.spanName ? span.spanName : undefined}
      style={style}
    >
      {q ? (
        <>
          <span className="text-muted-foreground">Queue Time</span>
          <span className="font-mono font-medium text-foreground">
            {formatDuration(queueMs, { precise: true })}
          </span>
          <span className="text-muted-foreground">Execution</span>
          <span className="font-mono font-medium text-foreground">
            {queuedOnly ? '–' : formatDuration(durationMs, { precise: true })}
          </span>
          <span className="text-muted-foreground">Total</span>
          <span className="font-mono font-medium text-foreground">
            {queuedOnly
              ? formatDuration(queueMs, { precise: true })
              : formatDuration(queueMs + durationMs, { precise: true })}
          </span>
        </>
      ) : (
        <>
          <span className="text-muted-foreground">
            {isQueuedEngineSpan(span) ? 'Queue Time' : 'Duration'}
          </span>
          <span className="font-mono font-medium text-foreground">
            {formatDuration(durationMs, { precise: true })}
          </span>
        </>
      )}

      <span className="text-muted-foreground">Status</span>
      <span className="flex items-center gap-1.5">
        <span
          className={cn('size-1.5 shrink-0 rounded-full', getSpanColor(span))}
        />
        <span className="font-mono text-foreground">
          {effectiveStatusLabel(span, queuedOnly)}
        </span>
      </span>

      <span className="text-muted-foreground">Started</span>
      <span className="font-mono text-foreground">{started}</span>

      {isEngineSpan(span) && (
        <>
          <span className="text-muted-foreground">Source</span>
          <span className="font-mono text-foreground">Engine</span>
        </>
      )}
    </TooltipShell>
  );
}

export function GroupTooltip({
  group,
  style,
}: {
  group: SpanGroupInfo;
  style: React.CSSProperties;
}) {
  const durationMs = group.latestEndMs - group.earliestStartMs;

  return (
    <TooltipShell
      title={group.groupName}
      subtitle={`${group.totalCount.toLocaleString()} spans`}
      style={style}
    >
      <span className="text-muted-foreground">Time range</span>
      <span className="font-mono font-medium text-foreground">
        {formatDuration(durationMs, { precise: true })}
      </span>
      {group.errorCount > 0 && (
        <>
          <span className="text-muted-foreground">Errors</span>
          <span className="font-mono font-medium text-red-500">
            {group.errorCount.toLocaleString()}
          </span>
        </>
      )}
    </TooltipShell>
  );
}
