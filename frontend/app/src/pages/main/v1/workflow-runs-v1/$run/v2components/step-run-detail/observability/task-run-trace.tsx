import { useRunDetailSearch } from '../../../../hooks/use-run-detail-search';
import { SpanDetail, GroupDetail } from './span-detail';
import { TraceTimeline, LABEL_WIDTH } from './timeline/trace-timeline';
import {
  groupSiblings,
  type SpanGroupInfo,
} from './timeline/trace-timeline-utils';
import { getStableKey } from './utils/span-tree-utils';
import type { OtelSpanTree } from '@/components/v1/agent-prism/span-tree-type';
import type {
  FilteredSpanTree,
  ParsedTraceQuery,
} from '@/components/v1/cloud/observability/trace-search';
import { Button } from '@/components/v1/ui/button';
import { ChevronsDownUp, ChevronsUpDown } from 'lucide-react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

const CONTEXT_EXPAND_SENTINEL = '__context_expand_done__';

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

function findGroupInTrees(
  trees: OtelSpanTree[],
  targetGroupId: string,
  parentSpanId?: string,
): SpanGroupInfo | undefined {
  const items = groupSiblings(trees, parentSpanId);
  for (const item of items) {
    if (item.kind === 'group' && item.group.groupId === targetGroupId) {
      return item.group;
    }
  }
  for (const item of items) {
    if (item.kind === 'span') {
      const found = findGroupInTrees(
        item.span.children,
        targetGroupId,
        item.span.spanId,
      );
      if (found) {
        return found;
      }
    } else if (item.kind === 'group') {
      for (const span of item.group.spans) {
        const found = findGroupInTrees(
          span.children,
          targetGroupId,
          span.spanId,
        );
        if (found) {
          return found;
        }
      }
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

function collectAllKeys(nodes: OtelSpanTree[]): string[] {
  const keys: string[] = [];
  for (const node of nodes) {
    if (node.children.length > 0) {
      keys.push(getStableKey(node));
      keys.push(...collectAllKeys(node.children));
    }
  }
  return keys;
}

function collectKeysUpToDepth(
  nodes: OtelSpanTree[],
  maxDepth: number,
  depth = 0,
): string[] {
  if (depth >= maxDepth) {
    return [];
  }
  const keys: string[] = [];
  for (const node of nodes) {
    if (node.children.length > 0) {
      keys.push(getStableKey(node));
      keys.push(...collectKeysUpToDepth(node.children, maxDepth, depth + 1));
    }
  }
  return keys;
}

export function TaskRunTrace({
  spanTrees,
  isRunning,
  activeFilters,
  onAddFilter,
  onRemoveFilter,
  contextTaskRunId,
  onClearFilters,
}: {
  spanTrees: FilteredSpanTree[];
  isRunning?: boolean;
  activeFilters?: ParsedTraceQuery;
  onAddFilter?: (key: string, value: string) => void;
  onRemoveFilter?: (key: string, value: string) => void;
  contextTaskRunId?: string;
  onClearFilters?: () => void;
}) {
  const {
    focusedTaskRunId,
    selectedSpanId,
    setSelectedSpanId,
    selectedGroupId,
    setSelectedGroupId,
  } = useRunDetailSearch();

  const [expandedSpansIds, setExpandedSpansIds] = useState<Set<string>>(() => {
    const set = new Set(collectKeysUpToDepth(spanTrees, 2));
    if (selectedSpanId) {
      const span = findSpanInTrees(spanTrees, selectedSpanId);
      if (span) {
        const ancestors = collectAncestorKeys(spanTrees, selectedSpanId);
        if (ancestors) {
          for (const id of ancestors) {
            set.add(id);
          }
        }
        set.add(getStableKey(span));
      }
    }
    return set;
  });

  const allExpandableKeys = useMemo(
    () => collectAllKeys(spanTrees),
    [spanTrees],
  );
  const isAllExpanded =
    allExpandableKeys.length > 0 &&
    allExpandableKeys.every((k) => expandedSpansIds.has(k));

  const handleExpandAll = useCallback(() => {
    setExpandedSpansIds((prev) => {
      const next = new Set(prev);
      for (const k of allExpandableKeys) {
        next.add(k);
      }
      return next;
    });
  }, [allExpandableKeys]);

  const handleCollapseAll = useCallback(() => {
    setExpandedSpansIds(new Set());
  }, []);

  const [groupVisibleCounts, setGroupVisibleCounts] = useState<
    Record<string, number>
  >({});

  const prevFocusedRef = useRef(focusedTaskRunId);
  if (focusedTaskRunId && focusedTaskRunId !== prevFocusedRef.current) {
    prevFocusedRef.current = focusedTaskRunId;
    const result = findSpanByTaskRunId(spanTrees, focusedTaskRunId);
    if (result) {
      setExpandedSpansIds((prev) => {
        const next = new Set(prev);
        for (const id of result.ancestorKeys) {
          next.add(id);
        }
        next.add(getStableKey(result.span));
        return next;
      });
      setSelectedSpanId(result.span.spanId);
    }
  }

  const contextExpandRef = useRef<string | null>(null);

  if (
    contextTaskRunId &&
    contextExpandRef.current !== CONTEXT_EXPAND_SENTINEL
  ) {
    const result = findSpanByTaskRunId(spanTrees, contextTaskRunId);
    if (result) {
      contextExpandRef.current = CONTEXT_EXPAND_SENTINEL;
      setExpandedSpansIds((prev) => {
        const next = new Set(prev);
        for (const id of result.ancestorKeys) {
          next.add(id);
        }
        next.add(getStableKey(result.span));
        return next;
      });
    }
  }

  const resolvedSelection = useMemo((): Selection | undefined => {
    if (selectedSpanId) {
      const span = findSpanInTrees(spanTrees, selectedSpanId);
      return span ? { kind: 'span', span } : undefined;
    }
    if (selectedGroupId) {
      const group = findGroupInTrees(spanTrees, selectedGroupId);
      return group ? { kind: 'group', group } : undefined;
    }
    return undefined;
  }, [selectedSpanId, selectedGroupId, spanTrees]);

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
      setSelectedSpanId(
        selectedSpanId === span.spanId ? undefined : span.spanId,
      );
    },
    [expandAncestors, selectedSpanId, setSelectedSpanId],
  );

  const handleGroupSelect = useCallback(
    (group: SpanGroupInfo) => {
      setSelectedGroupId(
        selectedGroupId === group.groupId ? undefined : group.groupId,
      );
    },
    [selectedGroupId, setSelectedGroupId],
  );

  const handleDetailClose = useCallback(() => {
    setSelectedSpanId(undefined);
    setSelectedGroupId(undefined);
  }, [setSelectedSpanId, setSelectedGroupId]);

  const handleShowMore = useCallback(
    (groupId: string, newVisibleCount: number) => {
      setGroupVisibleCounts((prev) => ({
        ...prev,
        [groupId]: newVisibleCount,
      }));
    },
    [],
  );

  const handleEscapeReset = useCallback(() => {
    setSelectedSpanId(undefined);
    setSelectedGroupId(undefined);
    onClearFilters?.();
  }, [setSelectedSpanId, setSelectedGroupId, onClearFilters]);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') {
        return;
      }
      const target = e.target as HTMLElement;
      if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA') {
        return;
      }
      handleEscapeReset();
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [handleEscapeReset]);

  const hasSelection = !!resolvedSelection;
  const containerRef = useRef<HTMLDivElement>(null);
  const timelineScrollRef = useRef<HTMLDivElement>(null);
  const [containerHeight, setContainerHeight] = useState<number | undefined>();

  useEffect(() => {
    const el = containerRef.current;
    if (!hasSelection || !el) {
      setContainerHeight(undefined);
      return;
    }

    function measure() {
      const top = el!.getBoundingClientRect().top;
      setContainerHeight(window.innerHeight - top - 16);
    }

    measure();
    window.addEventListener('resize', measure);
    return () => window.removeEventListener('resize', measure);
  }, [hasSelection]);

  useEffect(() => {
    if (!resolvedSelection) {
      return;
    }
    const key =
      resolvedSelection.kind === 'span'
        ? getStableKey(resolvedSelection.span)
        : resolvedSelection.group.groupId;

    let cancelled = false;
    requestAnimationFrame(() => {
      if (cancelled) {
        return;
      }
      requestAnimationFrame(() => {
        if (cancelled) {
          return;
        }
        const scrollContainer = timelineScrollRef.current;
        if (!scrollContainer) {
          return;
        }
        const row = scrollContainer.querySelector(
          `[data-row-key="${key}"]`,
        ) as HTMLElement | null;
        if (row) {
          const rowTop = row.offsetTop;
          const target = Math.max(0, rowTop - scrollContainer.clientHeight / 3);
          scrollContainer.scrollTo({ top: target, behavior: 'smooth' });
        }
      });
    });
    return () => {
      cancelled = true;
    };
  }, [resolvedSelection]);

  return (
    <div
      ref={containerRef}
      className="my-4 flex min-w-0 select-none flex-col gap-4"
      style={containerHeight ? { height: containerHeight } : undefined}
    >
      <div
        className={
          hasSelection ? 'flex min-h-0 flex-1 flex-col' : 'flex flex-col'
        }
      >
        <div className="shrink-0">
          <div style={{ width: LABEL_WIDTH }}>
            <Button
              variant="ghost"
              size="xs"
              className="gap-1 text-xs p-2"
              onClick={isAllExpanded ? handleCollapseAll : handleExpandAll}
            >
              {isAllExpanded ? (
                <ChevronsDownUp className="size-3" />
              ) : (
                <ChevronsUpDown className="size-3" />
              )}
              {isAllExpanded ? 'Collapse All' : 'Expand All'}
            </Button>
          </div>
        </div>
        <div
          ref={timelineScrollRef}
          className={hasSelection ? 'min-h-0 flex-1 overflow-y-auto' : ''}
        >
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
          />
        </div>
      </div>

      {resolvedSelection?.kind === 'span' && (
        <div className="flex min-h-0 flex-1 flex-col">
          <SpanDetail
            span={resolvedSelection.span}
            onClose={handleDetailClose}
            activeFilters={activeFilters}
            onAddFilter={onAddFilter}
            onRemoveFilter={onRemoveFilter}
            onSpanSelect={handleSpanSelect}
          />
        </div>
      )}
      {resolvedSelection?.kind === 'group' && (
        <div className="flex min-h-0 flex-1 flex-col">
          <GroupDetail
            group={resolvedSelection.group}
            onClose={handleDetailClose}
          />
        </div>
      )}
    </div>
  );
}
