import { useRunDetailSearch } from '../../../../hooks/use-run-detail-search';
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
import { useSidePanel } from '@/hooks/use-side-panel';
import { ChevronsDownUp, ChevronsUpDown } from 'lucide-react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

const CONTEXT_EXPAND_SENTINEL = '__context_expand_done__';

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

  const { open, close } = useSidePanel();

  const resolvedSpan = useMemo(
    () =>
      selectedSpanId ? findSpanInTrees(spanTrees, selectedSpanId) : undefined,
    [selectedSpanId, spanTrees],
  );
  const resolvedGroup = useMemo(
    () =>
      selectedGroupId
        ? findGroupInTrees(spanTrees, selectedGroupId)
        : undefined,
    [selectedGroupId, spanTrees],
  );

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

  const handleDetailClose = useCallback(() => {
    setSelectedSpanId(undefined);
    setSelectedGroupId(undefined);
    close();
  }, [setSelectedSpanId, setSelectedGroupId, close]);

  const handleSpanSelect = useCallback(
    (span: OtelSpanTree) => {
      expandAncestors(span);
      const isDeselecting = selectedSpanId === span.spanId;
      setSelectedSpanId(isDeselecting ? undefined : span.spanId);
      if (isDeselecting) {
        close();
      } else {
        open({
          type: 'span-details',
          content: {
            span,
            activeFilters,
            onAddFilter,
            onRemoveFilter,
            onSpanSelect: (childSpan) => {
              expandAncestors(childSpan);
              setSelectedSpanId(childSpan.spanId);
              open({
                type: 'span-details',
                content: {
                  span: childSpan,
                  activeFilters,
                  onAddFilter,
                  onRemoveFilter,
                  onClose: handleDetailClose,
                },
              });
            },
            onClose: handleDetailClose,
          },
        });
      }
    },
    [
      expandAncestors,
      selectedSpanId,
      setSelectedSpanId,
      open,
      close,
      activeFilters,
      onAddFilter,
      onRemoveFilter,
      handleDetailClose,
    ],
  );

  const handleGroupSelect = useCallback(
    (group: SpanGroupInfo) => {
      const isDeselecting = selectedGroupId === group.groupId;
      setSelectedGroupId(isDeselecting ? undefined : group.groupId);
      if (isDeselecting) {
        close();
      } else {
        open({
          type: 'group-details',
          content: {
            group,
            onClose: handleDetailClose,
          },
        });
      }
    },
    [selectedGroupId, setSelectedGroupId, open, close, handleDetailClose],
  );

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
    close();
    onClearFilters?.();
  }, [setSelectedSpanId, setSelectedGroupId, close, onClearFilters]);

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

  const timelineScrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const key = resolvedSpan
      ? getStableKey(resolvedSpan)
      : resolvedGroup?.groupId;

    if (!key) {
      return;
    }

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
  }, [resolvedSpan, resolvedGroup]);

  return (
    <div className="my-4 flex min-w-0 select-none flex-col gap-y-2">
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
      <div ref={timelineScrollRef}>
        <TraceTimeline
          spanTrees={spanTrees}
          isRunning={isRunning}
          expandedSpanIds={expandedSpansIds}
          onExpandChange={setExpandedSpansIds}
          groupVisibleCounts={groupVisibleCounts}
          onShowMore={handleShowMore}
          selectedSpan={resolvedSpan}
          selectedGroupId={resolvedGroup?.groupId}
          onSpanSelect={handleSpanSelect}
          onGroupSelect={handleGroupSelect}
        />
      </div>
    </div>
  );
}
