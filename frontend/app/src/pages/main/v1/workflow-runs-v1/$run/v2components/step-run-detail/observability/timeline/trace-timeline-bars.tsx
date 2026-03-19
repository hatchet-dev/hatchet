import { formatDuration } from '../utils/format-utils';
import {
  getBarColor,
  hasErrorInTree,
  isQueuedOnlyRoot,
} from '../utils/span-tree-utils';
import { SpanTooltip, GroupTooltip } from './trace-timeline-tooltips';
import {
  ROW_HEIGHT,
  type FlatRow,
  type SpanGroupInfo,
  type VisibleRange,
} from './trace-timeline-utils';
import type { OtelSpanTree } from '@/components/v1/agent-prism/span-tree-type';
import { cn } from '@/lib/utils';
import {
  memo,
  useMemo,
  useState,
  useCallback,
  useRef,
  type MouseEvent,
} from 'react';
import { createPortal } from 'react-dom';

interface TimelineBarsProps {
  flatRows: FlatRow[];
  ticks: number[];
  timelineMaxMs: number;
  visMinStart: number;
  visOffsetMs: number;
  traceMinStart: number;
  traceTotalMs: number;
  gridHeight: number;
  now: number;
  isRunning?: boolean;
  hasAnyInProgress: boolean;
  hasAnyLiveQueued: boolean;
  selectedSpan?: OtelSpanTree;
  selectedGroupId?: string;
  onSpanSelect?: (span: OtelSpanTree) => void;
  onGroupSelect?: (group: SpanGroupInfo) => void;
  expandOnly: (id: string) => void;
  onRangeChange?: (range: VisibleRange) => void;
  externalCursorPct?: number | null;
  onCursorPctChange?: (pct: number | null) => void;
}

