import type { AgentPrismTraceSpan } from './agent-prism-types';

export const flattenSpans = (
  spans: AgentPrismTraceSpan[],
): AgentPrismTraceSpan[] => {
  const result: AgentPrismTraceSpan[] = [];
  const traverse = (items: AgentPrismTraceSpan[]) => {
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
  cards: AgentPrismTraceSpan[],
): { minStart: number; maxEnd: number } =>
  cards.reduce(
    (acc, c) => {
      const start = new Date(c.created_at).getTime();
      const end = start + c.duration_ms;
      return {
        minStart: Math.min(acc.minStart, start),
        maxEnd: Math.max(acc.maxEnd, end),
      };
    },
    {
      minStart:
        cards.length > 0 ? new Date(cards[0].created_at).getTime() : Infinity,
      maxEnd:
        cards.length > 0
          ? new Date(cards[0].created_at).getTime() + cards[0].duration_ms
          : -Infinity,
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

export const getTimelineData = ({
  spanCard,
  minStart,
  maxEnd,
}: {
  spanCard: AgentPrismTraceSpan;
  minStart: number;
  maxEnd: number;
}): { durationMs: number; startPercent: number; widthPercent: number } => {
  const startMs = new Date(spanCard.created_at).getTime();
  const totalRange = maxEnd - minStart;
  const durationMs = spanCard.duration_ms;
  const startPercent = ((startMs - minStart) / totalRange) * 100;
  const widthPercent = (durationMs / totalRange) * 100;
  return { durationMs, startPercent, widthPercent };
};
