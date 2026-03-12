import { TreeView } from '@/components/v1/agent-prism/TreeView';
import { convertOtelSpansToOtelSpanTree } from '@/components/v1/agent-prism/convert-otel-spans-to-agent-prism-span-tree';
import type { OtelSpanTree } from '@/components/v1/agent-prism/span-tree-type';
import { OtelSpan } from '@/lib/api/generated/data-contracts';
import { useEffect, useMemo, useState } from 'react';

const getSpanIdsOfAllHatchetSpans = (spanTree: OtelSpanTree): string[] => {
  if (!spanTree.spanName.startsWith('hatchet.')) {
    return [];
  }

  return [
    spanTree.spanId,
    ...spanTree.children.flatMap(getSpanIdsOfAllHatchetSpans),
  ];
};

export function TaskRunTrace({ spans }: { spans: [OtelSpan, ...OtelSpan[]] }) {
  const traceSpanTree = useMemo(
    () => convertOtelSpansToOtelSpanTree(spans),
    [spans],
  );

  const [expandedSpansIds, setExpandedSpansIds] = useState<string[]>([]);

  useEffect(() => {
    setExpandedSpansIds(getSpanIdsOfAllHatchetSpans(traceSpanTree));
  }, [traceSpanTree]);

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
      <div className="max-h-[500px] overflow-y-auto">
        <TreeView
          spanTree={traceSpanTree}
          expandedSpansIds={expandedSpansIds}
          onExpandSpansIdsChange={setExpandedSpansIds}
        />
      </div>
    </div>
  );
}
