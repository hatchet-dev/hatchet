import type { OtelStatusCode } from '@/lib/api/generated/data-contracts';

export type AgentPrismTraceSpan<TMetadata = Record<string, unknown>> = {
  id: string;
  title: string;
  created_at: string;
  duration_ms: number;
  raw: string;
  children?: AgentPrismTraceSpan<TMetadata>[];
  status: OtelStatusCode;
  cost?: number;
  metadata?: TMetadata;
};
