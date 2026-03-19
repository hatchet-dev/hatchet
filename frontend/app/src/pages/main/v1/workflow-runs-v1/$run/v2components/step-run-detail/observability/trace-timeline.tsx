import type { OtelSpanTree } from '@/components/v1/agent-prism/span-tree-type';
import { OtelStatusCode } from '@/lib/api/generated/data-contracts';
import { cn } from '@/lib/utils';
import { ChevronRight, ChevronDown, AlertCircle } from 'lucide-react';
import {
  useEffect,
  useMemo,
  useState,
  useCallback,
  useRef,
  type MouseEvent,
} from 'react';
import { createPortal } from 'react-dom';

const ROW_HEIGHT = 40;
export const LABEL_WIDTH = 320;
const CONNECTOR_WIDTH = 12;
const CONNECTOR_GAP = 8;
const GROUP_THRESHOLD = 5;
const INITIAL_GROUP_VISIBLE = 20;
const SHOW_MORE_BATCH = 50;

export type SpanGroupInfo = {
  groupId: string;
  groupName: string;
  spans: OtelSpanTree[];
  errorCount: number;
  totalCount: number;
  earliestStartMs: number;
  latestEndMs: number;
};

type FlatSpanRow = {
  kind: 'span';
  rowKey: string;
  span: OtelSpanTree;
  depth: number;
  isLastChild: boolean;
  connectorFlags: boolean[];
  hasChildren: boolean;
  isExpanded: boolean;
  matchesFilter: boolean;
};

type FlatGroupRow = {
  kind: 'group';
  rowKey: string;
  group: SpanGroupInfo;
  depth: number;
  isLastChild: boolean;
  connectorFlags: boolean[];
  isExpanded: boolean;
};

type FlatShowMoreRow = {
  kind: 'show-more';
  rowKey: string;
  groupId: string;
  remaining: number;
  currentVisible: number;
  depth: number;
  connectorFlags: boolean[];
};

type FlatRow = FlatSpanRow | FlatGroupRow | FlatShowMoreRow;

function hasErrorInTree(span: OtelSpanTree): boolean {
  if (span.statusCode === OtelStatusCode.ERROR) {
    return true;
  }
  return span.children.some(hasErrorInTree);
}

function isEngineSpan(span: OtelSpanTree): boolean {
  return span.spanAttributes?.['hatchet.span_source'] === 'engine';
}

const ENGINE_SPAN_DISPLAY_NAMES: Record<string, string> = {
  'hatchet.engine.queued': 'Queued',
  'hatchet.engine.scheduling': 'Scheduling',
  'hatchet.engine.retry_backoff': 'Retry Backoff',
  'hatchet.engine.workflow_run': 'Workflow Run',
};

function getDisplayName(span: OtelSpanTree): string {
  if (ENGINE_SPAN_DISPLAY_NAMES[span.spanName]) {
    return ENGINE_SPAN_DISPLAY_NAMES[span.spanName];
  }
  if (!span.spanName.startsWith('hatchet.')) {
    return span.spanName;
  }
  // O11Y-FIXME: there is a naming consistency issue on the SDKs
  if (span.spanAttributes?.['hatchet.task_name']) {
    return span.spanAttributes['hatchet.task_name'];
  }
  if (span.spanAttributes?.['hatchet.step_name']) {
    return span.spanAttributes['hatchet.step_name'];
  }
  if (span.spanAttributes?.['hatchet.workflow_name']) {
    return span.spanAttributes['hatchet.workflow_name'];
  }
  const actionId = span.spanAttributes?.['hatchet.action_id'];
  if (actionId?.includes(':')) {
    return actionId.split(':')[0];
  }
  return span.spanName;
}

function groupSiblings(
  children: OtelSpanTree[],
  parentSpanId?: string,
): Array<
  { kind: 'span'; span: OtelSpanTree } | { kind: 'group'; group: SpanGroupInfo }
> {
  if (children.length <= GROUP_THRESHOLD) {
    return children.map((span) => ({ kind: 'span' as const, span }));
  }

  const byName = new Map<string, OtelSpanTree[]>();
  for (const child of children) {
    const name = getDisplayName(child);
    if (!byName.has(name)) {
      byName.set(name, []);
    }
    byName.get(name)!.push(child);
  }

  const result: Array<
    | { kind: 'span'; span: OtelSpanTree }
    | { kind: 'group'; group: SpanGroupInfo }
  > = [];
  const emittedGroups = new Set<string>();

  for (const child of children) {
    const name = getDisplayName(child);
    const siblings = byName.get(name)!;

    if (siblings.length <= GROUP_THRESHOLD) {
      result.push({ kind: 'span' as const, span: child });
    } else if (!emittedGroups.has(name)) {
      emittedGroups.add(name);

      const errors = siblings.filter((s) => hasErrorInTree(s));
      const nonErrors = siblings.filter((s) => !hasErrorInTree(s));
      errors.sort(
        (a, b) =>
          new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
      );
      nonErrors.sort(
        (a, b) =>
          new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
      );
      const sorted = [...errors, ...nonErrors];

      const startTimes = sorted.map((s) => new Date(s.createdAt).getTime());
      const endTimes = sorted.map(
        (s) => new Date(s.createdAt).getTime() + s.durationNs / 1e6,
      );

      result.push({
        kind: 'group' as const,
        group: {
          groupId: `__group_${parentSpanId || 'root'}_${name}`,
          groupName: name,
          spans: sorted,
          errorCount: errors.length,
          totalCount: sorted.length,
          earliestStartMs: Math.min(...startTimes),
          latestEndMs: Math.max(...endTimes),
        },
      });
    }
  }

  return result;
}

