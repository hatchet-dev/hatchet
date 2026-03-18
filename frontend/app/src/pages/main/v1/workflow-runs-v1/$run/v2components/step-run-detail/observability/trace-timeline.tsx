import { findTimeRange } from '@/components/v1/agent-prism/agent-prism-data';
import type { OtelSpanTree } from '@/components/v1/agent-prism/span-tree-type';
import { OtelStatusCode } from '@/lib/api/generated/data-contracts';
import { cn } from '@/lib/utils';
import { ChevronRight, ChevronDown } from 'lucide-react';
import { useMemo, useState, useCallback, useRef, type MouseEvent } from 'react';
import { createPortal } from 'react-dom';

const ROW_HEIGHT = 40;
export const LABEL_WIDTH = 320;
const CONNECTOR_WIDTH = 12;
const CONNECTOR_GAP = 8;

type FlatSpanRow = {
  span: OtelSpanTree;
  depth: number;
  isLastChild: boolean;
  connectorFlags: boolean[];
  hasChildren: boolean;
  isExpanded: boolean;
};

function flattenTree(
  trees: OtelSpanTree[],
  expandedIds: Set<string>,
  depth = 0,
  connectorFlags: boolean[] = [],
): FlatSpanRow[] {
  const rows: FlatSpanRow[] = [];

  trees.forEach((tree, idx) => {
    const isLast = idx === trees.length - 1;
    const hasChildren = tree.children.length > 0;
    const isExpanded = expandedIds.has(tree.spanId) && hasChildren;

    rows.push({
      span: tree,
      depth,
      isLastChild: isLast,
      connectorFlags: [...connectorFlags],
      hasChildren,
      isExpanded,
    });

    if (isExpanded) {
      rows.push(
        ...flattenTree(tree.children, expandedIds, depth + 1, [
          ...connectorFlags,
          !isLast,
        ]),
      );
    }
  });

  return rows;
}

function computeTimeTicks(totalDurationMs: number): {
  ticks: number[];
  maxTick: number;
} {
  if (totalDurationMs <= 0) {
    return { ticks: [0], maxTick: 0 };
  }

  const niceIntervals = [
    1, 2, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000, 10000, 20000, 30000,
    60000, 120000, 300000, 600000,
  ];
  const targetTicks = 5;
  const rawInterval = totalDurationMs / targetTicks;

  let interval = niceIntervals[niceIntervals.length - 1];
  for (const c of niceIntervals) {
    if (c >= rawInterval) {
      interval = c;
      break;
    }
  }

  const ticks: number[] = [];
  for (let t = 0; t <= totalDurationMs + interval * 0.5; t += interval) {
    ticks.push(t);
    if (ticks.length > 20) {
      break;
    }
  }

  return { ticks, maxTick: ticks[ticks.length - 1] || totalDurationMs };
}

function formatTimeLabel(ms: number): string {
  if (ms === 0) {
    return '0s';
  }
  if (ms < 1000) {
    return `${Math.round(ms)}ms`;
  }
  if (ms < 60000) {
    const s = ms / 1000;
    return Number.isInteger(s) ? `${s}s` : `${s.toFixed(1)}s`;
  }
  const m = Math.floor(ms / 60000);
  const s = Math.floor((ms % 60000) / 1000);
  return s > 0 ? `${m}m${s}s` : `${m}m`;
}

function formatDurationShort(ms: number): string {
  if (ms < 1) {
    return '<1ms';
  }
  if (ms < 1000) {
    return `${ms.toFixed(ms < 10 ? 2 : 1)}ms`;
  }
  if (ms < 60000) {
    return `${(ms / 1000).toFixed(2)}s`;
  }
  const m = Math.floor(ms / 60000);
  const s = ((ms % 60000) / 1000).toFixed(1);
  return `${m}m ${s}s`;
}

function formatTimestamp(iso: string): string {
  const d = new Date(iso);
  const base = d.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
    second: '2-digit',
    hour12: true,
  });
  const ms = String(d.getMilliseconds()).padStart(3, '0');
  return `${base}.${ms}`;
}

function statusLabel(code: string): string {
  switch (code) {
    case OtelStatusCode.OK:
      return 'OK';
    case OtelStatusCode.ERROR:
      return 'Error';
    default:
      return 'Unset';
  }
}

const barColorsByStatus: Record<string, string> = {
  [OtelStatusCode.OK]: 'bg-success',
  [OtelStatusCode.UNSET]: 'bg-success',
  [OtelStatusCode.ERROR]: 'bg-danger',
};

function getDisplayName(span: OtelSpanTree): string {
  if (!span.spanName.startsWith('hatchet.')) {
    return span.spanName;
  }
  if (span.spanAttributes?.['hatchet.step_name']) {
    return span.spanAttributes['hatchet.step_name'];
  }
  if (span.spanAttributes?.['hatchet.workflow_name']) {
    return span.spanAttributes['hatchet.workflow_name'];
  }
  const actionId = span.spanAttributes?.['hatchet.action_id'];
  if (actionId?.includes(':')) {
    return actionId.split(':')[0];
  }
  return span.spanName;
}

