import type { OtelSpanTree } from './span-tree-type';
import type { OtelSpan } from '@/lib/api/generated/data-contracts';
import invariant from 'tiny-invariant';

const convertOtelSpanToTraceSpan = (span: OtelSpan): OtelSpanTree => ({
  ...span,
  duration_ms: span.duration / 1_000_000,
  children: [],
});

export const convertOtelSpansToOtelSpanTree = (
  spans: [OtelSpan, ...OtelSpan[]],
): OtelSpanTree => {
  const spanMap = new Map<string, OtelSpanTree>();
  let rootSpan: OtelSpanTree | null = null;

  spans.forEach((span) => {
    const converted = convertOtelSpanToTraceSpan(span);
    spanMap.set(converted.span_id, converted);
  });

  spans.forEach((span) => {
    const converted = spanMap.get(span.span_id)!;
    const parentSpanId = span.parent_span_id;
    if (parentSpanId) {
      const parent = spanMap.get(parentSpanId);
      invariant(parent, 'Must have a parent span');
      if (!parent.children) {
        parent.children = [];
      }
      parent.children.push(converted);
    } else {
      rootSpan = converted;
    }
  });

  invariant(rootSpan, 'Must have a root span');

  return rootSpan;
};
