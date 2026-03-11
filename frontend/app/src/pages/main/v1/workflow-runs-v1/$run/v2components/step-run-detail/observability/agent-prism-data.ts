import type { TraceSpan } from './agent-prism-types';

export const flattenSpans = (spans: TraceSpan[]): TraceSpan[] => {
  const result: TraceSpan[] = [];
  const traverse = (items: TraceSpan[]) => {
    items.forEach((item) => {
      result.push(item);
      if (item.children?.length) {
        traverse(item.children);
      }
    });
  };
  traverse(spans);
  return result;
};

export const findTimeRange = (
  cards: TraceSpan[],
): { minStart: number; maxEnd: number } =>
  cards.reduce(
    (acc, c) => {
      const start = +new Date(c.startTime);
      const end = +new Date(c.endTime);
      return {
        minStart: Math.min(acc.minStart, start),
        maxEnd: Math.max(acc.maxEnd, end),
      };
    },
    {
      minStart: cards.length > 0 ? +new Date(cards[0].startTime) : Infinity,
      maxEnd: cards.length > 0 ? +new Date(cards[0].endTime) : -Infinity,
    },
  );

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

const getDurationMs = (spanCard: TraceSpan): number => {
  const startMs = +spanCard.startTime;
  const endMs = +spanCard.endTime;
  return endMs - startMs;
};

export const getTimelineData = ({
  spanCard,
  minStart,
  maxEnd,
}: {
  spanCard: TraceSpan;
  minStart: number;
  maxEnd: number;
}): { durationMs: number; startPercent: number; widthPercent: number } => {
  const startMs = +spanCard.startTime;
  const totalRange = maxEnd - minStart;
  const durationMs = getDurationMs(spanCard);
  const startPercent = ((startMs - minStart) / totalRange) * 100;
  const widthPercent = (durationMs / totalRange) * 100;
  return { durationMs, startPercent, widthPercent };
};