function flattenTree(
  trees: OtelSpanTree[],
  expandedIds: Set<string>,
  groupVisibleCounts: Record<string, number>,
  depth = 0,
  connectorFlags: boolean[] = [],
  parentSpanId?: string,
): FlatRow[] {
  const rows: FlatRow[] = [];
  const items = groupSiblings(trees, parentSpanId);

  items.forEach((item, idx) => {
    const isLast = idx === items.length - 1;

    if (item.kind === 'span') {
      const tree = item.span;
      const hasChildren = tree.children.length > 0;
      const isExpanded = expandedIds.has(tree.spanId) && hasChildren;
      const stableKey =
        tree.spanName === 'hatchet.start_step_run' &&
        tree.spanAttributes?.['hatchet.step_run_id']
          ? tree.spanAttributes['hatchet.step_run_id']
          : tree.spanId;

      rows.push({
        kind: 'span',
        rowKey: stableKey,
        span: tree,
        depth,
        isLastChild: isLast,
        connectorFlags: [...connectorFlags],
        hasChildren,
        isExpanded,
        matchesFilter:
          (tree as { matchesFilter?: boolean }).matchesFilter ?? true,
      });

      if (isExpanded) {
        rows.push(
          ...flattenTree(
            tree.children,
            expandedIds,
            groupVisibleCounts,
            depth + 1,
            [...connectorFlags, !isLast],
            tree.spanId,
          ),
        );
      }
    } else {
      const { group } = item;
      const isExpanded = expandedIds.has(group.groupId);

      rows.push({
        kind: 'group',
        rowKey: group.groupId,
        group,
        depth,
        isLastChild: isLast,
        connectorFlags: [...connectorFlags],
        isExpanded,
      });

      if (isExpanded) {
        const visibleCount =
          groupVisibleCounts[group.groupId] ?? INITIAL_GROUP_VISIBLE;
        const clamped = Math.min(visibleCount, group.spans.length);
        const visibleSpans = group.spans.slice(0, clamped);
        const remaining = group.totalCount - clamped;

        visibleSpans.forEach((span, spanIdx) => {
          const isLastInGroup =
            spanIdx === visibleSpans.length - 1 && remaining <= 0;
          const hasChildren = span.children.length > 0;
          const isSpanExpanded = expandedIds.has(span.spanId) && hasChildren;
          const stableKey =
            span.spanName === 'hatchet.start_step_run' &&
            span.spanAttributes?.['hatchet.step_run_id']
              ? span.spanAttributes['hatchet.step_run_id']
              : span.spanId;

          rows.push({
            kind: 'span',
            rowKey: stableKey,
            span,
            depth: depth + 1,
            isLastChild: isLastInGroup,
            connectorFlags: [...connectorFlags, !isLast],
            hasChildren,
            isExpanded: isSpanExpanded,
            matchesFilter:
              (span as { matchesFilter?: boolean }).matchesFilter ?? true,
          });

          if (isSpanExpanded) {
            rows.push(
              ...flattenTree(
                span.children,
                expandedIds,
                groupVisibleCounts,
                depth + 2,
                [...connectorFlags, !isLast, !isLastInGroup],
                span.spanId,
              ),
            );
          }
        });

        if (remaining > 0) {
          rows.push({
            kind: 'show-more',
            rowKey: `${group.groupId}__show-more`,
            groupId: group.groupId,
            remaining,
            currentVisible: clamped,
            depth: depth + 1,
            connectorFlags: [...connectorFlags, !isLast],
          });
        }
      }
    }
  });

  return rows;
}

function computeTimeTicks(totalDurationMs: number): {
  ticks: number[];
  maxTick: number;
} {
  if (totalDurationMs <= 0) {
    return { ticks: [0], maxTick: 0 };
  }

  const niceIntervals = [
    1, 2, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000, 10000, 20000, 30000,
    60000, 120000, 300000, 600000,
  ];
  const targetTicks = 5;
  const rawInterval = totalDurationMs / targetTicks;

  let interval = niceIntervals[niceIntervals.length - 1];
  for (const c of niceIntervals) {
    if (c >= rawInterval) {
      interval = c;
      break;
    }
  }

  const ticks: number[] = [];
  for (let t = 0; t <= totalDurationMs + interval * 0.5; t += interval) {
    ticks.push(t);
    if (ticks.length > 20) {
      break;
    }
  }

  return { ticks, maxTick: ticks[ticks.length - 1] || totalDurationMs };
}

function formatTimeLabel(ms: number): string {
  if (ms === 0) {
    return '0s';
  }
  if (ms < 1000) {
    return `${Math.round(ms)}ms`;
  }
  if (ms < 60000) {
    const s = ms / 1000;
    return Number.isInteger(s) ? `${s}s` : `${s.toFixed(1)}s`;
  }
  const m = Math.floor(ms / 60000);
  const s = Math.floor((ms % 60000) / 1000);
  return s > 0 ? `${m}m${s}s` : `${m}m`;
}

