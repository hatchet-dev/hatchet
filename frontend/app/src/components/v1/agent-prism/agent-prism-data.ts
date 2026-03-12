import type { OtelSpanTree } from './span-tree-type';

export const findTimeRange = (
  spanTree: OtelSpanTree,
): { minStart: number; maxEnd: number } => {
  let minStart = Infinity;
  let maxEnd = -Infinity;

  const traverse = (node: OtelSpanTree) => {
    const start = new Date(node.createdAt).getTime();
    const end = start + node.durationNs / 1_000_000;
    minStart = Math.min(minStart, start);
    maxEnd = Math.max(maxEnd, end);
    node.children?.forEach(traverse);
  };

  traverse(spanTree);

  return { minStart, maxEnd };
};

export const formatDuration = (durationMs: number): string => {
  if (durationMs <= 0) {
    return '0ms';
  }
  if (durationMs < 1000) {
    return `${Math.round(durationMs)}ms`;
  }
  if (durationMs < 60000) {
    return `${Math.round(durationMs / 1000)}s`;
  }
  if (durationMs < 3600000) {
    const m = Math.floor(durationMs / 60000);
    const s = Math.floor((durationMs % 60000) / 1000);
    return s > 0 ? `${m}m ${s}s` : `${m}m`;
  }
  const h = Math.floor(durationMs / 3600000);
  const m = Math.floor((durationMs % 3600000) / 60000);
  return m > 0 ? `${h}h ${m}m` : `${h}h`;
};

export const getTimelineData = ({
  spanCard,
  minStart,
  maxEnd,
}: {
  spanCard: OtelSpanTree;
  minStart: number;
  maxEnd: number;
}): { durationMs: number; startPercent: number; widthPercent: number } => {
  const startMs = new Date(spanCard.createdAt).getTime();
  const totalRange = maxEnd - minStart;
  const durationMs = spanCard.durationNs / 1_000_000;
  const startPercent = ((startMs - minStart) / totalRange) * 100;
  const widthPercent = (durationMs / totalRange) * 100;
  return { durationMs, startPercent, widthPercent };
};
