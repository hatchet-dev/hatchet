import type { OtelSpanTree } from '@/components/v1/agent-prism/span-tree-type';
import type { OtelStatusCode } from '@/lib/api/generated/data-contracts';

export const MIN_RANGE_PCT = 0.05;

export type TimeRange = { startPct: number; endPct: number };
export type DragMode = 'left' | 'right' | 'brush' | null;

export type SpanMarker = {
  pct: number;
  statusCode: OtelStatusCode;
  hasErrorInTree: boolean;
  inProgress: boolean;
  spanName: string;
  durationMs: number;
  visible: boolean;
  span: OtelSpanTree;
  ancestorSpanIds: string[];
};

export type DragState = {
  mouseX: number;
  startPct: number;
  endPct: number;
  anchorPct?: number;
};

export interface TraceMinimapProps {
  spanTrees: OtelSpanTree[];
  minMs: number;
  maxMs: number;
  visibleRange: TimeRange;
  onRangeChange: (range: TimeRange) => void;
  expandedSpanIds?: Set<string>;
  onSpanSelect?: (span: OtelSpanTree, ancestorSpanIds: string[]) => void;
  externalHoverPct?: number | null;
  onHoverPctChange?: (pct: number | null) => void;
}
