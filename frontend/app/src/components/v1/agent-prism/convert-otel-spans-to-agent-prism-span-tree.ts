import type { OtelSpanTree } from './span-tree-type';
import type { OtelSpan } from '@/lib/api/generated/data-contracts';
import invariant from 'tiny-invariant';

export const convertOtelSpansToOtelSpanTree = (
  spans: [OtelSpan, ...OtelSpan[]],
): OtelSpanTree => {
  const spanMap = new Map<string, OtelSpanTree>();
  let rootSpan: OtelSpanTree | null = null;

  spans.forEach((span) => {
    spanMap.set(span.spanId, {
      ...span,
      children: [],
    });
  });

  spans.forEach((span) => {
    const converted = spanMap.get(span.spanId)!;
    const parentSpanId = span.parentSpanId;
    if (parentSpanId) {
      const parent = spanMap.get(parentSpanId);
      invariant(parent, 'Must have a parent span');
      if (!parent.children) {
        parent.children = [];
      }
      parent.children.push(converted);
    } else {
      invariant(rootSpan === null, 'There can be only one (root span)');
      rootSpan = converted;
    }
  });

  invariant(rootSpan, 'Must have a root span');

  return rootSpan;
};
