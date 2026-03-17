import { TraceTimeline } from './trace-timeline';
import { convertOtelSpansToOtelSpanTree } from '@/components/v1/agent-prism/convert-otel-spans-to-agent-prism-span-tree';
import type {
  OtelSpanTree,
  RelevantOpenTelemetrySpanProperties,
} from '@/components/v1/agent-prism/span-tree-type';
import { useCallback, useMemo, useState } from 'react';

const getSpanIdsOfAllHatchetSpans = (spanTree: OtelSpanTree): string[] => {
  if (spanTree.spanAttributes?.instrumentor !== 'hatchet') {
    return [];
  }

  return [
    spanTree.spanId,
    ...spanTree.children.flatMap(getSpanIdsOfAllHatchetSpans),
  ];
};

export function TaskRunTrace({
  spans,
  onTaskRunClick,
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

  const [expandedSpansIds, setExpandedSpansIds] = useState<string[]>(
    spanTrees.flatMap(getSpanIdsOfAllHatchetSpans),
  );

  const [selectedSpan, setSelectedSpan] = useState<OtelSpanTree | undefined>();

  const handleSpanSelect = useCallback(
    (span: OtelSpanTree) => {
      setSelectedSpan(span);
      const stepRunId = span.spanAttributes?.['hatchet.step_run_id'];
      if (stepRunId && onTaskRunClick) {
        onTaskRunClick(stepRunId);
      }
    },
    [onTaskRunClick],
  );

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
    <div className="my-4 min-w-0 overflow-hidden">
      <TraceTimeline
        spanTrees={spanTrees}
        expandedSpanIds={expandedSpansIds}
        onExpandChange={setExpandedSpansIds}
        selectedSpan={selectedSpan}
        onSpanSelect={handleSpanSelect}
      />
    </div>
  );
}
