import type {
  OtelSpanTree,
  RelevantOpenTelemetrySpanProperties,
} from './span-tree-type';
import invariant from 'tiny-invariant';

export const convertOtelSpansToOtelSpanTree = (
  spans: [
    RelevantOpenTelemetrySpanProperties,
    ...RelevantOpenTelemetrySpanProperties[],
  ],
): OtelSpanTree => {
  const spanMap = new Map<string, OtelSpanTree>();
  let rootSpan: OtelSpanTree | null = null;

  spans.forEach((span) => {
    spanMap.set(span.spanId, {
      spanId: span.spanId,
      parentSpanId: span.parentSpanId,
      spanName: span.spanName,
      statusCode: span.statusCode,
      durationNs: span.durationNs,
      createdAt: span.createdAt,
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
