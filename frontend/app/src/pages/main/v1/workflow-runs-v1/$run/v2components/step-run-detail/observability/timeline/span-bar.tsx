import {
  getBarColor,
  hasErrorInTree,
  isQueuedOnlyRoot,
} from '../utils/span-tree-utils';
import {
  ROW_HEIGHT,
  rowHighlightClass,
  type FlatSpanRow,
} from './trace-timeline-utils';
import type { OtelSpanTree } from '@/components/v1/agent-prism/span-tree-type';
import { cn } from '@/lib/utils';
import { memo, type MouseEvent } from 'react';

interface SpanBarProps {
  row: FlatSpanRow;
  now: number;
  isRunning?: boolean;
  timelineMaxMs: number;
  visMinStart: number;
  hasAnyInProgress: boolean;
  hasAnyLiveQueued: boolean;
  isSelected: boolean;
  isChildOfSelected: boolean;
  isHovered: boolean;
  onHover: (rowKey: string | null, event?: MouseEvent) => void;
  onMouseMove: (e: MouseEvent) => void;
  onSpanSelect?: (span: OtelSpanTree) => void;
  expandOnly: (id: string) => void;
}

export const SpanBar = memo(function SpanBar({
  row,
  now,
  isRunning,
  timelineMaxMs,
  visMinStart,
  hasAnyInProgress,
  hasAnyLiveQueued,
  isSelected,
  isChildOfSelected,
  isHovered,
  onHover,
  onMouseMove,
  onSpanSelect,
  expandOnly,
}: SpanBarProps) {
  const startMs = new Date(row.span.createdAt).getTime();
  const durationMs = row.span.inProgress
    ? Math.max(0, now - startMs)
    : row.span.durationNs / 1_000_000;
  const hideExecutionBar = isQueuedOnlyRoot(row.span) && durationMs <= 0;
  const leftPct =
    timelineMaxMs > 0 ? ((startMs - visMinStart) / timelineMaxMs) * 100 : 0;
  const widthPct = timelineMaxMs > 0 ? (durationMs / timelineMaxMs) * 100 : 0;
  const isBarDimmed = !row.matchesFilter;
  const noTransition = hasAnyInProgress || hasAnyLiveQueued;

  const q = row.span.queuedPhase;
  let qLeftPct = 0;
  let qWidthPct = 0;
  if (q) {
    const qStartMs = new Date(q.createdAt).getTime();
    const qEndMs = isRunning && isQueuedOnlyRoot(row.span) ? now : startMs;
    // FIXME: snapping hides a real gap (typically 0.5–2ms) between queue-end
    // and exec-start. Consider a synthetic "network/dispatch" span to
    // visualize scheduling + worker dispatch latency instead of hiding it.
    const snappedDurMs = qEndMs - qStartMs;
    qLeftPct =
      timelineMaxMs > 0 ? ((qStartMs - visMinStart) / timelineMaxMs) * 100 : 0;
    qWidthPct = timelineMaxMs > 0 ? (snappedDurMs / timelineMaxMs) * 100 : 0;
  }

  const handleClick = () => {
    if (row.hasChildren) {
      expandOnly(row.rowKey);
    }
    onSpanSelect?.(row.span);
  };

  return (
    <div
      className={cn(
        'relative shrink-0 transition-colors',
        rowHighlightClass({
          hovered: isHovered,
          selected: isSelected,
          childOfSelected: isChildOfSelected,
        }),
        isBarDimmed && 'opacity-40',
      )}
      style={{ height: ROW_HEIGHT }}
    >
      {q && (
        <div
          className={cn(
            'absolute bottom-[10px] top-[10px] cursor-pointer rounded-[2px] border border-dashed',
            !noTransition && 'transition-all',
            (row.span.inProgress || isQueuedOnlyRoot(row.span)) &&
              'animate-pulse',
            row.span.inProgress || isQueuedOnlyRoot(row.span)
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
          onMouseEnter={(e) => onHover(row.rowKey, e)}
          onMouseMove={onMouseMove}
          onMouseLeave={() => onHover(null)}
          onClick={handleClick}
        />
      )}
      {!hideExecutionBar && (
        <div
          className={cn(
            'absolute bottom-[10px] top-[10px] cursor-pointer rounded-[2px]',
            getBarColor(row.span),
            !noTransition && 'transition-all',
            row.span.inProgress && 'animate-pulse',
            isSelected
              ? 'ring-2 ring-primary ring-offset-1 ring-offset-background'
              : isHovered
                ? 'ring-1 ring-foreground/20'
                : '',
          )}
          style={{
            left: `min(${leftPct}%, calc(100% - 4px))`,
            width: `${Math.max(widthPct, 0.3)}%`,
            minWidth: 4,
          }}
          onMouseEnter={(e) => onHover(row.rowKey, e)}
          onMouseMove={onMouseMove}
          onMouseLeave={() => onHover(null)}
          onClick={handleClick}
        />
      )}
    </div>
  );
});