function formatDurationShort(ms: number): string {
  if (ms < 1) {
    return '<1ms';
  }
  if (ms < 1000) {
    return `${ms.toFixed(ms < 10 ? 2 : 1)}ms`;
  }
  if (ms < 60000) {
    return `${(ms / 1000).toFixed(2)}s`;
  }
  const m = Math.floor(ms / 60000);
  const s = ((ms % 60000) / 1000).toFixed(1);
  return `${m}m ${s}s`;
}

function formatTimestamp(iso: string): string {
  const d = new Date(iso);
  const base = d.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
    second: '2-digit',
    hour12: true,
  });
  const ms = String(d.getMilliseconds()).padStart(3, '0');
  return `${base}.${ms}`;
}

function statusLabel(code: string): string {
  switch (code) {
    case OtelStatusCode.OK:
      return 'OK';
    case OtelStatusCode.ERROR:
      return 'Error';
    default:
      return 'Unset';
  }
}

const barColorsByStatus: Record<string, string> = {
  [OtelStatusCode.OK]: 'bg-green-500',
  [OtelStatusCode.UNSET]: 'bg-green-500',
  [OtelStatusCode.ERROR]: 'bg-red-500',
};

function getBarColor(span: OtelSpanTree): string {
  if (span.inProgress) {
    return 'bg-blue-500/60';
  }
  if (isEngineSpan(span)) {
    return span.statusCode === OtelStatusCode.ERROR
      ? 'bg-red-500/40'
      : 'bg-green-500/40';
  }
  if (hasErrorInTree(span)) {
    return 'bg-red-500';
  }
  if (barColorsByStatus[span.statusCode]) {
    return barColorsByStatus[span.statusCode];
  }
  return 'bg-green-500';
}

function getDotColor(span: OtelSpanTree): string {
  if (span.inProgress) {
    return 'bg-blue-500';
  }
  if (hasErrorInTree(span)) {
    return 'bg-red-500';
  }
  return 'bg-green-500';
}

function SpanTooltip({
  row,
  style,
}: {
  row: FlatSpanRow;
  style: React.CSSProperties;
}) {
  const durationMs = row.span.durationNs / 1_000_000;
  const displayName = getDisplayName(row.span);
  const ownStatus = statusLabel(row.span.statusCode);
  const descendantError =
    row.span.statusCode !== OtelStatusCode.ERROR && hasErrorInTree(row.span);
  const started = formatTimestamp(row.span.createdAt);
  const q = row.span.queuedPhase;
  const queueMs = q ? q.durationNs / 1_000_000 : 0;

  return (
    <div
      className="pointer-events-none z-50 overflow-hidden rounded-lg border border-border bg-popover shadow-lg"
      style={{ maxWidth: 420, ...style }}
    >
      <div className="border-b border-border px-3 py-2">
        <div className="font-mono text-sm font-medium text-foreground">
          {displayName}
        </div>
        {displayName !== row.span.spanName && (
          <div className="mt-0.5 truncate font-mono text-xs text-muted-foreground">
            {row.span.spanName}
          </div>
        )}
      </div>

      <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5 px-3 py-2 text-xs">
        {q ? (
          <>
            <span className="text-muted-foreground">Queue Time</span>
            <span className="font-mono font-medium text-foreground">
              {formatDurationShort(queueMs)}
            </span>
            <span className="text-muted-foreground">Execution</span>
            <span className="font-mono font-medium text-foreground">
              {formatDurationShort(durationMs)}
            </span>
            <span className="text-muted-foreground">Total</span>
            <span className="font-mono font-medium text-foreground">
              {formatDurationShort(queueMs + durationMs)}
            </span>
          </>
        ) : (
          <>
            <span className="text-muted-foreground">
              {isEngineSpan(row.span) &&
              row.span.spanName === 'hatchet.engine.queued'
                ? 'Queue Time'
                : 'Duration'}
            </span>
            <span className="font-mono font-medium text-foreground">
              {formatDurationShort(durationMs)}
            </span>
          </>
        )}

        <span className="text-muted-foreground">Status</span>
        <span className="flex items-center gap-1.5">
          <span
            className={cn(
              'size-1.5 shrink-0 rounded-full',
              getDotColor(row.span),
            )}
          />
          <span className="font-mono text-foreground">
            {row.span.inProgress
              ? 'In Progress'
              : descendantError
                ? 'Error (child)'
                : ownStatus}
          </span>
        </span>

        <span className="text-muted-foreground">Started</span>
        <span className="font-mono text-foreground">{started}</span>

        {isEngineSpan(row.span) && (
          <>
            <span className="text-muted-foreground">Source</span>
            <span className="font-mono text-foreground">Engine</span>
          </>
        )}
      </div>
    </div>
  );
}

