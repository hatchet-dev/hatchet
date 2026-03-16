import { TaskRunTrace } from './task-run-trace';
import type { RelevantOpenTelemetrySpanProperties } from '@/components/v1/agent-prism/span-tree-type';
import { Loading } from '@/components/v1/ui/loading';
import api from '@/lib/api/api';
import { useQuery } from '@tanstack/react-query';

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
  taskExternalId: string,
): Promise<RelevantOpenTelemetrySpanProperties[]> {
  const allSpans: RelevantOpenTelemetrySpanProperties[] = [];
  let offset = 0;

  // eslint-disable-next-line no-constant-condition
  while (true) {
    const res = await api.v1TaskGetTrace(taskExternalId, {
      offset,
      limit: PAGE_SIZE,
    });

    const rows = res.data.rows ?? [];
    allSpans.push(...rows.map(pickSpan));

    const numPages = res.data.pagination?.num_pages ?? 1;
    const currentPage = res.data.pagination?.current_page ?? 1;

    if (currentPage >= numPages || rows.length === 0) {
      break;
    }

    offset += PAGE_SIZE;
  }

  return allSpans;
}

async function fetchAllSpansByWorkflowRun(
  workflowRunExternalId: string,
): Promise<RelevantOpenTelemetrySpanProperties[]> {
  const allSpans: RelevantOpenTelemetrySpanProperties[] = [];
  let offset = 0;

  // eslint-disable-next-line no-constant-condition
  while (true) {
    const res = await api.v1WorkflowRunGetTrace(workflowRunExternalId, {
      offset,
      limit: PAGE_SIZE,
    });

    const rows = res.data.rows ?? [];
    allSpans.push(...rows.map(pickSpan));

    const numPages = res.data.pagination?.num_pages ?? 1;
    const currentPage = res.data.pagination?.current_page ?? 1;

    if (currentPage >= numPages || rows.length === 0) {
      break;
    }

    offset += PAGE_SIZE;
  }

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

  const tracesQuery = useQuery({
    queryKey: [queryType + ':trace', queryId],
    queryFn: () =>
      queryType === 'task'
        ? fetchAllSpansByTask(queryId)
        : fetchAllSpansByWorkflowRun(queryId),
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
