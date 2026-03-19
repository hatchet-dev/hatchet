import { SpanDetail, GroupDetail } from './span-detail';
import { TraceMinimap } from './trace-minimap';
import type { FilteredSpanTree } from './trace-search/filter';
import type { ParsedTraceQuery } from './trace-search/types';
import {
  TraceTimeline,
  LABEL_WIDTH,
  type VisibleRange,
  type SpanGroupInfo,
} from './trace-timeline';
import { findTimeRange } from '@/components/v1/agent-prism/agent-prism-data';
import type { OtelSpanTree } from '@/components/v1/agent-prism/span-tree-type';
import { useCallback, useMemo, useState } from 'react';

type Selection =
  | { kind: 'span'; span: OtelSpanTree }
  | { kind: 'group'; group: SpanGroupInfo };

function findSpanInTrees(
  nodes: OtelSpanTree[],
  spanId: string,
): OtelSpanTree | undefined {
  for (const node of nodes) {
    if (node.spanId === spanId) {
      return node;
    }
    const found = findSpanInTrees(node.children, spanId);
    if (found) {
      return found;
    }
  }
  return undefined;
}

export function TaskRunTrace({
  spanTrees,
  isRunning,
  activeFilters,
  onAddFilter,
  onRemoveFilter,
}: {
  spanTrees: FilteredSpanTree[];
  isRunning?: boolean;
  activeFilters?: ParsedTraceQuery;
  onAddFilter?: (key: string, value: string) => void;
  onRemoveFilter?: (key: string, value: string) => void;
}) {
  const { minStart, maxEnd } = useMemo(
    () => findTimeRange(spanTrees),
    [spanTrees],
  );

  const [expandedSpansIds, setExpandedSpansIds] = useState<string[]>(() =>
    spanTrees.map((s) => s.spanId),
  );

  const [groupVisibleCounts, setGroupVisibleCounts] = useState<
    Record<string, number>
  >({});

  const [selection, setSelection] = useState<Selection | undefined>();
  const [visibleRange, setVisibleRange] = useState<VisibleRange>({
    startPct: 0,
    endPct: 1,
  });

  const [minimapHoverPct, setMinimapHoverPct] = useState<number | null>(null);
  const [timelineHoverPct, setTimelineHoverPct] = useState<number | null>(null);

  const resolvedSelection = useMemo((): Selection | undefined => {
    if (!selection) {
      return undefined;
    }
    if (selection.kind === 'group') {
      return selection;
    }
    const fresh = findSpanInTrees(spanTrees, selection.span.spanId);
    return fresh ? { kind: 'span', span: fresh } : selection;
  }, [selection, spanTrees]);

  const handleSpanSelect = useCallback((span: OtelSpanTree) => {
    setSelection((prev) =>
      prev?.kind === 'span' && prev.span.spanId === span.spanId
        ? undefined
        : { kind: 'span', span },
    );
  }, []);

  const handleMinimapSpanSelect = useCallback(
    (span: OtelSpanTree, ancestorSpanIds: string[]) => {
      setExpandedSpansIds((prev) => {
        const set = new Set(prev);
        for (const id of ancestorSpanIds) {
          set.add(id);
        }
        set.add(span.spanId);
        return Array.from(set);
      });
      setSelection({ kind: 'span', span });
    },
    [],
  );

  const handleGroupSelect = useCallback((group: SpanGroupInfo) => {
    setSelection((prev) =>
      prev?.kind === 'group' && prev.group.groupId === group.groupId
        ? undefined
        : { kind: 'group', group },
    );
  }, []);

  const handleDetailClose = useCallback(() => {
    setSelection(undefined);
  }, []);

  const handleShowMore = useCallback(
    (groupId: string, newVisibleCount: number) => {
      setGroupVisibleCounts((prev) => ({
        ...prev,
        [groupId]: newVisibleCount,
      }));
    },
    [],
  );

  return (
    <div className="my-4 flex min-w-0 flex-col gap-4 overflow-hidden">
      <div className="flex min-w-0">
        <div className="shrink-0" style={{ width: LABEL_WIDTH }} />
        <div className="min-w-0 flex-1 pr-10">
          <TraceMinimap
            spanTrees={spanTrees}
            minMs={minStart}
            maxMs={maxEnd}
            visibleRange={visibleRange}
            onRangeChange={setVisibleRange}
            expandedSpanIds={expandedSpansIds}
            onSpanSelect={handleMinimapSpanSelect}
            externalHoverPct={timelineHoverPct}
            onHoverPctChange={setMinimapHoverPct}
          />
        </div>
      </div>
      <TraceTimeline
        spanTrees={spanTrees}
        isRunning={isRunning}
        expandedSpanIds={expandedSpansIds}
        onExpandChange={setExpandedSpansIds}
        groupVisibleCounts={groupVisibleCounts}
        onShowMore={handleShowMore}
        selectedSpan={
          resolvedSelection?.kind === 'span'
            ? resolvedSelection.span
            : undefined
        }
        selectedGroupId={
          resolvedSelection?.kind === 'group'
            ? resolvedSelection.group.groupId
            : undefined
        }
        onSpanSelect={handleSpanSelect}
        onGroupSelect={handleGroupSelect}
        visibleRange={visibleRange}
        onRangeChange={setVisibleRange}
        externalCursorPct={minimapHoverPct}
        onCursorPctChange={setTimelineHoverPct}
      />
      {resolvedSelection?.kind === 'span' && (
        <SpanDetail
          span={resolvedSelection.span}
          onClose={handleDetailClose}
          activeFilters={activeFilters}
          onAddFilter={onAddFilter}
          onRemoveFilter={onRemoveFilter}
          onSpanSelect={handleSpanSelect}
        />
      )}
      {resolvedSelection?.kind === 'group' && (
        <GroupDetail
          group={resolvedSelection.group}
          onClose={handleDetailClose}
        />
      )}
    </div>
  );
}
