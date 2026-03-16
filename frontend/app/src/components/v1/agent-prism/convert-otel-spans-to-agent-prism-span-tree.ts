import type {
  OtelSpanTree,
  RelevantOpenTelemetrySpanProperties,
} from './span-tree-type';
import { OtelStatusCode } from '@/lib/api/generated/data-contracts';
import invariant from 'tiny-invariant';

export const convertOtelSpansToOtelSpanTree = (
  spans: [
    RelevantOpenTelemetrySpanProperties,
    ...RelevantOpenTelemetrySpanProperties[],
  ],
): OtelSpanTree => {
  const spanMap = new Map<string, OtelSpanTree>();
  const rootSpans: OtelSpanTree[] = [];

  spans.forEach((span) => {
    spanMap.set(span.spanId, {
      spanId: span.spanId,
      parentSpanId: span.parentSpanId,
      spanName: span.spanName,
      statusCode: span.statusCode,
      durationNs: span.durationNs,
      createdAt: span.createdAt,
      spanAttributes: span.spanAttributes,
      children: [],
    });
  });

  spans.forEach((span) => {
    const converted = spanMap.get(span.spanId)!;
    const parentSpanId = span.parentSpanId;
    if (parentSpanId) {
      const parent = spanMap.get(parentSpanId);
      if (parent) {
        parent.children.push(converted);
      } else {
        // Parent not in this span set (e.g. cross-trace reference) — treat as root
        rootSpans.push(converted);
      }
    } else {
      rootSpans.push(converted);
    }
  });

  invariant(rootSpans.length > 0, 'Must have at least one root span');

  if (rootSpans.length === 1) {
    return rootSpans[0];
  }

  // Multiple roots (e.g. DAG with multiple task runs) — create a synthetic root
  const minCreatedAt = rootSpans.reduce(
    (min, s) => (s.createdAt < min ? s.createdAt : min),
    rootSpans[0].createdAt,
  );
  const maxEndNs = rootSpans.reduce((max, s) => {
    const endNs = new Date(s.createdAt).getTime() * 1_000_000 + s.durationNs;
    return endNs > max ? endNs : max;
  }, 0);
  const startNs = new Date(minCreatedAt).getTime() * 1_000_000;

  return {
    spanId: '__synthetic_root__',
    parentSpanId: undefined,
    spanName: 'hatchet.start_workflow',
    statusCode: rootSpans.some((s) => s.statusCode === OtelStatusCode.ERROR)
      ? OtelStatusCode.ERROR
      : OtelStatusCode.OK,
    durationNs: maxEndNs - startNs,
    createdAt: minCreatedAt,
    children: rootSpans,
  };
};
