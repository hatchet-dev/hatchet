import type { OpenTelemetrySpan, TraceSpan } from './agent-prism-types';

const convertRawSpanToTraceSpan = (span: OpenTelemetrySpan): TraceSpan => ({
  id: span.spanId,
  title: span.name,
  status: span.status_code,
  duration_ms: span.duration_ns / 1_000_000,
  raw: JSON.stringify(span, null, 2),
  created_at: span.created_at,
  children: [],
});

export const convertSpansToTree = (spans: OpenTelemetrySpan[]): TraceSpan[] => {
  const spanMap = new Map<string, TraceSpan>();
  const rootSpans: TraceSpan[] = [];

  spans.forEach((span) => {
    const converted = convertRawSpanToTraceSpan(span);
    spanMap.set(converted.id, converted);
  });

  spans.forEach((span) => {
    const converted = spanMap.get(span.spanId)!;
    const parentSpanId = span.parentSpanId;
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
