import { useRunDetailSearch } from '../../../../hooks/use-run-detail-search';
import { TraceMinimap } from './minimap/trace-minimap';
import { SpanDetail, GroupDetail } from './span-detail';
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
import { formatDuration } from './utils/format-utils';
import { getStableKey } from './utils/span-tree-utils';
import { findTimeRange } from '@/components/v1/agent-prism/agent-prism-data';
import type { OtelSpanTree } from '@/components/v1/agent-prism/span-tree-type';
import type {
  FilteredSpanTree,
  ParsedTraceQuery,
} from '@/components/v1/cloud/observability/trace-search';
import { Button } from '@/components/v1/ui/button';
import { ChevronsDownUp, ChevronsUpDown, XIcon } from 'lucide-react';
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

export function TaskRunTrace({
  spanTrees,
  isRunning,
  activeFilters,
  onAddFilter,
  onRemoveFilter,
  showInContext,
  onToggleShowInContext,
  contextTaskRunId,
  onClearFilters,
}: {
  spanTrees: FilteredSpanTree[];
  isRunning?: boolean;
  activeFilters?: ParsedTraceQuery;
  onAddFilter?: (key: string, value: string) => void;
  onRemoveFilter?: (key: string, value: string) => void;
  showInContext?: boolean;
  onToggleShowInContext?: () => void;
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
    const set = new Set(spanTrees.map((s) => s.spanId));
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

  const pendingContextExpandRef = useRef(false);

  const handleToggleContext = useCallback(() => {
    if (!showInContext) {
      pendingContextExpandRef.current = true;
    }
    onToggleShowInContext?.();
  }, [showInContext, onToggleShowInContext]);

  if (pendingContextExpandRef.current && showInContext && contextTaskRunId) {
    pendingContextExpandRef.current = false;
    const result = findSpanByTaskRunId(spanTrees, contextTaskRunId);
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

  const handleMinimapSpanSelect = useCallback(
    (span: OtelSpanTree, _ancestorSpanIds: string[]) => {
      expandAncestors(span);
      setSelectedSpanId(span.spanId);
    },
    [expandAncestors, setSelectedSpanId],
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
  }, [setSelectedSpanId]);

  const handleShowMore = useCallback(
    (groupId: string, newVisibleCount: number) => {
      setGroupVisibleCounts((prev) => ({
        ...prev,
        [groupId]: newVisibleCount,
      }));
    },
    [],
  );

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') {
        return;
      }
      const target = e.target as HTMLElement;
      if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA') {
        return;
      }

      setSelectedSpanId(undefined);
      setSelectedGroupId(undefined);
      setVisibleRange({ startPct: 0, endPct: 1 });
      onClearFilters?.();
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [
    resolvedSelection,
    isZoomed,
    setSelectedSpanId,
    setSelectedGroupId,
    onClearFilters,
  ]);

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
      className="my-4 flex min-w-0 flex-col gap-4"
      style={containerHeight ? { height: containerHeight } : undefined}
    >
      <div
        className={
          hasSelection ? 'flex min-h-0 flex-1 flex-col' : 'flex flex-col'
        }
      >
        <div className="shrink-0">
          <div className="flex min-w-0">
            <div
              className="flex shrink-0 flex-wrap items-end justify-end gap-1 pb-1 pr-2"
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
              {onToggleShowInContext && (
                <>
                  <span className="text-xs text-muted-foreground">|</span>
                  <Button
                    variant="ghost"
                    size="xs"
                    className="gap-1 text-xs"
                    onClick={handleToggleContext}
                  >
                    {showInContext ? 'show task only' : 'show in context'}
                  </Button>
                </>
              )}
              {isZoomed && (
                <>
                  <span className="text-xs text-muted-foreground">|</span>
                  <Button
                    variant="ghost"
                    size="xs"
                    className="gap-1 text-xs"
                    onClick={() => setVisibleRange({ startPct: 0, endPct: 1 })}
                  >
                    <XIcon className="size-3" />
                    clear zoom
                  </Button>
                </>
              )}
            </div>
            <div className="min-w-0 flex-1 pr-10">
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
          </div>
          {isZoomed && (
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
          <div className="flex min-w-0">
            <div className="shrink-0" style={{ width: LABEL_WIDTH }} />
            <div className="min-w-0 flex-1 pr-10">
              <div className="relative h-5 overflow-hidden">
                {zoomedTicks?.ticks.map((t) => {
                  if (t >= zoomedTicks.visDurationMs) {
                    return null;
                  }
                  return (
                    <div
                      key={t}
                      className="absolute flex h-full items-center"
                      style={{
                        left: `${zoomedTicks.visDurationMs > 0 ? (t / zoomedTicks.visDurationMs) * 100 : 0}%`,
                      }}
                    >
                      <span className="whitespace-nowrap font-mono text-xs uppercase tracking-wider text-muted-foreground">
                        {formatDuration(t + zoomedTicks.visOffsetMs)}
                      </span>
                    </div>
                  );
                })}
                {zoomedTicks && (
                  <div className="absolute right-0 z-10 flex h-full items-center">
                    <span className="whitespace-nowrap rounded-sm bg-background px-1 font-mono text-xs uppercase tracking-wider text-muted-foreground">
                      {formatDuration(
                        zoomedTicks.visDurationMs + zoomedTicks.visOffsetMs,
                      )}
                    </span>
                  </div>
                )}
              </div>
            </div>
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
            visibleRange={visibleRange}
            onRangeChange={setVisibleRange}
            externalCursorPct={minimapHoverPct}
            onCursorPctChange={setTimelineHoverPct}
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