function GroupTooltip({
  group,
  style,
}: {
  group: SpanGroupInfo;
  style: React.CSSProperties;
}) {
  const durationMs = group.latestEndMs - group.earliestStartMs;

  return (
    <div
      className="pointer-events-none z-50 overflow-hidden rounded-lg border border-border bg-popover shadow-lg"
      style={{ maxWidth: 420, ...style }}
    >
      <div className="border-b border-border px-3 py-2">
        <div className="font-mono text-sm font-medium text-foreground">
          {group.groupName}
        </div>
        <div className="mt-0.5 font-mono text-xs text-muted-foreground">
          {group.totalCount.toLocaleString()} spans
        </div>
      </div>

      <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5 px-3 py-2 text-xs">
        <span className="text-muted-foreground">Time range</span>
        <span className="font-mono font-medium text-foreground">
          {formatDurationShort(durationMs)}
        </span>
        {group.errorCount > 0 && (
          <>
            <span className="text-muted-foreground">Errors</span>
            <span className="font-mono font-medium text-red-500">
              {group.errorCount.toLocaleString()}
            </span>
          </>
        )}
      </div>
    </div>
  );
}

export type VisibleRange = { startPct: number; endPct: number };

interface TraceTimelineProps {
  spanTrees: OtelSpanTree[];
  isRunning?: boolean;
  expandedSpanIds: string[];
  onExpandChange: (ids: string[]) => void;
  groupVisibleCounts: Record<string, number>;
  onShowMore: (groupId: string, newVisibleCount: number) => void;
  selectedSpan?: OtelSpanTree;
  selectedGroupId?: string;
  onSpanSelect?: (span: OtelSpanTree) => void;
  onGroupSelect?: (group: SpanGroupInfo) => void;
  visibleRange?: VisibleRange;
  onRangeChange?: (range: VisibleRange) => void;
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
}: TraceTimelineProps) {
  const [hoveredRowKey, setHoveredRowKey] = useState<string | null>(null);
  const [tooltipPos, setTooltipPos] = useState<{
    x: number;
    y: number;
  } | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const barsRef = useRef<HTMLDivElement>(null);
  const [cursorPct, setCursorPct] = useState<number | null>(null);
  const [brushRange, setBrushRange] = useState<{
    lo: number;
    hi: number;
  } | null>(null);

  const hasAnyInProgress = useMemo(
    () =>
      (function check(nodes: OtelSpanTree[]): boolean {
        return nodes.some((n) => n.inProgress || check(n.children));
      })(spanTrees),
    [spanTrees],
  );

  useEffect(() => {
    if (!spanTrees || spanTrees.length === 0) {
      return;
    }

    const summarizeNode = (n: OtelSpanTree) => ({
      spanId: n.spanId.slice(0, 12),
      spanName: n.spanName,
      displayName: getDisplayName(n),
      source: n.spanAttributes?.['hatchet.span_source'] ?? 'sdk',
      childCount: n.children.length,
      inProgress: n.inProgress ?? false,
      task_name: n.spanAttributes?.['hatchet.task_name'] ?? '',
      step_name: n.spanAttributes?.['hatchet.step_name'] ?? '',
    });

    const lines: string[] = [];
    lines.push(`[group-debug] tree updated — ${spanTrees.length} root(s)`);

    for (const root of spanTrees) {
      const children = root.children;
      const items = groupSiblings(children, root.spanId);
      const groups = items.filter(
        (i): i is { kind: 'group'; group: SpanGroupInfo } => i.kind === 'group',
      );
      const ungrouped = items.filter(
        (i): i is { kind: 'span'; span: OtelSpanTree } => i.kind === 'span',
      );

      lines.push(
        `  root "${getDisplayName(root)}": ${children.length} children → ${groups.length} groups, ${ungrouped.length} ungrouped`,
      );

      for (const item of items) {
        if (item.kind === 'group') {
          const g = item.group;
          const names = g.spans.map(
            (s) =>
              `${s.spanAttributes?.['hatchet.span_source'] === 'engine' ? 'E' : 'S'}:${s.spanId.slice(0, 8)}`,
          );
          lines.push(
            `    group "${g.groupName}" (${g.totalCount} spans) [${names.join(', ')}]`,
          );
        } else {
          const s = item.span;
          const isEngine =
            s.spanAttributes?.['hatchet.span_source'] === 'engine';
          const src = isEngine ? 'E' : 'S';
          lines.push(
            `    span "${getDisplayName(s)}" ${src}:${s.spanId.slice(0, 8)} name=${s.spanName} task_name=${s.spanAttributes?.['hatchet.task_name'] ?? ''} step_name=${s.spanAttributes?.['hatchet.step_name'] ?? ''} children=${s.children.length} inProgress=${s.inProgress ?? false}`,
          );
        }
      }

      for (const child of children) {
        if (child.children.length > GROUP_THRESHOLD) {
          const childItems = groupSiblings(child.children, child.spanId);
          const cGroups = childItems.filter(
            (i): i is { kind: 'group'; group: SpanGroupInfo } =>
              i.kind === 'group',
          );
          const cUngrouped = childItems.filter(
            (i): i is { kind: 'span'; span: OtelSpanTree } => i.kind === 'span',
          );
          lines.push(
            `    nested "${getDisplayName(child)}": ${child.children.length} children → ${cGroups.length} groups, ${cUngrouped.length} ungrouped`,
          );
          for (const ci of childItems) {
            if (ci.kind === 'group') {
              lines.push(
                `      group "${ci.group.groupName}" (${ci.group.totalCount} spans)`,
              );
            } else {
              lines.push(
                `      span "${getDisplayName(ci.span)}" name=${ci.span.spanName}`,
              );
            }
          }
        }
      }
    }

    console.log(lines.join('\n'));
    console.table(
      spanTrees.flatMap((root) => root.children.map(summarizeNode)),
    );
  }, [spanTrees]);

  const [now, setNow] = useState(Date.now);
  useEffect(() => {
    if (!isRunning || !hasAnyInProgress) {
      return;
    }
    let raf: number;
    const tick = () => {
      setNow(Date.now());
      raf = requestAnimationFrame(tick);
    };
    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
  }, [isRunning, hasAnyInProgress]);

  const expandedSet = useMemo(
    () => new Set(expandedSpanIds),
    [expandedSpanIds],
  );

  const flatRows = useMemo(
    () => flattenTree(spanTrees, expandedSet, groupVisibleCounts),
    [spanTrees, expandedSet, groupVisibleCounts],
  );

  const { visMinStart, ticks, timelineMaxMs, traceMinStart, traceTotalMs } =
    useMemo(() => {
      let minStart = Infinity;
      let maxEnd = -Infinity;
      const traverse = (node: OtelSpanTree) => {
        const start = new Date(node.createdAt).getTime();
        const end = node.inProgress ? now : start + node.durationNs / 1e6;
        minStart = Math.min(minStart, start);
        maxEnd = Math.max(maxEnd, end);
        if (node.queuedPhase) {
          const qStart = new Date(node.queuedPhase.createdAt).getTime();
          const qEnd = qStart + node.queuedPhase.durationNs / 1e6;
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
          ticks,
          timelineMaxMs: hasAnyInProgress
            ? visDurationMs
            : Math.max(maxTick, visDurationMs),
          traceMinStart: minStart,
          traceTotalMs: totalDurationMs,
        };
      }

      const { ticks, maxTick } = computeTimeTicks(totalDurationMs);
      const timelineMaxMs = hasAnyInProgress
        ? totalDurationMs
        : Math.max(maxTick, totalDurationMs);
      return {
        visMinStart: minStart,
        ticks,
        timelineMaxMs,
        traceMinStart: minStart,
        traceTotalMs: totalDurationMs,
      };
    }, [spanTrees, visibleRange, now, hasAnyInProgress]);

  const toggleExpand = useCallback(
    (id: string) => {
      if (expandedSpanIds.includes(id)) {
        onExpandChange(expandedSpanIds.filter((eid) => eid !== id));
      } else {
        onExpandChange([...expandedSpanIds, id]);
      }
    },
    [expandedSpanIds, onExpandChange],
  );

  const expandOnly = useCallback(
    (id: string) => {
      if (!expandedSet.has(id)) {
        onExpandChange([...expandedSpanIds, id]);
      }
    },
    [expandedSet, expandedSpanIds, onExpandChange],
  );

  const handleBarHover = useCallback(
    (rowKey: string | null, event?: MouseEvent) => {
      setHoveredRowKey(rowKey);
      if (event) {
        setTooltipPos({ x: event.clientX, y: event.clientY });
      } else {
        setTooltipPos(null);
      }
    },
    [],
  );

  const handleBarMouseMove = useCallback((e: MouseEvent) => {
    setTooltipPos({ x: e.clientX, y: e.clientY });
  }, []);

  const timelineValuesRef = useRef({
    visMinStart: 0,
    timelineMaxMs: 0,
    traceMinStart: 0,
    traceTotalMs: 0,
  });
  timelineValuesRef.current = {
    visMinStart,
    timelineMaxMs,
    traceMinStart,
    traceTotalMs,
  };

  const handleBarsPointerDown = useCallback(
    (e: React.PointerEvent) => {
      if (!barsRef.current || !onRangeChange) {
        return;
      }

      const rect = barsRef.current.getBoundingClientRect();
      const startPct = Math.max(
        0,
        Math.min(1, (e.clientX - rect.left) / rect.width),
      );

      const onMove = (ev: PointerEvent) => {
        if (!barsRef.current) {
          return;
        }
        const r = barsRef.current.getBoundingClientRect();
        const pct = Math.max(0, Math.min(1, (ev.clientX - r.left) / r.width));
        const lo = Math.min(startPct, pct);
        const hi = Math.max(startPct, pct);
        if (hi - lo > 0.005) {
          setBrushRange({ lo, hi });
        }
      };

      const onUp = (ev: PointerEvent) => {
        document.removeEventListener('pointermove', onMove);
        document.removeEventListener('pointerup', onUp);

        if (!barsRef.current) {
          setBrushRange(null);
          return;
        }
        const r = barsRef.current.getBoundingClientRect();
        const pct = Math.max(0, Math.min(1, (ev.clientX - r.left) / r.width));
        const lo = Math.min(startPct, pct);
        const hi = Math.max(startPct, pct);

        setBrushRange(null);

        if (hi - lo >= 0.02) {
          const v = timelineValuesRef.current;
          const newStartMs = v.visMinStart + v.timelineMaxMs * lo;
          const newEndMs = v.visMinStart + v.timelineMaxMs * hi;
          onRangeChange({
            startPct: Math.max(
              0,
              (newStartMs - v.traceMinStart) / v.traceTotalMs,
            ),
            endPct: Math.min(1, (newEndMs - v.traceMinStart) / v.traceTotalMs),
          });
        }
      };

      document.addEventListener('pointermove', onMove);
      document.addEventListener('pointerup', onUp);
    },
    [onRangeChange],
  );

  const handleBarsDoubleClick = useCallback(() => {
    onRangeChange?.({ startPct: 0, endPct: 1 });
  }, [onRangeChange]);

  const hoveredRow = hoveredRowKey
    ? flatRows.find((r) => r.rowKey === hoveredRowKey)
    : null;

  const gridHeight = flatRows.length * ROW_HEIGHT;

  return (
    <div className="relative flex min-w-0 overflow-hidden" ref={containerRef}>
      <div
        className="flex shrink-0 flex-col overflow-hidden pt-6"
        style={{ width: LABEL_WIDTH }}
      >
        {flatRows.map((row) => {
          if (row.kind === 'show-more') {
            return (
              <div
                key={row.rowKey}
                className="flex shrink-0 items-center px-2"
                style={{ height: ROW_HEIGHT }}
              >
                {Array.from({ length: row.depth }).map((_, i) => (
                  <div
                    key={i}
                    className="flex shrink-0 items-center justify-center"
                    style={{ width: CONNECTOR_WIDTH, height: ROW_HEIGHT }}
                  >
                    {row.connectorFlags[i] && (
                      <div className="h-full w-px bg-border" />
                    )}
                  </div>
                ))}
                <div style={{ width: CONNECTOR_GAP }} className="shrink-0" />
                <button
                  className="truncate text-sm text-primary hover:underline"
                  onClick={() =>
                    onShowMore(
                      row.groupId,
                      row.currentVisible + SHOW_MORE_BATCH,
                    )
                  }
                >
                  Show {Math.min(row.remaining, SHOW_MORE_BATCH)} more
                  {row.remaining > SHOW_MORE_BATCH &&
                    ` (${row.remaining.toLocaleString()} remaining)`}
                </button>
              </div>
            );
          }

          if (row.kind === 'group') {
            const isSelected = selectedGroupId === row.group.groupId;
            return (
              <div
                key={row.rowKey}
                className={cn(
                  'flex shrink-0 cursor-pointer items-center rounded-l px-2 transition-colors',
                  isSelected ? 'bg-primary/8' : 'hover:bg-muted/50',
                )}
                style={{ height: ROW_HEIGHT }}
                onClick={() => {
                  expandOnly(row.group.groupId);
                  onGroupSelect?.(row.group);
                }}
              >
                {Array.from({ length: row.depth }).map((_, i) => {
                  const isOwnLevel = i === row.depth - 1;
                  const showLine = isOwnLevel
                    ? row.connectorFlags[i] || !row.isLastChild
                    : row.connectorFlags[i];
                  return (
                    <div
                      key={i}
                      className="flex shrink-0 items-center justify-center"
                      style={{ width: CONNECTOR_WIDTH, height: ROW_HEIGHT }}
                    >
                      {showLine && <div className="h-full w-px bg-border" />}
                    </div>
                  );
                })}

                <button
                  className="flex shrink-0 items-center justify-center text-muted-foreground transition-colors hover:text-foreground"
                  style={{ width: CONNECTOR_WIDTH + CONNECTOR_GAP }}
                  onClick={(e) => {
                    e.stopPropagation();
                    toggleExpand(row.group.groupId);
                  }}
                >
                  {row.isExpanded ? (
                    <ChevronDown className="size-3" />
                  ) : (
                    <ChevronRight className="size-3" />
                  )}
                </button>

                <span
                  className={cn(
                    'truncate text-sm leading-tight',
                    isSelected
                      ? 'font-medium text-foreground'
                      : 'text-muted-foreground',
                  )}
                  title={`batch: ${row.group.groupName} (${row.group.totalCount.toLocaleString()} spans)`}
                >
                  batch: {row.group.groupName}
                </span>
                <span className="ml-1.5 shrink-0 rounded bg-muted px-1.5 py-0.5 font-mono text-xs text-muted-foreground">
                  {row.group.totalCount.toLocaleString()}
                </span>
                {row.group.errorCount > 0 && (
                  <span className="ml-1 flex shrink-0 items-center gap-0.5 rounded bg-red-500/10 px-1.5 py-0.5 font-mono text-xs text-red-500">
                    <AlertCircle className="size-3" />
                    {row.group.errorCount.toLocaleString()}
                  </span>
                )}
              </div>
            );
          }

          const isSelected = selectedSpan?.spanId === row.span.spanId;
          const isDimmed = !row.matchesFilter;

          return (
            <div
              key={row.rowKey}
              className={cn(
                'flex shrink-0 cursor-pointer items-center rounded-l px-2 transition-colors',
                isSelected ? 'bg-primary/8' : 'hover:bg-muted/50',
                isDimmed && 'opacity-40',
              )}
              style={{ height: ROW_HEIGHT }}
              onClick={() => {
                if (row.hasChildren) {
                  expandOnly(row.span.spanId);
                }
                onSpanSelect?.(row.span);
              }}
            >
              {Array.from({ length: row.depth }).map((_, i) => {
                const isOwnLevel = i === row.depth - 1;
                const showLine = isOwnLevel
                  ? row.connectorFlags[i] || !row.isLastChild
                  : row.connectorFlags[i];
                return (
                  <div
                    key={i}
                    className="flex shrink-0 items-center justify-center"
                    style={{ width: CONNECTOR_WIDTH, height: ROW_HEIGHT }}
                  >
                    {showLine && <div className="h-full w-px bg-border" />}
                  </div>
                );
              })}

              {row.hasChildren ? (
                <button
                  className="flex shrink-0 items-center justify-center text-muted-foreground transition-colors hover:text-foreground"
                  style={{ width: CONNECTOR_WIDTH + CONNECTOR_GAP }}
                  onClick={(e) => {
                    e.stopPropagation();
                    toggleExpand(row.span.spanId);
                  }}
                >
                  {row.isExpanded ? (
                    <ChevronDown className="size-3" />
                  ) : (
                    <ChevronRight className="size-3" />
                  )}
                </button>
              ) : row.depth > 0 ? (
                <div style={{ width: CONNECTOR_GAP }} className="shrink-0" />
              ) : null}

              <span
                className={cn(
                  'truncate text-sm leading-tight',
                  isSelected
                    ? 'font-medium text-foreground'
                    : row.depth === 0
                      ? 'text-foreground'
                      : 'text-muted-foreground',
                )}
                title={getDisplayName(row.span)}
              >
                {getDisplayName(row.span)}
              </span>
              {isEngineSpan(row.span) && (
                <span className="ml-1.5 shrink-0 rounded bg-muted px-1 py-0.5 font-mono text-[10px] text-muted-foreground">
                  engine
                </span>
              )}
            </div>
          );
        })}
      </div>

      <div className="flex min-w-0 flex-1 flex-col overflow-hidden pr-10">
        <div className="relative h-6 shrink-0">
          {ticks.map((t, i) => {
            const isLast = i === ticks.length - 1;
            return (
              <div
                key={t}
                className="absolute flex h-full items-center"
                style={{
                  left: `${(t / timelineMaxMs) * 100}%`,
                  transform: isLast ? 'translateX(-100%)' : undefined,
                }}
              >
                <span className="whitespace-nowrap font-mono text-xs uppercase tracking-wider text-muted-foreground">
                  {formatTimeLabel(t)}
                </span>
              </div>
            );
          })}

          {cursorPct !== null && !brushRange && (
            <div
              className="pointer-events-none absolute z-10 flex h-full items-center whitespace-nowrap rounded bg-foreground/90 px-1 py-px font-mono text-[10px] leading-tight text-background"
              style={{
                left: `${cursorPct * 100}%`,
                transform:
                  cursorPct < 0.05
                    ? 'none'
                    : cursorPct > 0.95
                      ? 'translateX(-100%)'
                      : 'translateX(-50%)',
              }}
            >
              {formatTimeLabel(timelineMaxMs * cursorPct)}
            </div>
          )}

          {brushRange && (
            <>
              <div
                className="pointer-events-none absolute z-10 flex h-full items-center whitespace-nowrap rounded bg-foreground/90 px-1 py-px font-mono text-[10px] leading-tight text-background"
                style={{
                  left: `${brushRange.lo * 100}%`,
                  transform: brushRange.lo < 0.05 ? 'none' : 'translateX(-50%)',
                }}
              >
                {formatTimeLabel(timelineMaxMs * brushRange.lo)}
              </div>
              <div
                className="pointer-events-none absolute z-10 flex h-full items-center whitespace-nowrap rounded bg-foreground/90 px-1 py-px font-mono text-[10px] leading-tight text-background"
                style={{
                  left: `${brushRange.hi * 100}%`,
                  transform:
                    brushRange.hi > 0.95
                      ? 'translateX(-100%)'
                      : 'translateX(-50%)',
                }}
              >
                {formatTimeLabel(timelineMaxMs * brushRange.hi)}
              </div>
            </>
          )}
        </div>

        <div
          className="relative"
          ref={barsRef}
          style={{ cursor: onRangeChange ? 'crosshair' : undefined }}
          onMouseMove={(e) => {
            if (!barsRef.current) {
              return;
            }
            const rect = barsRef.current.getBoundingClientRect();
            setCursorPct(
              Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width)),
            );
          }}
          onMouseLeave={() => setCursorPct(null)}
          onPointerDown={handleBarsPointerDown}
          onDoubleClick={handleBarsDoubleClick}
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
              const durationMs =
                row.group.latestEndMs - row.group.earliestStartMs;
              const leftPct =
                timelineMaxMs > 0
                  ? ((row.group.earliestStartMs - visMinStart) /
                      timelineMaxMs) *
                    100
                  : 0;
              const widthPct =
                timelineMaxMs > 0 ? (durationMs / timelineMaxMs) * 100 : 0;
              const isSelected = selectedGroupId === row.group.groupId;
              const hasErrors = row.group.errorCount > 0;

              return (
                <div
                  key={row.rowKey}
                  className={cn(
                    'relative shrink-0 transition-colors',
                    isSelected && 'bg-primary/8',
                  )}
                  style={{ height: ROW_HEIGHT }}
                >
                  <div
                    className={cn(
                      'absolute bottom-[10px] top-[10px] cursor-pointer rounded-sm',
                      !hasAnyInProgress && 'transition-all',
                      hasErrors ? 'bg-red-500' : 'bg-green-500',
                      isSelected
                        ? 'ring-2 ring-primary ring-offset-1 ring-offset-background'
                        : hoveredRowKey === row.rowKey
                          ? 'ring-1 ring-foreground/20'
                          : '',
                    )}
                    style={{
                      left: `${leftPct}%`,
                      width: `${Math.max(widthPct, 0.3)}%`,
                      minWidth: 2,
                    }}
                    onMouseEnter={(e) => handleBarHover(row.rowKey, e)}
                    onMouseMove={handleBarMouseMove}
                    onMouseLeave={() => handleBarHover(null)}
                    onClick={() => {
                      expandOnly(row.group.groupId);
                      onGroupSelect?.(row.group);
                    }}
                  />
                </div>
              );
            }

            const startMs = new Date(row.span.createdAt).getTime();
            const durationMs = row.span.inProgress
              ? Math.max(0, now - startMs)
              : row.span.durationNs / 1_000_000;
            const leftPct =
              timelineMaxMs > 0
                ? ((startMs - visMinStart) / timelineMaxMs) * 100
                : 0;
            const widthPct =
              timelineMaxMs > 0 ? (durationMs / timelineMaxMs) * 100 : 0;
            const isSelected = selectedSpan?.spanId === row.span.spanId;
            const isBarDimmed = !row.matchesFilter;

            const q = row.span.queuedPhase;
            let qLeftPct = 0;
            let qWidthPct = 0;
            if (q) {
              const qStartMs = new Date(q.createdAt).getTime();
              // FIXME: snapping hides a real gap (typically 0.5–2ms) between queue-end
              // and exec-start. Consider a synthetic "network/dispatch" span to
              // visualize scheduling + worker dispatch latency instead of hiding it.
              const snappedDurMs = startMs - qStartMs;
              qLeftPct =
                timelineMaxMs > 0
                  ? ((qStartMs - visMinStart) / timelineMaxMs) * 100
                  : 0;
              qWidthPct =
                timelineMaxMs > 0 ? (snappedDurMs / timelineMaxMs) * 100 : 0;
            }

            return (
              <div
                key={row.rowKey}
                className={cn(
                  'relative shrink-0 transition-colors',
                  isSelected && 'bg-primary/8',
                  isBarDimmed && 'opacity-40',
                )}
                style={{ height: ROW_HEIGHT }}
              >
                {q && (
                  <div
                    className={cn(
                      'absolute bottom-[10px] top-[10px] cursor-pointer overflow-hidden rounded-l-sm',
                      !hasAnyInProgress && 'transition-all',
                      row.span.inProgress
                        ? 'bg-blue-500/20'
                        : hasErrorInTree(row.span)
                          ? 'bg-red-500/20'
                          : 'bg-green-500/20',
                    )}
                    style={{
                      left: `${qLeftPct}%`,
                      width: `${Math.max(qWidthPct, 0.3)}%`,
                      minWidth: 2,
                    }}
                    onMouseEnter={(e) => handleBarHover(row.rowKey, e)}
                    onMouseMove={handleBarMouseMove}
                    onMouseLeave={() => handleBarHover(null)}
                    onClick={() => {
                      if (row.hasChildren) {
                        expandOnly(row.span.spanId);
                      }
                      onSpanSelect?.(row.span);
                    }}
                  >
                    <div
                      className="absolute inset-0 opacity-40"
                      style={{
                        backgroundImage:
                          'repeating-linear-gradient(-45deg, transparent, transparent 3px, rgba(255,255,255,0.18) 3px, rgba(255,255,255,0.18) 6px)',
                      }}
                    />
                  </div>
                )}
                <div
                  className={cn(
                    'absolute bottom-[10px] top-[10px] cursor-pointer rounded-sm',
                    getBarColor(row.span),
                    !hasAnyInProgress && 'transition-all',
                    isSelected
                      ? 'ring-2 ring-primary ring-offset-1 ring-offset-background'
                      : hoveredRowKey === row.rowKey
                        ? 'ring-1 ring-foreground/20'
                        : '',
                  )}
                  style={{
                    left: `${leftPct}%`,
                    width: `${Math.max(widthPct, 0.3)}%`,
                    minWidth: 2,
                  }}
                  onMouseEnter={(e) => handleBarHover(row.rowKey, e)}
                  onMouseMove={handleBarMouseMove}
                  onMouseLeave={() => handleBarHover(null)}
                  onClick={() => {
                    if (row.hasChildren) {
                      expandOnly(row.span.spanId);
                    }
                    onSpanSelect?.(row.span);
                  }}
                />
              </div>
            );
          })}

          {cursorPct !== null && !brushRange && (
            <div
              className="pointer-events-none absolute top-0 z-10 w-px bg-foreground/40"
              style={{ left: `${cursorPct * 100}%`, height: gridHeight }}
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
      </div>

      {hoveredRow &&
        tooltipPos &&
        hoveredRow.kind === 'span' &&
        createPortal(
          <SpanTooltip
            row={hoveredRow}
            style={{
              position: 'fixed',
              left: Math.min(tooltipPos.x + 12, window.innerWidth - 440),
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
              left: Math.min(tooltipPos.x + 12, window.innerWidth - 440),
              top: tooltipPos.y + 16,
            }}
          />,
          document.body,
        )}
    </div>
  );
}
