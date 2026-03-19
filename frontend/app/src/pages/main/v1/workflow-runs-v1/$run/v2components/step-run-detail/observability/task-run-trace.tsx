import { SpanDetail, GroupDetail } from './span-detail';
import {
  TraceTimeline,
  LABEL_WIDTH,
  type VisibleRange,
} from './timeline/trace-timeline';
import type { SpanGroupInfo } from './timeline/trace-timeline-utils';
import { TraceMinimap } from './trace-minimap';
import { getStableKey } from './utils/span-tree-utils';
import { findTimeRange } from '@/components/v1/agent-prism/agent-prism-data';
import type { OtelSpanTree } from '@/components/v1/agent-prism/span-tree-type';
import type {
  FilteredSpanTree,
  ParsedTraceQuery,
} from '@/components/v1/cloud/observability/trace-search';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

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

function findSpanByTaskRunId(
  nodes: OtelSpanTree[],
  taskRunId: string,
  ancestorKeys: string[] = [],
): { span: OtelSpanTree; ancestorKeys: string[] } | undefined {
  for (const node of nodes) {
    const key = getStableKey(node);
    if (
      key === taskRunId ||
      node.spanAttributes?.['hatchet.step_run_id'] === taskRunId
    ) {
      return { span: node, ancestorKeys };
    }
    const found = findSpanByTaskRunId(node.children, taskRunId, [
      ...ancestorKeys,
      key,
    ]);
    if (found) {
      return found;
    }
  }
  return undefined;
}

function collectAncestorKeys(
  nodes: OtelSpanTree[],
  targetSpanId: string,
  ancestorKeys: string[] = [],
): string[] | undefined {
  for (const node of nodes) {
    if (node.spanId === targetSpanId) {
      return ancestorKeys;
    }
    const found = collectAncestorKeys(node.children, targetSpanId, [
      ...ancestorKeys,
      getStableKey(node),
    ]);
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
  focusedTaskRunId,
}: {
  spanTrees: FilteredSpanTree[];
  isRunning?: boolean;
  activeFilters?: ParsedTraceQuery;
  onAddFilter?: (key: string, value: string) => void;
  onRemoveFilter?: (key: string, value: string) => void;
  focusedTaskRunId?: string;
}) {
  const { minStart, maxEnd } = useMemo(
    () => findTimeRange(spanTrees),
    [spanTrees],
  );

  const [expandedSpansIds, setExpandedSpansIds] = useState<Set<string>>(
    () => new Set(spanTrees.map((s) => s.spanId)),
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

  const lastFocusedRef = useRef<string | undefined>();
  useEffect(() => {
    if (!focusedTaskRunId || focusedTaskRunId === lastFocusedRef.current) {
      return;
    }
    const result = findSpanByTaskRunId(spanTrees, focusedTaskRunId);
    if (!result) {
      return;
    }
    lastFocusedRef.current = focusedTaskRunId;
    setExpandedSpansIds((prev) => {
      const next = new Set(prev);
      for (const id of result.ancestorKeys) {
        next.add(id);
      }
      next.add(getStableKey(result.span));
      return next;
    });
    setSelection({ kind: 'span', span: result.span });
  }, [focusedTaskRunId, spanTrees]);

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

  const expandAncestors = useCallback(
    (span: OtelSpanTree) => {
      const ancestors = collectAncestorKeys(spanTrees, span.spanId);
      if (!ancestors) {
        return;
      }
      setExpandedSpansIds((prev) => {
        const next = new Set(prev);
        for (const id of ancestors) {
          next.add(id);
        }
        next.add(getStableKey(span));
        return next;
      });
    },
    [spanTrees],
  );

  const handleSpanSelect = useCallback(
    (span: OtelSpanTree) => {
      expandAncestors(span);
      setSelection((prev) =>
        prev?.kind === 'span' && prev.span.spanId === span.spanId
          ? undefined
          : { kind: 'span', span },
      );
    },
    [expandAncestors],
  );

  const handleMinimapSpanSelect = useCallback(
    (span: OtelSpanTree, _ancestorSpanIds: string[]) => {
      expandAncestors(span);
      setSelection({ kind: 'span', span });
    },
    [expandAncestors],
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
