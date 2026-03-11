import { flattenSpans } from './agent-prism-data';
import { convertSpansToTree } from './convert-spans-to-tree';
import { TreeView } from '@/components/v1/agent-prism/TreeView';
import { OtelSpan } from '@/lib/api/generated/data-contracts';
import { useEffect, useMemo, useState } from 'react';

export function TaskRunTrace({ spans }: { spans: OtelSpan[] }) {
  const traceSpans = useMemo(() => convertSpansToTree(spans), [spans]);

  const allIds = useMemo(
    () => flattenSpans(traceSpans).map((s) => s.id),
    [traceSpans],
  );

  const [expandedSpansIds, setExpandedSpansIds] = useState<string[]>([]);

  useEffect(() => {
    if (allIds.length > 0) {
      setExpandedSpansIds(allIds);
    }
  }, [allIds]);

  if (traceSpans.length === 0) {
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
          spans={traceSpans}
          expandedSpansIds={expandedSpansIds}
          onExpandSpansIdsChange={setExpandedSpansIds}
        />
      </div>
    </div>
  );
}
