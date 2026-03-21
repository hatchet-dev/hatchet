import { TaskRunTrace } from './task-run-trace';
import type { RelevantOpenTelemetrySpanProperties } from '@/components/v1/agent-prism/span-tree-type';
import { Loading } from '@/components/v1/ui/loading';
import api from '@/lib/api/api';
import { appRoutes } from '@/router';
import { useQuery } from '@tanstack/react-query';
import { useParams } from '@tanstack/react-router';

function hasAtLeastOneElement<T>(arr: T[]): arr is [T, ...T[]] {
  return arr.length > 0;
}

const pickSpan = (
  span: RelevantOpenTelemetrySpanProperties,
): RelevantOpenTelemetrySpanProperties => ({
  spanId: span.spanId,
  parentSpanId: span.parentSpanId,
  spanName: span.spanName,
  statusCode: span.statusCode,
  durationNs: span.durationNs,
  createdAt: span.createdAt,
  spanAttributes: span.spanAttributes,
});

type ObservabilityProps = {
  isRunning: boolean;
  onTaskRunClick?: (taskRunId: string) => void;
} & (
  | { taskRunId: string; workflowRunExternalId?: never }
  | { taskRunId?: never; workflowRunExternalId: string }
);

export const Observability = (props: ObservabilityProps) => {
  const { isRunning } = props;

  const runExternalId = props.taskRunId ?? props.workflowRunExternalId;
  const { tenant } = useParams({ from: appRoutes.tenantRoute.to });

  const tracesQuery = useQuery({
    queryKey: [tenant, runExternalId],
    queryFn: async () => {
      const res = await api.v1ObservabilityGetTrace(tenant, {
        run_external_id: runExternalId,
        limit: 1_000, // arbitrary limit that allows a lot of spans to come back. if we're fetching more than this, it's probably an issue
      });

      return res.data?.rows?.map(pickSpan);
    },
    refetchInterval: isRunning ? 5000 : false,
  });

  if (!tracesQuery.isFetched) {
    return <Loading />;
  }

  const traces = tracesQuery.data;

  if (!traces || !hasAtLeastOneElement(traces)) {
    return (
      <div className="py-4 text-sm text-muted-foreground">
        No traces found. To collect traces, use the{' '}
        <code className="rounded bg-muted px-1 py-0.5 text-xs">
          HatchetInstrumentor
        </code>{' '}
        in your SDK.
      </div>
    );
  }

  return <TaskRunTrace spans={traces} onTaskRunClick={props.onTaskRunClick} />;
};
