import { GroupBar } from './group-bar';
import { SpanBar } from './span-bar';
import {
  SpanTooltip,
  GroupTooltip,
  TOOLTIP_EDGE_LIMIT,
} from './trace-timeline-tooltips';
import {
  ROW_HEIGHT,
  type FlatRow,
  type SpanGroupInfo,
  type VisibleRange,
} from './trace-timeline-utils';
import { useBrushZoom } from './use-brush-zoom';
import { useCursorSync } from './use-cursor-sync';
import type { OtelSpanTree } from '@/components/v1/agent-prism/span-tree-type';
import { memo, useState, useCallback, useRef, type MouseEvent } from 'react';
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
  selectedDescendantIds: Set<string>;
  hoveredRowKey: string | null;
  onRowHover: (key: string | null) => void;
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
  selectedDescendantIds,
  hoveredRowKey,
  onRowHover,
  onSpanSelect,
  onGroupSelect,
  expandOnly,
  onRangeChange,
  externalCursorPct,
  onCursorPctChange,
}: TimelineBarsProps) {
  const barsRef = useRef<HTMLDivElement>(null);

  const timelineValues = {
    visMinStart,
    timelineMaxMs,
    traceMinStart,
    traceTotalMs,
  };

  const { brushRange, onPointerDown, onDoubleClick } = useBrushZoom(
    barsRef,
    timelineValues,
    onRangeChange,
  );

  const { effectiveCursorPct, onMouseMove, onMouseLeave } = useCursorSync(
    barsRef,
    timelineValues,
    externalCursorPct,
    onCursorPctChange,
  );

  const [tooltipPos, setTooltipPos] = useState<{
    x: number;
    y: number;
  } | null>(null);

  const handleBarHover = useCallback(
    (rowKey: string | null, event?: MouseEvent) => {
      onRowHover(rowKey);
      if (event) {
        setTooltipPos({ x: event.clientX, y: event.clientY });
      } else {
        setTooltipPos(null);
      }
    },
    [onRowHover],
  );

  const handleBarMouseMove = useCallback((e: MouseEvent) => {
    setTooltipPos({ x: e.clientX, y: e.clientY });
  }, []);

  const hoveredRow = hoveredRowKey
    ? flatRows.find((r) => r.rowKey === hoveredRowKey)
    : null;

  return (
    <div className="flex min-w-0 flex-1 flex-col overflow-hidden pr-10">
      <div
        className="relative overflow-hidden"
        ref={barsRef}
        style={{ cursor: onRangeChange ? 'crosshair' : undefined }}
        onMouseMove={onMouseMove}
        onMouseLeave={onMouseLeave}
        onPointerDown={onPointerDown}
        onDoubleClick={onDoubleClick}
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
            return (
              <GroupBar
                key={row.rowKey}
                row={row}
                timelineMaxMs={timelineMaxMs}
                visMinStart={visMinStart}
                hasAnyInProgress={hasAnyInProgress}
                hasAnyLiveQueued={hasAnyLiveQueued}
                isSelected={selectedGroupId === row.group.groupId}
                isHovered={hoveredRowKey === row.rowKey}
                onHover={handleBarHover}
                onMouseMove={handleBarMouseMove}
                onGroupSelect={onGroupSelect}
                expandOnly={expandOnly}
              />
            );
          }

          return (
            <SpanBar
              key={row.rowKey}
              row={row}
              now={now}
              isRunning={isRunning}
              timelineMaxMs={timelineMaxMs}
              visMinStart={visMinStart}
              hasAnyInProgress={hasAnyInProgress}
              hasAnyLiveQueued={hasAnyLiveQueued}
              isSelected={selectedSpan?.spanId === row.span.spanId}
              isChildOfSelected={selectedDescendantIds.has(row.span.spanId)}
              isHovered={hoveredRowKey === row.rowKey}
              onHover={handleBarHover}
              onMouseMove={handleBarMouseMove}
              onSpanSelect={onSpanSelect}
              expandOnly={expandOnly}
            />
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
              left: Math.min(
                tooltipPos.x + 12,
                window.innerWidth - TOOLTIP_EDGE_LIMIT,
              ),
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
              left: Math.min(
                tooltipPos.x + 12,
                window.innerWidth - TOOLTIP_EDGE_LIMIT,
              ),
              top: tooltipPos.y + 16,
            }}
          />,
          document.body,
        )}
    </div>
  );
});
