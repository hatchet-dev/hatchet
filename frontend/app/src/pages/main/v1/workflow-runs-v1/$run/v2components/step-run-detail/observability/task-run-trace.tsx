import { useRunDetailSearch } from '../../../../hooks/use-run-detail-search';
import { TraceMinimap } from './minimap/trace-minimap';
import { TimeTickLabels } from './timeline/time-tick-labels';
import {
  TraceTimeline,
  LABEL_WIDTH,
  type VisibleRange,
} from './timeline/trace-timeline';
import {
  computeTimeTicks,
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
import { useIsFeatureEnabled, FeatureFlagId } from '@/hooks/use-feature-flags';
import { useSidePanel } from '@/hooks/use-side-panel';
import { ChevronsDownUp, ChevronsUpDown, XIcon } from 'lucide-react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

const CONTEXT_EXPAND_SENTINEL = '__context_expand_done__';

const findTimeRange = (
  spanTrees: OtelSpanTree[],
): { minStart: number; maxEnd: number } => {
  let minStart = Infinity;
  let maxEnd = -Infinity;

  const traverse = (node: OtelSpanTree) => {
    const start = new Date(node.createdAt).getTime();
    const end = start + node.durationNs / 1_000_000;
    minStart = Math.min(minStart, start);
    maxEnd = Math.max(maxEnd, end);
    if (node.queuedPhase) {
      const qStart = new Date(node.queuedPhase.createdAt).getTime();
      const qEnd = qStart + node.queuedPhase.durationNs / 1_000_000;
      minStart = Math.min(minStart, qStart);
      maxEnd = Math.max(maxEnd, qEnd);
    }
    node.children?.forEach(traverse);
  };

  spanTrees.forEach(traverse);

  return { minStart, maxEnd };
};

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

  const { minStart, maxEnd } = useMemo(
    () => findTimeRange(spanTrees),
    [spanTrees],
  );

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

  const [visibleRange, setVisibleRange] = useState<VisibleRange>({
    startPct: 0,
    endPct: 1,
  });

  const isZoomed = visibleRange.startPct > 0.001 || visibleRange.endPct < 0.999;

  const zoomedTicks = useMemo(() => {
    if (!isZoomed) {
      return null;
    }
    const totalMs = maxEnd - minStart;
    const visStartMs = totalMs * visibleRange.startPct;
    const visDurationMs =
      totalMs * (visibleRange.endPct - visibleRange.startPct);
    const { ticks } = computeTimeTicks(visDurationMs);
    return { ticks, visDurationMs, visOffsetMs: visStartMs };
  }, [isZoomed, minStart, maxEnd, visibleRange]);

  const [minimapHoverPct, setMinimapHoverPct] = useState<number | null>(null);
  const [timelineHoverPct, setTimelineHoverPct] = useState<number | null>(null);

  const { isEnabled: minimapEnabled } = useIsFeatureEnabled(
    FeatureFlagId.TraceMinimapEnabled,
    false,
  );

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

  const { open, close, isOpen } = useSidePanel();

  const handleDetailClose = useCallback(() => {
    setSelectedSpanId(undefined);
    setSelectedGroupId(undefined);
    close();
  }, [setSelectedSpanId, setSelectedGroupId, close]);

  const prevIsOpenRef = useRef(isOpen);
  useEffect(() => {
    const wasOpen = prevIsOpenRef.current;
    prevIsOpenRef.current = isOpen;
    if (wasOpen && !isOpen) {
      setSelectedSpanId(undefined);
      setSelectedGroupId(undefined);
    }
  }, [isOpen, setSelectedSpanId, setSelectedGroupId]);

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

  const handleMinimapSpanSelect = useCallback(
    (span: OtelSpanTree, _ancestorSpanIds: string[]) => {
      expandAncestors(span);
      setSelectedSpanId(span.spanId);
    },
    [expandAncestors, setSelectedSpanId],
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
    setVisibleRange({ startPct: 0, endPct: 1 });
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

  const timelineRef = useRef<HTMLDivElement>(null);

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
        const container = timelineRef.current;
        if (!container) {
          return;
        }
        const row = container.querySelector(
          `[data-row-key="${key}"]`,
        ) as HTMLElement | null;
        row?.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
      });
    });
    return () => {
      cancelled = true;
    };
  }, [resolvedSpan, resolvedGroup]);

  return (
    <div className="my-4 flex min-w-0 select-none flex-col gap-y-2">
      <div className="shrink-0">
        <div className="flex min-w-0">
          <div
            className="flex shrink-0 flex-wrap items-end gap-1 pb-1 pr-2"
            style={{ width: LABEL_WIDTH }}
          >
            <Button
              variant="ghost"
              size="xs"
              className="gap-1 text-xs"
              onClick={isAllExpanded ? handleCollapseAll : handleExpandAll}
            >
              {isAllExpanded ? (
                <ChevronsDownUp className="size-3" />
              ) : (
                <ChevronsUpDown className="size-3" />
              )}
              {isAllExpanded ? 'collapse all' : 'expand all'}
            </Button>
          </div>
          {minimapEnabled && (
            <div className="min-w-0 flex-1 pr-10">
              <div className="flex justify-end pb-1">
                <Button
                  variant="ghost"
                  size="xs"
                  className={`gap-1 text-xs ${isZoomed ? '' : 'invisible'}`}
                  onClick={() => setVisibleRange({ startPct: 0, endPct: 1 })}
                  tabIndex={isZoomed ? undefined : -1}
                >
                  <XIcon className="size-3" />
                  clear zoom
                </Button>
              </div>
              <TraceMinimap
                spanTrees={spanTrees}
                minMs={minStart}
                maxMs={maxEnd}
                isRunning={isRunning}
                visibleRange={visibleRange}
                onRangeChange={setVisibleRange}
                expandedSpanIds={expandedSpansIds}
                onSpanSelect={handleMinimapSpanSelect}
                externalHoverPct={timelineHoverPct}
                onHoverPctChange={setMinimapHoverPct}
              />
            </div>
          )}
        </div>
        {minimapEnabled && isZoomed && (
          <div className="flex min-w-0">
            <div className="shrink-0" style={{ width: LABEL_WIDTH }} />
            <div className="min-w-0 flex-1 pr-10">
              <svg
                className="h-5 w-full"
                viewBox="0 0 100 100"
                preserveAspectRatio="none"
              >
                <line
                  x1={visibleRange.startPct * 100}
                  y1="0"
                  x2="0"
                  y2="100"
                  className="stroke-border"
                  strokeWidth="1"
                  vectorEffect="non-scaling-stroke"
                />
                <line
                  x1={visibleRange.endPct * 100}
                  y1="0"
                  x2="100"
                  y2="100"
                  className="stroke-border"
                  strokeWidth="1"
                  vectorEffect="non-scaling-stroke"
                />
              </svg>
            </div>
          </div>
        )}
        {minimapEnabled && zoomedTicks && (
          <div className="flex min-w-0">
            <div className="shrink-0" style={{ width: LABEL_WIDTH }} />
            <div className="min-w-0 flex-1 pr-10">
              <TimeTickLabels
                ticks={zoomedTicks.ticks}
                totalMs={zoomedTicks.visDurationMs}
                offsetMs={zoomedTicks.visOffsetMs}
              />
            </div>
          </div>
        )}
      </div>
      <div ref={timelineRef}>
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
          visibleRange={visibleRange}
          onRangeChange={setVisibleRange}
          externalCursorPct={minimapHoverPct}
          onCursorPctChange={setTimelineHoverPct}
        />
      </div>
    </div>
  );
}
