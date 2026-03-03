import type {
  OpenTelemetrySpan,
  OpenTelemetrySpanKind,
  OpenTelemetryStatusCode,
  TraceSpanAttribute,
} from '@evilmartians/agent-prism-types';
import type { OtelSpan } from '@/lib/api/generated/cloud/data-contracts';

const SPAN_KIND_MAP: Record<string, OpenTelemetrySpanKind> = {
  INTERNAL: 'SPAN_KIND_INTERNAL',
  SERVER: 'SPAN_KIND_SERVER',
  CLIENT: 'SPAN_KIND_CLIENT',
  PRODUCER: 'SPAN_KIND_PRODUCER',
  CONSUMER: 'SPAN_KIND_CONSUMER',
};

const STATUS_CODE_MAP: Record<string, OpenTelemetryStatusCode> = {
  OK: 'STATUS_CODE_OK',
  ERROR: 'STATUS_CODE_ERROR',
  UNSET: 'STATUS_CODE_UNSET',
};

function recordToAttributes(
  record: Record<string, string> | undefined,
): TraceSpanAttribute[] {
  if (!record) return [];
  return Object.entries(record).map(([key, value]) => ({
    key,
    value: { stringValue: value },
  }));
}

/**
 * Converts our API's flat OtelSpan format to the OTLP-style OpenTelemetrySpan
 * expected by @evilmartians/agent-prism-data adapters.
 */
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
    kind: SPAN_KIND_MAP[span.span_kind] || 'SPAN_KIND_INTERNAL',
    startTimeUnixNano: startNanos.toString(),
    endTimeUnixNano: endNanos.toString(),
    attributes: [
      ...recordToAttributes(span.span_attributes),
      ...recordToAttributes(span.resource_attributes),
      { key: 'service.name', value: { stringValue: span.service_name } },
    ],
    status: {
      code: STATUS_CODE_MAP[span.status_code] || 'STATUS_CODE_UNSET',
      message: span.status_message,
    },
    flags: 0,
  };
}

export function convertOtelSpans(spans: OtelSpan[]): OpenTelemetrySpan[] {
  const converted = spans.map(convertOtelSpanToOpenTelemetrySpan);

  // Promote orphaned spans to root spans: if a span's parentSpanId
  // references a span not in this set, clear it so the tree builder
  // treats it as a root instead of silently dropping it.
  const spanIdSet = new Set(converted.map((s) => s.spanId));
  return converted.map((s) =>
    s.parentSpanId && !spanIdSet.has(s.parentSpanId)
      ? { ...s, parentSpanId: undefined }
      : s,
  );
}