export const TimelineBars = memo(function TimelineBars({
  flatRows,
  ticks,
  timelineMaxMs,
  visMinStart,
  visOffsetMs,
  traceMinStart,
  traceTotalMs,
  gridHeight,
  now,
  isRunning,
  hasAnyInProgress,
  hasAnyLiveQueued,
  selectedSpan,
  selectedGroupId,
  onSpanSelect,
  onGroupSelect,
  expandOnly,
  onRangeChange,
  externalCursorPct,
  onCursorPctChange,
}: TimelineBarsProps) {
  const barsRef = useRef<HTMLDivElement>(null);
  const [hoveredRowKey, setHoveredRowKey] = useState<string | null>(null);
  const [tooltipPos, setTooltipPos] = useState<{
    x: number;
    y: number;
  } | null>(null);
  const [cursorPct, setCursorPct] = useState<number | null>(null);
  const [brushRange, setBrushRange] = useState<{
    lo: number;
    hi: number;
  } | null>(null);

  const handleBarHover = useCallback(
    (rowKey: string | null, event?: MouseEvent) => {
      setHoveredRowKey(rowKey);
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

  const timelineValuesRef = useRef({
    visMinStart: 0,
    visOffsetMs: 0,
    timelineMaxMs: 0,
    traceMinStart: 0,
    traceTotalMs: 0,
  });
  timelineValuesRef.current = {
    visMinStart,
    visOffsetMs,
    timelineMaxMs,
    traceMinStart,
    traceTotalMs,
  };

  const handleBarsPointerDown = useCallback(
    (e: React.PointerEvent) => {
      if (!barsRef.current || !onRangeChange) {
        return;
      }

      const rect = barsRef.current.getBoundingClientRect();
      const startPct = Math.max(
        0,
        Math.min(1, (e.clientX - rect.left) / rect.width),
      );

      const onMove = (ev: PointerEvent) => {
        if (!barsRef.current) {
          return;
        }
        const r = barsRef.current.getBoundingClientRect();
        const pct = Math.max(0, Math.min(1, (ev.clientX - r.left) / r.width));
        const lo = Math.min(startPct, pct);
        const hi = Math.max(startPct, pct);
        if (hi - lo > 0.005) {
          setBrushRange({ lo, hi });
        }
      };

      const onUp = (ev: PointerEvent) => {
        document.removeEventListener('pointermove', onMove);
        document.removeEventListener('pointerup', onUp);

        if (!barsRef.current) {
          setBrushRange(null);
          return;
        }
        const r = barsRef.current.getBoundingClientRect();
        const pct = Math.max(0, Math.min(1, (ev.clientX - r.left) / r.width));
        const lo = Math.min(startPct, pct);
        const hi = Math.max(startPct, pct);

        setBrushRange(null);

        if (hi - lo >= 0.02) {
          const v = timelineValuesRef.current;
          const newStartMs = v.visMinStart + v.timelineMaxMs * lo;
          const newEndMs = v.visMinStart + v.timelineMaxMs * hi;
          onRangeChange({
            startPct: Math.max(
              0,
              (newStartMs - v.traceMinStart) / v.traceTotalMs,
            ),
            endPct: Math.min(1, (newEndMs - v.traceMinStart) / v.traceTotalMs),
          });
        }
      };

      document.addEventListener('pointermove', onMove);
      document.addEventListener('pointerup', onUp);
    },
    [onRangeChange],
  );

  const handleBarsDoubleClick = useCallback(() => {
    onRangeChange?.({ startPct: 0, endPct: 1 });
  }, [onRangeChange]);

  const hoveredRow = hoveredRowKey
    ? flatRows.find((r) => r.rowKey === hoveredRowKey)
    : null;

  const effectiveCursorPct = useMemo(() => {
    if (cursorPct !== null) {
      return cursorPct;
    }
    if (externalCursorPct == null || timelineMaxMs <= 0) {
      return null;
    }
    const timeMs = traceMinStart + externalCursorPct * traceTotalMs;
    const localPct = (timeMs - visMinStart) / timelineMaxMs;
    if (localPct < 0 || localPct > 1) {
      return null;
    }
    return localPct;
  }, [
    cursorPct,
    externalCursorPct,
    visMinStart,
    timelineMaxMs,
    traceMinStart,
    traceTotalMs,
  ]);

  return (
    <div className="flex min-w-0 flex-1 flex-col overflow-hidden pr-10">
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
                {formatDuration(t + visOffsetMs)}
              </span>
            </div>
          );
        })}

        {effectiveCursorPct !== null && !brushRange && (
          <div
            className="pointer-events-none absolute z-10 flex h-full items-center whitespace-nowrap rounded bg-foreground/90 px-1 py-px font-mono text-[10px] leading-tight text-background"
            style={{
              left: `${effectiveCursorPct * 100}%`,
              transform:
                effectiveCursorPct < 0.05
                  ? 'none'
                  : effectiveCursorPct > 0.95
                    ? 'translateX(-100%)'
                    : 'translateX(-50%)',
            }}
          >
            {formatDuration(timelineMaxMs * effectiveCursorPct + visOffsetMs)}
          </div>
        )}

        {brushRange && (
          <div
            className="pointer-events-none absolute z-20 flex h-full items-center"
            style={{
              left: `${brushRange.lo * 100}%`,
              width: `${(brushRange.hi - brushRange.lo) * 100}%`,
            }}
          >
            <span className="shrink-0 whitespace-nowrap rounded bg-foreground/90 px-1 py-px font-mono text-[10px] leading-tight text-background">
              {formatDuration(timelineMaxMs * brushRange.lo + visOffsetMs)}
            </span>
            <div className="flex min-w-1 flex-1 items-center">
              <svg
                width="5"
                height="6"
                viewBox="0 0 5 6"
                className="shrink-0 fill-primary"
              >
                <path d="M5 0L0 3L5 6Z" />
              </svg>
              <div className="h-px flex-1 bg-primary" />
            </div>
            <span className="shrink-0 whitespace-nowrap rounded bg-primary px-1.5 py-0.5 font-mono text-[10px] font-medium leading-tight text-primary-foreground">
              {formatDuration(timelineMaxMs * (brushRange.hi - brushRange.lo))}
            </span>
            <div className="flex min-w-1 flex-1 items-center">
              <div className="h-px flex-1 bg-primary" />
              <svg
                width="5"
                height="6"
                viewBox="0 0 5 6"
                className="shrink-0 fill-primary"
              >
                <path d="M0 0L5 3L0 6Z" />
              </svg>
            </div>
            <span className="shrink-0 whitespace-nowrap rounded bg-foreground/90 px-1 py-px font-mono text-[10px] leading-tight text-background">
              {formatDuration(timelineMaxMs * brushRange.hi + visOffsetMs)}
            </span>
          </div>
        )}
      </div>

      <div
        className="relative"
        ref={barsRef}
        style={{ cursor: onRangeChange ? 'crosshair' : undefined }}
        onMouseMove={(e) => {
          if (!barsRef.current) {
            return;
          }
          const rect = barsRef.current.getBoundingClientRect();
          const localPct = Math.max(
            0,
            Math.min(1, (e.clientX - rect.left) / rect.width),
          );
          setCursorPct(localPct);
          if (onCursorPctChange) {
            const v = timelineValuesRef.current;
            const timeMs = v.visMinStart + localPct * v.timelineMaxMs;
            const fullPct =
              v.traceTotalMs > 0
                ? (timeMs - v.traceMinStart) / v.traceTotalMs
                : 0;
            onCursorPctChange(Math.max(0, Math.min(1, fullPct)));
          }
        }}
        onMouseLeave={() => {
          setCursorPct(null);
          onCursorPctChange?.(null);
        }}
        onPointerDown={handleBarsPointerDown}
        onDoubleClick={handleBarsDoubleClick}
      >
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

        {flatRows.map((row) => {
          if (row.kind === 'show-more') {
            return (
              <div
                key={row.rowKey}
                className="relative shrink-0"
                style={{ height: ROW_HEIGHT }}
              />
            );
          }

          if (row.kind === 'group') {
            const durationMs =
              row.group.latestEndMs - row.group.earliestStartMs;
            const leftPct =
              timelineMaxMs > 0
                ? ((row.group.earliestStartMs - visMinStart) / timelineMaxMs) *
                  100
                : 0;
            const widthPct =
              timelineMaxMs > 0 ? (durationMs / timelineMaxMs) * 100 : 0;
            const isSelected = selectedGroupId === row.group.groupId;
            const hasErrors = row.group.errorCount > 0;

            return (
              <div
                key={row.rowKey}
                className={cn(
                  'relative shrink-0 transition-colors',
                  isSelected && 'bg-primary/8',
                )}
                style={{ height: ROW_HEIGHT }}
              >
                <div
                  className={cn(
                    'absolute bottom-[10px] top-[10px] cursor-pointer rounded-[2px]',
                    !hasAnyInProgress && !hasAnyLiveQueued && 'transition-all',
                    hasErrors ? 'bg-red-500' : 'bg-green-500',
                    isSelected
                      ? 'ring-2 ring-primary ring-offset-1 ring-offset-background'
                      : hoveredRowKey === row.rowKey
                        ? 'ring-1 ring-foreground/20'
                        : '',
                  )}
                  style={{
                    left: `${leftPct}%`,
                    width: `${Math.max(widthPct, 0.3)}%`,
                    minWidth: 2,
                  }}
                  onMouseEnter={(e) => handleBarHover(row.rowKey, e)}
                  onMouseMove={handleBarMouseMove}
                  onMouseLeave={() => handleBarHover(null)}
                  onClick={() => {
                    expandOnly(row.group.groupId);
                    onGroupSelect?.(row.group);
                  }}
                />
              </div>
            );
          }

          const startMs = new Date(row.span.createdAt).getTime();
          const durationMs = row.span.inProgress
            ? Math.max(0, now - startMs)
            : row.span.durationNs / 1_000_000;
          const hideExecutionBar =
            isQueuedOnlyRoot(row.span) && durationMs <= 0;
          const leftPct =
            timelineMaxMs > 0
              ? ((startMs - visMinStart) / timelineMaxMs) * 100
              : 0;
          const widthPct =
            timelineMaxMs > 0 ? (durationMs / timelineMaxMs) * 100 : 0;
          const isSelected = selectedSpan?.spanId === row.span.spanId;
          const isBarDimmed = !row.matchesFilter;

          const q = row.span.queuedPhase;
          let qLeftPct = 0;
          let qWidthPct = 0;
          if (q) {
            const qStartMs = new Date(q.createdAt).getTime();
            const qEndMs =
              isRunning && isQueuedOnlyRoot(row.span) ? now : startMs;
            // FIXME: snapping hides a real gap (typically 0.5–2ms) between queue-end
            // and exec-start. Consider a synthetic "network/dispatch" span to
            // visualize scheduling + worker dispatch latency instead of hiding it.
            const snappedDurMs = qEndMs - qStartMs;
            qLeftPct =
              timelineMaxMs > 0
                ? ((qStartMs - visMinStart) / timelineMaxMs) * 100
                : 0;
            qWidthPct =
              timelineMaxMs > 0 ? (snappedDurMs / timelineMaxMs) * 100 : 0;
          }

          return (
            <div
              key={row.rowKey}
              className={cn(
                'relative shrink-0 transition-colors',
                isSelected && 'bg-primary/8',
                isBarDimmed && 'opacity-40',
              )}
              style={{ height: ROW_HEIGHT }}
            >
              {q && (
                <div
                  className={cn(
                    'absolute bottom-[10px] top-[10px] cursor-pointer rounded-[2px] border border-dashed',
                    !hasAnyInProgress && !hasAnyLiveQueued && 'transition-all',
                    (row.span.inProgress || isQueuedOnlyRoot(row.span)) &&
                      'animate-pulse',
                    row.span.inProgress
                      ? 'border-yellow-500 bg-yellow-500/10'
                      : hasErrorInTree(row.span)
                        ? 'border-red-500 bg-red-500/10'
                        : 'border-green-500 bg-green-500/10',
                  )}
                  style={{
                    left: `${qLeftPct}%`,
                    width: `${Math.max(qWidthPct, 0.3)}%`,
                    minWidth: 2,
                  }}
                  onMouseEnter={(e) => handleBarHover(row.rowKey, e)}
                  onMouseMove={handleBarMouseMove}
                  onMouseLeave={() => handleBarHover(null)}
                  onClick={() => {
                    if (row.hasChildren) {
                      expandOnly(row.rowKey);
                    }
                    onSpanSelect?.(row.span);
                  }}
                />
              )}
              {!hideExecutionBar && (
                <div
                  className={cn(
                    'absolute bottom-[10px] top-[10px] cursor-pointer rounded-[2px]',
                    getBarColor(row.span),
                    !hasAnyInProgress && !hasAnyLiveQueued && 'transition-all',
                    row.span.inProgress && 'animate-pulse',
                    isSelected
                      ? 'ring-2 ring-primary ring-offset-1 ring-offset-background'
                      : hoveredRowKey === row.rowKey
                        ? 'ring-1 ring-foreground/20'
                        : '',
                  )}
                  style={{
                    left: `${leftPct}%`,
                    width: `${Math.max(widthPct, 0.3)}%`,
                    minWidth: 2,
                  }}
                  onMouseEnter={(e) => handleBarHover(row.rowKey, e)}
                  onMouseMove={handleBarMouseMove}
                  onMouseLeave={() => handleBarHover(null)}
                  onClick={() => {
                    if (row.hasChildren) {
                      expandOnly(row.rowKey);
                    }
                    onSpanSelect?.(row.span);
                  }}
                />
              )}
            </div>
          );
        })}

        {effectiveCursorPct !== null && !brushRange && (
          <div
            className="pointer-events-none absolute top-0 z-10 w-px bg-foreground/40"
            style={{
              left: `${effectiveCursorPct * 100}%`,
              height: gridHeight,
            }}
          />
        )}

        {brushRange && (
          <>
            <div
              className="pointer-events-none absolute top-0 z-10 border-x border-primary/30 bg-primary/10"
              style={{
                left: `${brushRange.lo * 100}%`,
                width: `${(brushRange.hi - brushRange.lo) * 100}%`,
                height: gridHeight,
              }}
            />
            <div
              className="pointer-events-none absolute top-0 z-10 w-px bg-foreground/70"
              style={{
                left: `${brushRange.lo * 100}%`,
                height: gridHeight,
              }}
            />
            <div
              className="pointer-events-none absolute top-0 z-10 w-px bg-foreground/70"
              style={{
                left: `${brushRange.hi * 100}%`,
                height: gridHeight,
              }}
            />
          </>
        )}
      </div>

      {hoveredRow &&
        tooltipPos &&
        hoveredRow.kind === 'span' &&
        createPortal(
          <SpanTooltip
            row={hoveredRow}
            now={now}
            style={{
              position: 'fixed',
              left: Math.min(tooltipPos.x + 12, window.innerWidth - 440),
              top: tooltipPos.y + 16,
            }}
          />,
          document.body,
        )}
      {hoveredRow &&
        tooltipPos &&
        hoveredRow.kind === 'group' &&
        createPortal(
          <GroupTooltip
            group={hoveredRow.group}
            style={{
              position: 'fixed',
              left: Math.min(tooltipPos.x + 12, window.innerWidth - 440),
              top: tooltipPos.y + 16,
            }}
          />,
          document.body,
        )}
    </div>
  );
});
