import type { TraceSpan } from './agent-prism-types';
import type { OtelSpan } from '@/lib/api/generated/data-contracts';

const convertOtelSpanToTraceSpan = (span: OtelSpan): TraceSpan => ({
  id: span.span_id,
  title: span.span_name,
  status: span.status_code,
  duration_ms: span.duration / 1_000_000,
  raw: JSON.stringify(span, null, 2),
  created_at: span.created_at,
  children: [],
});

export const convertSpansToTree = (spans: OtelSpan[]): TraceSpan[] => {
  const spanMap = new Map<string, TraceSpan>();
  const rootSpans: TraceSpan[] = [];

  spans.forEach((span) => {
    const converted = convertOtelSpanToTraceSpan(span);
    spanMap.set(converted.id, converted);
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