function getBarColor(span: OtelSpanTree): string {
  if (span.statusCode === OtelStatusCode.ERROR) {
    return 'bg-danger';
  }
  if (barColorsByStatus[span.statusCode]) {
    return barColorsByStatus[span.statusCode];
  }
  return 'bg-success';
}

function getDotColor(span: OtelSpanTree): string {
  if (span.statusCode === OtelStatusCode.ERROR) {
    return 'bg-danger';
  }
  return 'bg-success';
}

function SpanTooltip({
  row,
  style,
}: {
  row: FlatSpanRow;
  style: React.CSSProperties;
}) {
  const durationMs = row.span.durationNs / 1_000_000;
  const displayName = getDisplayName(row.span);
  const status = statusLabel(row.span.statusCode);
  const started = formatTimestamp(row.span.createdAt);

  return (
    <div
      className="pointer-events-none z-50 overflow-hidden rounded-lg border border-border bg-popover shadow-lg"
      style={{ maxWidth: 420, ...style }}
    >
      <div className="border-b border-border px-3 py-2">
        <div className="font-mono text-sm font-medium text-foreground">
          {displayName}
        </div>
        {displayName !== row.span.spanName && (
          <div className="mt-0.5 truncate font-mono text-xs text-muted-foreground">
            {row.span.spanName}
          </div>
        )}
      </div>

      <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5 px-3 py-2 text-xs">
        <span className="text-muted-foreground">Duration</span>
        <span className="font-mono font-medium text-foreground">
          {formatDurationShort(durationMs)}
        </span>

        <span className="text-muted-foreground">Status</span>
        <span className="flex items-center gap-1.5">
          <span
            className={cn(
              'size-1.5 shrink-0 rounded-full',
              getDotColor(row.span),
            )}
          />
          <span className="font-mono text-foreground">{status}</span>
        </span>

        <span className="text-muted-foreground">Started</span>
        <span className="font-mono text-foreground">{started}</span>
      </div>
    </div>
  );
}

export type VisibleRange = { startPct: number; endPct: number };

interface TraceTimelineProps {
  spanTrees: OtelSpanTree[];
  expandedSpanIds: string[];
  onExpandChange: (ids: string[]) => void;
  selectedSpan?: OtelSpanTree;
  onSpanSelect?: (span: OtelSpanTree) => void;
  visibleRange?: VisibleRange;
}

