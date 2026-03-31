import { isQueuedOnlyRoot } from '../utils/span-tree-utils';
import { useLiveClock } from '../utils/use-live-clock';
import { TimelineBars } from './trace-timeline-bars';
import { TimelineLabels } from './trace-timeline-labels';
import {
  flattenTree,
  computeTimeTicks,
  ROW_HEIGHT,
  type SpanGroupInfo,
  type VisibleRange,
} from './trace-timeline-utils';
import type { OtelSpanTree } from '@/components/v1/agent-prism/span-tree-type';
import { useMemo, useCallback, useRef } from 'react';

export const LABEL_WIDTH = 320;

export type { VisibleRange } from './trace-timeline-utils';

interface TraceTimelineProps {
  spanTrees: OtelSpanTree[];
  isRunning?: boolean;
  expandedSpanIds: Set<string>;
  onExpandChange: (ids: Set<string>) => void;
  groupVisibleCounts: Record<string, number>;
  onShowMore: (groupId: string, newVisibleCount: number) => void;
  selectedSpan?: OtelSpanTree;
  selectedGroupId?: string;
  onSpanSelect?: (span: OtelSpanTree) => void;
  onGroupSelect?: (group: SpanGroupInfo) => void;
  visibleRange?: VisibleRange;
  onRangeChange?: (range: VisibleRange) => void;
  externalCursorPct?: number | null;
  onCursorPctChange?: (pct: number | null) => void;
}

export function TraceTimeline({
  spanTrees,
  isRunning,
  expandedSpanIds,
  onExpandChange,
  groupVisibleCounts,
  onShowMore,
  selectedSpan,
  selectedGroupId,
  onSpanSelect,
  onGroupSelect,
  visibleRange,
  onRangeChange,
  externalCursorPct,
  onCursorPctChange,
}: TraceTimelineProps) {
  const containerRef = useRef<HTMLDivElement>(null);

  const hasAnyInProgress = useMemo(
    () =>
      (function check(nodes: OtelSpanTree[]): boolean {
        return nodes.some((n) => n.inProgress || check(n.children));
      })(spanTrees),
    [spanTrees],
  );

  const hasAnyLiveQueued = useMemo(
    () =>
      (function check(nodes: OtelSpanTree[]): boolean {
        return nodes.some((n) => isQueuedOnlyRoot(n) || check(n.children));
      })(spanTrees),
    [spanTrees],
  );
  const hasLiveProgress = hasAnyInProgress || hasAnyLiveQueued;

  const now = useLiveClock(!!isRunning && hasLiveProgress);

  const flatRows = useMemo(
    () => flattenTree(spanTrees, expandedSpanIds, groupVisibleCounts),
    [spanTrees, expandedSpanIds, groupVisibleCounts],
  );

  const {
    visMinStart,
    visOffsetMs,
    ticks,
    timelineMaxMs,
    traceMinStart,
    traceTotalMs,
  } = useMemo(() => {
    let minStart = Infinity;
    let maxEnd = -Infinity;
    const traverse = (node: OtelSpanTree) => {
      const start = new Date(node.createdAt).getTime();
      const end = node.inProgress ? now : start + node.durationNs / 1e6;
      minStart = Math.min(minStart, start);
      maxEnd = Math.max(maxEnd, end);
      if (node.queuedPhase) {
        const qStart = new Date(node.queuedPhase.createdAt).getTime();
        const qEnd =
          isRunning && isQueuedOnlyRoot(node)
            ? now
            : qStart + node.queuedPhase.durationNs / 1e6;
        minStart = Math.min(minStart, qStart);
        maxEnd = Math.max(maxEnd, qEnd);
      }
      node.children?.forEach(traverse);
    };
    spanTrees.forEach(traverse);

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
        visOffsetMs: visStartMs - minStart,
        ticks,
        timelineMaxMs: hasLiveProgress
          ? visDurationMs
          : Math.max(maxTick, visDurationMs),
        traceMinStart: minStart,
        traceTotalMs: totalDurationMs,
      };
    }

    const { ticks, maxTick } = computeTimeTicks(totalDurationMs);
    const timelineMaxMs = hasLiveProgress
      ? totalDurationMs
      : Math.max(maxTick, totalDurationMs);
    return {
      visMinStart: minStart,
      visOffsetMs: 0,
      ticks,
      timelineMaxMs,
      traceMinStart: minStart,
      traceTotalMs: totalDurationMs,
    };
  }, [spanTrees, visibleRange, now, hasLiveProgress, isRunning]);

  const toggleExpand = useCallback(
    (id: string) => {
      const next = new Set(expandedSpanIds);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      onExpandChange(next);
    },
    [expandedSpanIds, onExpandChange],
  );

  const expandOnly = useCallback(
    (id: string) => {
      if (!expandedSpanIds.has(id)) {
        const next = new Set(expandedSpanIds);
        next.add(id);
        onExpandChange(next);
      }
    },
    [expandedSpanIds, onExpandChange],
  );

  const gridHeight = flatRows.length * ROW_HEIGHT;

  return (
    <div className="relative flex min-w-0 overflow-hidden" ref={containerRef}>
      <div
        className="flex shrink-0 flex-col overflow-hidden pt-6"
        style={{ width: LABEL_WIDTH }}
      >
        <TimelineLabels
          flatRows={flatRows}
          selectedSpan={selectedSpan}
          selectedGroupId={selectedGroupId}
          onSpanSelect={onSpanSelect}
          onGroupSelect={onGroupSelect}
          onShowMore={onShowMore}
          toggleExpand={toggleExpand}
          expandOnly={expandOnly}
        />
      </div>

      <TimelineBars
        flatRows={flatRows}
        ticks={ticks}
        timelineMaxMs={timelineMaxMs}
        visMinStart={visMinStart}
        visOffsetMs={visOffsetMs}
        traceMinStart={traceMinStart}
        traceTotalMs={traceTotalMs}
        gridHeight={gridHeight}
        now={now}
        isRunning={isRunning}
        hasAnyInProgress={hasAnyInProgress}
        hasAnyLiveQueued={hasAnyLiveQueued}
        selectedSpan={selectedSpan}
        selectedGroupId={selectedGroupId}
        onSpanSelect={onSpanSelect}
        onGroupSelect={onGroupSelect}
        expandOnly={expandOnly}
        onRangeChange={onRangeChange}
        externalCursorPct={externalCursorPct}
        onCursorPctChange={onCursorPctChange}
      />
    </div>
  );
}
