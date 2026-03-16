import { TreeView } from '@/components/v1/agent-prism/TreeView';
import { convertOtelSpansToOtelSpanTree } from '@/components/v1/agent-prism/convert-otel-spans-to-agent-prism-span-tree';
import type {
  OtelSpanTree,
  RelevantOpenTelemetrySpanProperties,
} from '@/components/v1/agent-prism/span-tree-type';
import { useCallback, useMemo, useState } from 'react';

const getSpanIdsOfAllHatchetSpans = (spanTree: OtelSpanTree): string[] => {
  if (!spanTree.spanName.startsWith('hatchet.')) {
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
  const traceSpanTree = useMemo(
    () => convertOtelSpansToOtelSpanTree(spans),
    [spans],
  );

  const [expandedSpansIds, setExpandedSpansIds] = useState<string[]>(
    getSpanIdsOfAllHatchetSpans(traceSpanTree),
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

  if (!traceSpanTree) {
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
    <div className="my-4">
      <div className="mb-3 flex items-center justify-between">
        <div>
          <h4 className="text-sm font-medium text-foreground">
            OpenTelemetry Traces
          </h4>
        </div>
      </div>
      <div className="overflow-y-auto">
        <TreeView
          spanTree={traceSpanTree}
          expandedSpansIds={expandedSpansIds}
          onExpandSpansIdsChange={setExpandedSpansIds}
          selectedSpan={selectedSpan}
          onSpanSelect={onTaskRunClick ? handleSpanSelect : undefined}
        />
      </div>
    </div>
  );
}