export function TraceTimeline({
  spanTrees,
  expandedSpanIds,
  onExpandChange,
  selectedSpan,
  onSpanSelect,
  visibleRange,
}: TraceTimelineProps) {
  const [hoveredSpanId, setHoveredSpanId] = useState<string | null>(null);
  const [tooltipPos, setTooltipPos] = useState<{
    x: number;
    y: number;
  } | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  const expandedSet = useMemo(
    () => new Set(expandedSpanIds),
    [expandedSpanIds],
  );

  const flatRows = useMemo(
    () => flattenTree(spanTrees, expandedSet),
    [spanTrees, expandedSet],
  );

  const { visMinStart, ticks, timelineMaxMs } = useMemo(() => {
    const { minStart, maxEnd } = findTimeRange(spanTrees);
    const totalDurationMs = maxEnd - minStart;

    const isZoomed =
      visibleRange &&
      (visibleRange.startPct > 0.001 || visibleRange.endPct < 0.999);

    if (isZoomed) {
      const visStartMs = minStart + totalDurationMs * visibleRange.startPct;
      const visEndMs = minStart + totalDurationMs * visibleRange.endPct;
      const visDurationMs = visEndMs - visStartMs;
      const { ticks, maxTick } = computeTimeTicks(visDurationMs);
      return {
        visMinStart: visStartMs,
        ticks,
        timelineMaxMs: Math.max(maxTick, visDurationMs),
      };
    }

    const { ticks, maxTick } = computeTimeTicks(totalDurationMs);
    const timelineMaxMs = Math.max(maxTick, totalDurationMs);
    return { visMinStart: minStart, ticks, timelineMaxMs };
  }, [spanTrees, visibleRange]);

  const toggleExpand = useCallback(
    (spanId: string) => {
      if (expandedSpanIds.includes(spanId)) {
        onExpandChange(expandedSpanIds.filter((id) => id !== spanId));
      } else {
        onExpandChange([...expandedSpanIds, spanId]);
      }
    },
    [expandedSpanIds, onExpandChange],
  );

  const handleBarHover = useCallback(
    (spanId: string | null, event?: MouseEvent) => {
      setHoveredSpanId(spanId);
      if (event) {
        setTooltipPos({ x: event.clientX, y: event.clientY });
      } else {
        setTooltipPos(null);
      }
    },
    [],
  );

  const handleBarMouseMove = useCallback((e: MouseEvent) => {
    setTooltipPos({ x: e.clientX, y: e.clientY });
  }, []);

  const hoveredRow = hoveredSpanId
    ? flatRows.find((r) => r.span.spanId === hoveredSpanId)
    : null;

  const gridHeight = flatRows.length * ROW_HEIGHT;

  return (
    <div className="relative flex min-w-0 overflow-hidden" ref={containerRef}>
      {/* Left panel: span labels */}
      <div
        className="flex shrink-0 flex-col overflow-hidden pt-6"
        style={{ width: LABEL_WIDTH }}
      >
        {flatRows.map((row) => {
          const isSelected = selectedSpan?.spanId === row.span.spanId;

          return (
            <div
              key={row.span.spanId}
              className={cn(
                'flex shrink-0 cursor-pointer items-center rounded-l px-2',
                isSelected && 'bg-[rgba(0,92,158,0.02)]',
              )}
              style={{ height: ROW_HEIGHT }}
              onClick={() => {
                if (row.hasChildren) {
                  toggleExpand(row.span.spanId);
                }
                onSpanSelect?.(row.span);
              }}
            >
              {/* Tree connector lines */}
              {Array.from({ length: row.depth }).map((_, i) => {
                const isOwnLevel = i === row.depth - 1;
                const showLine = isOwnLevel
                  ? row.connectorFlags[i] || !row.isLastChild
                  : row.connectorFlags[i];
                return (
                  <div
                    key={i}
                    className="flex shrink-0 items-center justify-center"
                    style={{ width: CONNECTOR_WIDTH, height: ROW_HEIGHT }}
                  >
                    {showLine && <div className="h-full w-px bg-border" />}
                  </div>
                );
              })}

              {/* Expand/collapse chevron */}
              {row.hasChildren ? (
                <button
                  className="flex shrink-0 items-center justify-center text-muted-foreground transition-colors hover:text-foreground"
                  style={{ width: CONNECTOR_WIDTH + CONNECTOR_GAP }}
                  onClick={(e) => {
                    e.stopPropagation();
                    toggleExpand(row.span.spanId);
                  }}
                >
                  {row.isExpanded ? (
                    <ChevronDown className="size-3" />
                  ) : (
                    <ChevronRight className="size-3" />
                  )}
                </button>
              ) : row.depth > 0 ? (
                <div style={{ width: CONNECTOR_GAP }} className="shrink-0" />
              ) : null}

              {/* Span name */}
              <span
                className={cn(
                  'truncate text-sm leading-tight',
                  row.depth === 0 ? 'text-foreground' : 'text-muted-foreground',
                )}
                title={getDisplayName(row.span)}
              >
                {getDisplayName(row.span)}
              </span>
            </div>
          );
        })}
      </div>

      {/* Right panel: timeline */}
      <div className="flex min-w-0 flex-1 flex-col overflow-hidden pr-10">
        {/* Time axis labels */}
        <div className="relative h-6 shrink-0">
          {ticks.map((t, i) => {
            const isLast = i === ticks.length - 1;
            return (
              <div
                key={t}
                className="absolute flex h-full items-center"
                style={{
                  left: `${(t / timelineMaxMs) * 100}%`,
                  transform: isLast ? 'translateX(-100%)' : undefined,
                }}
              >
                <span className="whitespace-nowrap font-mono text-xs uppercase tracking-wider text-muted-foreground">
                  {formatTimeLabel(t)}
                </span>
              </div>
            );
          })}
        </div>

        {/* Timeline bars area */}
        <div className="relative">
          {/* Vertical grid lines */}
          {ticks.map((t) => (
            <div
              key={t}
              className="absolute top-0 w-px bg-border/40"
              style={{
                left: `${(t / timelineMaxMs) * 100}%`,
                height: gridHeight,
              }}
            />
          ))}

          {/* Bar rows */}
          {flatRows.map((row) => {
            const startMs = new Date(row.span.createdAt).getTime();
            const durationMs = row.span.durationNs / 1_000_000;
            const leftPct =
              timelineMaxMs > 0
                ? ((startMs - visMinStart) / timelineMaxMs) * 100
                : 0;
            const widthPct =
              timelineMaxMs > 0 ? (durationMs / timelineMaxMs) * 100 : 0;

            return (
              <div
                key={row.span.spanId}
                className="relative shrink-0"
                style={{ height: ROW_HEIGHT }}
              >
                <div
                  className={cn(
                    'absolute bottom-[10px] top-[10px] cursor-pointer rounded-sm transition-shadow',
                    getBarColor(row.span),
                    hoveredSpanId === row.span.spanId &&
                      'ring-1 ring-foreground/20',
                  )}
                  style={{
                    left: `${leftPct}%`,
                    width: `${Math.max(widthPct, 0.3)}%`,
                    minWidth: 2,
                  }}
                  onMouseEnter={(e) => handleBarHover(row.span.spanId, e)}
                  onMouseMove={handleBarMouseMove}
                  onMouseLeave={() => handleBarHover(null)}
                  onClick={() => onSpanSelect?.(row.span)}
                />
              </div>
            );
          })}
        </div>
      </div>

      {/* Hover tooltip rendered via portal to escape overflow-hidden */}
      {hoveredRow &&
        tooltipPos &&
        createPortal(
          <SpanTooltip
            row={hoveredRow}
            style={{
              position: 'fixed',
              left: Math.min(tooltipPos.x + 12, window.innerWidth - 440),
              top: Math.max(8, tooltipPos.y - 100),
            }}
          />,
          document.body,
        )}
    </div>
  );
}
