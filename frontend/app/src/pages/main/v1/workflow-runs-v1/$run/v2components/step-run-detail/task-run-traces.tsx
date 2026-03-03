import { openTelemetrySpanAdapter } from '@evilmartians/agent-prism-data';
import { flattenSpans } from '@evilmartians/agent-prism-data';
import { TreeView } from '@/components/v1/agent-prism/TreeView';
import { cloudApi } from '@/lib/api/api';
import { Loading } from '@/components/v1/ui/loading';
import { useQuery } from '@tanstack/react-query';
import { useEffect, useMemo, useState } from 'react';
import { convertOtelSpans } from './otel-span-adapter';

export function TaskRunTraces({
  taskExternalId,
}: {
  taskExternalId: string;
}) {
  const tracesQuery = useQuery({
    queryKey: ['cloud:traces', taskExternalId],
    queryFn: async () => {
      const res = await cloudApi.otelTracesList(taskExternalId);
      return res.data;
    },
  });

  const traceSpans = useMemo(() => {
    const rows = tracesQuery.data?.rows;
    if (!rows || rows.length === 0) return [];

    const otlpSpans = convertOtelSpans(rows);
    return openTelemetrySpanAdapter.convertRawSpansToSpanTree(otlpSpans);
  }, [tracesQuery.data]);

  const allIds = useMemo(
    () => flattenSpans(traceSpans).map((s) => s.id),
    [traceSpans],
  );

  const [expandedSpansIds, setExpandedSpansIds] = useState<string[]>([]);

  // Auto-expand all spans when data loads
  useEffect(() => {
    if (allIds.length > 0) {
      setExpandedSpansIds(allIds);
    }
  }, [allIds]);

  if (tracesQuery.isLoading) {
    return <Loading />;
  }

  if (tracesQuery.isError) {
    return (
      <div className="py-4 text-sm text-muted-foreground">
        Failed to load traces.
      </div>
    );
  }

  if (traceSpans.length === 0) {
    return (
      <div className="py-4 text-sm text-muted-foreground">
        No traces found for this workflow run. To collect traces, use the{' '}
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
