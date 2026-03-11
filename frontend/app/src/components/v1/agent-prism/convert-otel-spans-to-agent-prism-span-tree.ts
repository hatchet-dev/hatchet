import type { OtelSpanTree } from './span-tree-type';
import type { OtelSpan } from '@/lib/api/generated/data-contracts';

const convertOtelSpanToTraceSpan = (span: OtelSpan): OtelSpanTree => ({
  ...span,
  duration_ms: span.duration / 1_000_000,
  children: [],
});

export const convertOtelSpansToAgentPrismSpanTree = (
  spans: OtelSpan[],
): OtelSpanTree[] => {
  const spanMap = new Map<string, OtelSpanTree>();
  const rootSpans: OtelSpanTree[] = [];

  spans.forEach((span) => {
    const converted = convertOtelSpanToTraceSpan(span);
    spanMap.set(converted.span_id, converted);
  });

  spans.forEach((span) => {
    const converted = spanMap.get(span.span_id)!;
    const parentSpanId = span.parent_span_id;
    if (parentSpanId) {
      const parent = spanMap.get(parentSpanId);
      if (parent) {
        if (!parent.children) {
          parent.children = [];
        }
        parent.children.push(converted);
      } else {
        rootSpans.push(converted);
      }
    } else {
      rootSpans.push(converted);
    }
  });

  return rootSpans;
};
