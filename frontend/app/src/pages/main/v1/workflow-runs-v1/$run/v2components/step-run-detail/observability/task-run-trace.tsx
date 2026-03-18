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

function collectAncestorIds(
  trees: OtelSpanTree[],
  targetSpanId: string,
): string[] {
  const path: string[] = [];

  function walk(node: OtelSpanTree, ancestors: string[]): boolean {
    if (node.spanId === targetSpanId) {
      path.push(...ancestors);
      return true;
    }
    for (const child of node.children) {
      if (walk(child, [...ancestors, node.spanId])) {
        return true;
      }
    }
    return false;
  }

  for (const tree of trees) {
    if (walk(tree, [])) {
      break;
    }
  }
  return path;
}

type Selection =
  | { kind: 'span'; span: OtelSpanTree }
  | { kind: 'group'; group: SpanGroupInfo };

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

  const handleSpanSelect = useCallback((span: OtelSpanTree) => {
    setSelection((prev) =>
      prev?.kind === 'span' && prev.span.spanId === span.spanId
        ? undefined
        : { kind: 'span', span },
    );
  }, []);

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

  const handleMinimapSpanClick = useCallback(
    (span: OtelSpanTree) => {
      const ancestorIds = collectAncestorIds(spanTrees, span.spanId);
      const idsToExpand = [...ancestorIds, span.spanId];
      setExpandedSpansIds((prev) => {
        const set = new Set(prev);
        for (const id of idsToExpand) {
          set.add(id);
        }
        return Array.from(set);
      });
      setSelection({ kind: 'span', span });
    },
    [spanTrees],
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
            onSpanClick={handleMinimapSpanClick}
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
        selectedSpan={selection?.kind === 'span' ? selection.span : undefined}
        selectedGroupId={
          selection?.kind === 'group' ? selection.group.groupId : undefined
        }
        onSpanSelect={handleSpanSelect}
        onGroupSelect={handleGroupSelect}
        visibleRange={visibleRange}
        onRangeChange={setVisibleRange}
      />
      {selection?.kind === 'span' && (
        <SpanDetail
          span={selection.span}
          onClose={handleDetailClose}
          activeFilters={activeFilters}
          onAddFilter={onAddFilter}
          onRemoveFilter={onRemoveFilter}
        />
      )}
      {selection?.kind === 'group' && (
        <GroupDetail group={selection.group} onClose={handleDetailClose} />
      )}
    </div>
  );
}
