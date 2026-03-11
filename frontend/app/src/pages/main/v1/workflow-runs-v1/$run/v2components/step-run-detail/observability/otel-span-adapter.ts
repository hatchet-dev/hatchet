import type { OpenTelemetrySpan } from './agent-prism-types';
import type { OtelSpan } from '@/lib/api/generated/data-contracts';

function convertOtelSpanToAgentPrismSpan(span: OtelSpan): OpenTelemetrySpan {
  return {
    traceId: span.trace_id,
    spanId: span.span_id,
    parentSpanId: span.parent_span_id || undefined,
    name: span.span_name,
    kind: span.span_kind,
    created_at: span.created_at,
    duration_ns: span.duration,
    span_attributes: span.span_attributes,
    resource_attributes: span.resource_attributes,
    status_code: span.status_code,
  };
}

export const convertOtelSpansToAgentPrismSpans = (spans: OtelSpan[]) =>
  spans.map(convertOtelSpanToAgentPrismSpan);
