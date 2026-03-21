import {
  getDisplayName,
  getStableKey,
  hasErrorInTree,
} from '../utils/span-tree-utils';
import type { OtelSpanTree } from '@/components/v1/agent-prism/span-tree-type';

const GROUP_THRESHOLD = 5;
const INITIAL_GROUP_VISIBLE = 20;

export type VisibleRange = { startPct: number; endPct: number };

function getCreatedAtMs(span: OtelSpanTree): number {
  return new Date(span.createdAt).getTime();
}

function byCreatedAtAsc(a: OtelSpanTree, b: OtelSpanTree): number {
  return getCreatedAtMs(a) - getCreatedAtMs(b);
}

function getMatchesFilter(span: OtelSpanTree): boolean {
  return (span as { matchesFilter?: boolean }).matchesFilter ?? true;
}

export const ROW_HEIGHT = 40;
export const CONNECTOR_WIDTH = 12;
export const CONNECTOR_GAP = 8;
export const SHOW_MORE_BATCH = 50;

export type SpanGroupInfo = {
  groupId: string;
  groupName: string;
  spans: OtelSpanTree[];
  errorCount: number;
  totalCount: number;
  earliestStartMs: number;
  latestEndMs: number;
};

export type FlatSpanRow = {
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

export type FlatGroupRow = {
  kind: 'group';
  rowKey: string;
  group: SpanGroupInfo;
  depth: number;
  isLastChild: boolean;
  connectorFlags: boolean[];
  isExpanded: boolean;
};

export type FlatShowMoreRow = {
  kind: 'show-more';
  rowKey: string;
  groupId: string;
  remaining: number;
  currentVisible: number;
  depth: number;
  connectorFlags: boolean[];
};

export type FlatRow = FlatSpanRow | FlatGroupRow | FlatShowMoreRow;

export function groupSiblings(
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
      errors.sort(byCreatedAtAsc);
      nonErrors.sort(byCreatedAtAsc);
      const sorted = [...errors, ...nonErrors];

      let earliestStartMs = Infinity;
      let latestEndMs = -Infinity;
      for (const s of sorted) {
        const t = getCreatedAtMs(s);
        earliestStartMs = Math.min(earliestStartMs, t);
        latestEndMs = Math.max(latestEndMs, t + s.durationNs / 1e6);
      }

      result.push({
        kind: 'group' as const,
        group: {
          groupId: `__group_${parentSpanId || 'root'}_${name}`,
          groupName: name,
          spans: sorted,
          errorCount: errors.length,
          totalCount: sorted.length,
          earliestStartMs,
          latestEndMs,
        },
      });
    }
  }

  return result;
}

export function flattenTree(
  trees: OtelSpanTree[],
  expandedIds: Set<string>,
  groupVisibleCounts: Record<string, number>,
  depth = 0,
  connectorFlags: boolean[] = [],
  parentSpanId?: string,
): FlatRow[] {
  const rows: FlatRow[] = [];
  const items = groupSiblings(trees, parentSpanId);

  function pushSpanRow(
    span: OtelSpanTree,
    rowDepth: number,
    isLastChild: boolean,
    flags: boolean[],
  ) {
    const hasChildren = span.children.length > 0;
    const stableKey = getStableKey(span);
    const isExpanded = expandedIds.has(stableKey) && hasChildren;

    rows.push({
      kind: 'span',
      rowKey: stableKey,
      span,
      depth: rowDepth,
      isLastChild,
      connectorFlags: [...flags],
      hasChildren,
      isExpanded,
      matchesFilter: getMatchesFilter(span),
    });

    if (isExpanded) {
      rows.push(
        ...flattenTree(
          span.children,
          expandedIds,
          groupVisibleCounts,
          rowDepth + 1,
          [...flags, !isLastChild],
          span.spanId,
        ),
      );
    }
  }

  items.forEach((item, idx) => {
    const isLast = idx === items.length - 1;

    if (item.kind === 'span') {
      pushSpanRow(item.span, depth, isLast, connectorFlags);
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
        const childFlags = [...connectorFlags, !isLast];

        visibleSpans.forEach((span, spanIdx) => {
          const isLastInGroup =
            spanIdx === visibleSpans.length - 1 && remaining <= 0;
          pushSpanRow(span, depth + 1, isLastInGroup, childFlags);
        });

        if (remaining > 0) {
          rows.push({
            kind: 'show-more',
            rowKey: `${group.groupId}__show-more`,
            groupId: group.groupId,
            remaining,
            currentVisible: clamped,
            depth: depth + 1,
            connectorFlags: childFlags,
          });
        }
      }
    }
  });

  return rows;
}

export function computeTimeTicks(totalDurationMs: number): {
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
