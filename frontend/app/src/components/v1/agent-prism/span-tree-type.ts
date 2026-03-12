import type { OtelSpan } from '@/lib/api/generated/data-contracts';

export type OtelSpanTree = OtelSpan & {
  durationMs: number;
  children: OtelSpanTree[];
};
