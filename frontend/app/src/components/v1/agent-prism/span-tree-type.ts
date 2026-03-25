import type { OtelSpan } from "@/lib/api/generated/data-contracts";

export type RelevantOpenTelemetrySpanProperties = Pick<
  OtelSpan,
  | "spanId"
  | "parentSpanId"
  | "spanName"
  | "statusCode"
  | "statusMessage"
  | "durationNs"
  | "createdAt"
  | "spanAttributes"
>;

export type OtelSpanTree = RelevantOpenTelemetrySpanProperties & {
  children: OtelSpanTree[];
  queuedPhase?: OtelSpanTree;
  inProgress?: boolean;
  hasErrorInSubtree?: boolean;
};
