import type {
  OtelSpanKind,
  OtelStatusCode,
} from '@/lib/api/generated/data-contracts';

export type InputOutputData = {
  input?: string;
  output?: string;
};

export type TraceSpan<TMetadata = Record<string, unknown>> = InputOutputData & {
  id: string;
  title: string;
  startTime: Date;
  endTime: Date;
  duration: number;
  raw: string;
  attributes?: TraceSpanAttribute[];
  children?: TraceSpan<TMetadata>[];
  status: OtelStatusCode;
  cost?: number;
  metadata?: TMetadata;
};

export type TraceSpanAttribute = {
  key: string;
  value: TraceSpanAttributeValue;
};

export type TraceSpanAttributeValue = {
  stringValue?: string;
  intValue?: string;
  boolValue?: boolean;
};

// OpenTelemetry types

export type OpenTelemetryDocument = {
  resourceSpans: OpenTelemetryResourceSpan[];
};

export type OpenTelemetryResourceSpan = {
  resource: OpenTelemetryResource;
  scopeSpans: OpenTelemetryScopeSpan[];
  schemaUrl?: string;
};

export type OpenTelemetryResource = {
  attributes: TraceSpanAttribute[];
};

export type OpenTelemetryScopeSpan = {
  scope: OpenTelemetryScope;
  spans: OpenTelemetrySpan[];
  schemaUrl?: string;
};

export type OpenTelemetryScope = {
  name: string;
  version?: string;
};

export type OpenTelemetrySpan = {
  traceId: string;
  spanId: string;
  parentSpanId?: string;
  name: string;
  kind: OtelSpanKind;
  startTimeUnixNano: string;
  endTimeUnixNano: string;
  attributes: TraceSpanAttribute[];
  status: OpenTelemetryStatus;
  flags: number;
  events?: OpenTelemetryEvent[];
  traceState?: string;
  droppedAttributesCount?: number;
  droppedEventsCount?: number;
  droppedLinksCount?: number;
  links?: OpenTelemetryLink[];
};

export type OpenTelemetryEvent = {
  timeUnixNano: string;
  name: string;
  attributes?: TraceSpanAttribute[];
  droppedAttributesCount?: number;
};

export type OpenTelemetryLink = {
  traceId: string;
  spanId: string;
  traceState?: string;
  attributes?: TraceSpanAttribute[];
  droppedAttributesCount?: number;
};

export type OpenTelemetryStatus = {
  code: OtelStatusCode;
  message?: string;
};

export const INPUT_OUTPUT_ATTRIBUTES = {
  INPUT_VALUE: 'input.value',
  OUTPUT_VALUE: 'output.value',
} as const;
