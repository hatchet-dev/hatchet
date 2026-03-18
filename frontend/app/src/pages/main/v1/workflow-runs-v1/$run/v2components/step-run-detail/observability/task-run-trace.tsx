import { SpanDetail } from './span-detail';
import { TraceMinimap } from './trace-minimap';
import {
  TraceTimeline,
  LABEL_WIDTH,
  type VisibleRange,
} from './trace-timeline';
import { findTimeRange } from '@/components/v1/agent-prism/agent-prism-data';
import { convertOtelSpansToOtelSpanTree } from '@/components/v1/agent-prism/convert-otel-spans-to-agent-prism-span-tree';
import type {
  OtelSpanTree,
  RelevantOpenTelemetrySpanProperties,
} from '@/components/v1/agent-prism/span-tree-type';
import { useCallback, useMemo, useState } from 'react';

export function TaskRunTrace({
  spans,
}: {
  spans: [
    RelevantOpenTelemetrySpanProperties,
    ...RelevantOpenTelemetrySpanProperties[],
  ];
  onTaskRunClick?: (taskRunId: string) => void;
}) {
  const spanTrees = useMemo(
    () => convertOtelSpansToOtelSpanTree(spans),
    [spans],
  );

  const { minStart, maxEnd } = useMemo(
    () => findTimeRange(spanTrees),
    [spanTrees],
  );

  const [expandedSpansIds, setExpandedSpansIds] = useState<string[]>(() =>
    spanTrees.map((s) => s.spanId),
  );

  const [selectedSpan, setSelectedSpan] = useState<OtelSpanTree | undefined>();
  const [visibleRange, setVisibleRange] = useState<VisibleRange>({
    startPct: 0,
    endPct: 1,
  });

  const handleSpanSelect = useCallback((span: OtelSpanTree) => {
    setSelectedSpan((prev) =>
      prev?.spanId === span.spanId ? undefined : span,
    );
  }, []);

  const handleSpanDetailClose = useCallback(() => {
    setSelectedSpan(undefined);
  }, []);

  if (spanTrees.length === 0) {
    return (
      <div className="py-4 text-sm text-muted-foreground">
        No trace found for this task run. To collect traces, use the{' '}
        <code className="rounded bg-muted px-1 py-0.5 text-xs">
          HatchetInstrumentor
        </code>{' '}
        in your SDK.
      </div>
    );
  }

  return (
    <div className="my-4 flex min-w-0 flex-col gap-4 overflow-hidden">
      <div className="flex min-w-0">
        <div className="shrink-0" style={{ width: LABEL_WIDTH }} />
        <div className="min-w-0 flex-1 pr-10">
          <TraceMinimap
            spanTrees={spanTrees}
            minMs={minStart}
            maxMs={maxEnd}
            visibleRange={visibleRange}
            onRangeChange={setVisibleRange}
          />
        </div>
      </div>
      <TraceTimeline
        spanTrees={spanTrees}
        expandedSpanIds={expandedSpansIds}
        onExpandChange={setExpandedSpansIds}
        selectedSpan={selectedSpan}
        onSpanSelect={handleSpanSelect}
        visibleRange={visibleRange}
      />
      {selectedSpan && (
        <SpanDetail span={selectedSpan} onClose={handleSpanDetailClose} />
      )}
    </div>
  );
}
