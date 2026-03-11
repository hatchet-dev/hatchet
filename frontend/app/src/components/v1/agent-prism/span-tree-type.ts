import type { OtelSpan } from '@/lib/api/generated/data-contracts';

export type OtelSpanTree = OtelSpan & {
  duration_ms: number;
  children: OtelSpanTree[];
};
