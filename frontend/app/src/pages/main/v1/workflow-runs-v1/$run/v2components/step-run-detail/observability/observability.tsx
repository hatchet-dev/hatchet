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

const PAGE_SIZE = 200;

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

async function fetchAllSpansByTask(
  tenantId: string,
  taskExternalId: string,
): Promise<RelevantOpenTelemetrySpanProperties[]> {
  const allSpans: RelevantOpenTelemetrySpanProperties[] = [];
  let currentPage = 0;
  let numPages = 1;

  do {
    const res = await api.v1WorkflowRunGetTrace(tenantId, {
      runExternalId: taskExternalId,
      offset: currentPage * PAGE_SIZE,
      limit: PAGE_SIZE,
    });

    const rows = res.data.rows ?? [];
    allSpans.push(...rows.map(pickSpan));

    numPages = res.data.pagination?.num_pages ?? 1;
    currentPage = res.data.pagination?.current_page ?? 1;
  } while (currentPage < numPages);

  return allSpans;
}

async function fetchAllSpansByWorkflowRun(
  tenantId: string,
  workflowRunExternalId: string,
): Promise<RelevantOpenTelemetrySpanProperties[]> {
  const allSpans: RelevantOpenTelemetrySpanProperties[] = [];
  let currentPage = 0;
  let numPages = 1;

  do {
    const res = await api.v1WorkflowRunGetTrace(tenantId, {
      runExternalId: workflowRunExternalId,
      offset: currentPage * PAGE_SIZE,
      limit: PAGE_SIZE,
    });

    const rows = res.data.rows ?? [];
    allSpans.push(...rows.map(pickSpan));

    numPages = res.data.pagination?.num_pages ?? 1;
    currentPage = res.data.pagination?.current_page ?? 1;
  } while (currentPage < numPages);

  return allSpans;
}

type ObservabilityProps = {
  isRunning: boolean;
  onTaskRunClick?: (taskRunId: string) => void;
} & (
  | { taskRunId: string; workflowRunExternalId?: never }
  | { taskRunId?: never; workflowRunExternalId: string }
);

export const Observability = (props: ObservabilityProps) => {
  const { isRunning } = props;

  const queryId = props.taskRunId ?? props.workflowRunExternalId;
  const queryType = props.taskRunId ? 'task' : 'workflow-run';
  const { tenant } = useParams({ from: appRoutes.tenantRoute.to });

  const tracesQuery = useQuery({
    queryKey: [tenant, queryType + ':trace', queryId],
    queryFn: () =>
      queryType === 'task'
        ? fetchAllSpansByTask(tenant, queryId)
        : fetchAllSpansByWorkflowRun(tenant, queryId),
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
