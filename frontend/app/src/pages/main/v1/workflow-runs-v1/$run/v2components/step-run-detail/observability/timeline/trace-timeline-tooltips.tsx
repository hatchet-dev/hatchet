import { formatDuration, formatTimestamp } from '../utils/format-utils';
import {
  getSpanColor,
  getDisplayName,
  hasErrorInTree,
  isEngineSpan,
  isQueuedOnlyRoot,
  statusLabel,
} from '../utils/span-tree-utils';
import type { FlatSpanRow, SpanGroupInfo } from './trace-timeline-utils';
import { OtelStatusCode } from '@/lib/api/generated/data-contracts';
import { cn } from '@/lib/utils';

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

  const displayName = getDisplayName(span);
  const ownStatus = statusLabel(span.statusCode);
  const descendantError =
    span.statusCode !== OtelStatusCode.ERROR && hasErrorInTree(span);
  const started = formatTimestamp(span.createdAt, { ms: true });
  const q = span.queuedPhase;

  let queueMs = 0;
  if (q) {
    const qStartMs = new Date(q.createdAt).getTime();
    queueMs = queuedOnly ? Math.max(0, now - qStartMs) : q.durationNs / 1e6;
  }

  return (
    <div
      className="pointer-events-none z-50 overflow-hidden rounded-lg border border-border bg-popover shadow-lg"
      style={{ maxWidth: 420, ...style }}
    >
      <div className="border-b border-border px-3 py-2">
        <div className="font-mono text-sm font-medium text-foreground">
          {displayName}
        </div>
        {displayName !== span.spanName && (
          <div className="mt-0.5 truncate font-mono text-xs text-muted-foreground">
            {span.spanName}
          </div>
        )}
      </div>

      <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5 px-3 py-2 text-xs">
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
              {isEngineSpan(span) && span.spanName === 'hatchet.engine.queued'
                ? 'Queue Time'
                : 'Duration'}
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
            {queuedOnly
              ? 'Queued'
              : span.inProgress
                ? 'In Progress'
                : descendantError
                  ? 'Error (child)'
                  : ownStatus}
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
      </div>
    </div>
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
    <div
      className="pointer-events-none z-50 overflow-hidden rounded-lg border border-border bg-popover shadow-lg"
      style={{ maxWidth: 420, ...style }}
    >
      <div className="border-b border-border px-3 py-2">
        <div className="font-mono text-sm font-medium text-foreground">
          {group.groupName}
        </div>
        <div className="mt-0.5 font-mono text-xs text-muted-foreground">
          {group.totalCount.toLocaleString()} spans
        </div>
      </div>

      <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5 px-3 py-2 text-xs">
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
      </div>
    </div>
  );
}
