import type {
  OtelSpanKind,
  OtelStatusCode,
} from '@/lib/api/generated/data-contracts';

export type TraceSpan<TMetadata = Record<string, unknown>> = {
  id: string;
  title: string;
  created_at: string;
  duration_ms: number;
  raw: string;
  children?: TraceSpan<TMetadata>[];
  status: OtelStatusCode;
  cost?: number;
  metadata?: TMetadata;
};

export type OpenTelemetrySpan = {
  traceId: string;
  spanId: string;
  parentSpanId?: string;
  name: string;
  kind: OtelSpanKind;
  created_at: string;
  duration_ns: number;
  span_attributes?: Record<string, string>;
  resource_attributes?: Record<string, string>;
  status_code: OtelStatusCode;
};
