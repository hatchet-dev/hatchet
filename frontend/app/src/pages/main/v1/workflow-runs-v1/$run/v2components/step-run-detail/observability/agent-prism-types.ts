import type { OtelStatusCode } from '@/lib/api/generated/data-contracts';

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
