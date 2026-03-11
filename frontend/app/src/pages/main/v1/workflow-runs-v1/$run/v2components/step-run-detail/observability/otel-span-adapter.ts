import type {
  OpenTelemetrySpan,
  TraceSpanAttribute,
} from './agent-prism-types';
import type { OtelSpan } from '@/lib/api/generated/data-contracts';

function recordToAttributes(
  record: Record<string, string> | undefined,
): TraceSpanAttribute[] {
  if (!record) {
    return [];
  }
  return Object.entries(record).map(([key, value]) => ({
    key,
    value: { stringValue: value },
  }));
}

export function convertOtelSpanToOpenTelemetrySpan(
  span: OtelSpan,
): OpenTelemetrySpan {
  const startNanos = BigInt(new Date(span.created_at).getTime()) * 1_000_000n;
  const durationNanos = BigInt(span.duration);
  const endNanos = startNanos + durationNanos;

  return {
    traceId: span.trace_id,
    spanId: span.span_id,
    parentSpanId: span.parent_span_id || undefined,
    name: span.span_name,
    kind: span.span_kind,
    startTimeUnixNano: startNanos.toString(),
    endTimeUnixNano: endNanos.toString(),
    attributes: [
      ...recordToAttributes(span.span_attributes),
      ...recordToAttributes(span.resource_attributes),
      { key: 'service.name', value: { stringValue: span.service_name } },
    ],
    status: {
      code: span.status_code,
      message: span.status_message,
    },
    flags: 0,
  };
}

/**
 * Filters spans to only include those belonging to a specific task and its
 * descendants. Finds the task's root span via hatchet.step_run_id attribute,
 * then collects all descendant spans by following parent_span_id chains.
 */
function filterSpansForTask(
  spans: OtelSpan[],
  taskExternalId: string,
): OtelSpan[] {
  const taskRootSpan = spans.find(
    (s) => s.span_attributes?.['hatchet.step_run_id'] === taskExternalId,
  );

  if (!taskRootSpan) {
    return spans;
  }

  const childrenByParent = new Map<string, OtelSpan[]>();
  for (const s of spans) {
    const pid = s.parent_span_id;
    if (pid) {
      const children = childrenByParent.get(pid) || [];
      children.push(s);
      childrenByParent.set(pid, children);
    }
  }

  const result: OtelSpan[] = [taskRootSpan];
  const queue = [taskRootSpan.span_id];
  while (queue.length > 0) {
    const parentId = queue.shift()!;
    const children = childrenByParent.get(parentId) || [];
    for (const child of children) {
      result.push(child);
      queue.push(child.span_id);
    }
  }

  return result;
}

export function convertOtelSpans(
  spans: OtelSpan[],
  taskExternalId?: string,
): OpenTelemetrySpan[] {
  const filtered = taskExternalId
    ? filterSpansForTask(spans, taskExternalId)
    : spans;
  const converted = filtered.map(convertOtelSpanToOpenTelemetrySpan);

  const spanIdSet = new Set(converted.map((s) => s.spanId));
  return converted.map((s) =>
    s.parentSpanId && !spanIdSet.has(s.parentSpanId)
      ? { ...s, parentSpanId: undefined }
      : s,
  );
}
