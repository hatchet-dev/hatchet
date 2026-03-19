import { TaskRunTrace } from './task-run-trace';
import { filterSpanTrees } from './trace-search/filter';
import { parseTraceQuery } from './trace-search/parser';
import { TraceSearchInput } from './trace-search/trace-search-input';
import type { TraceAutocompleteContext } from './trace-search/types';
import {
  convertOtelSpansToOtelSpanTree,
  type TaskSummaryForSynthesis,
} from '@/components/v1/agent-prism/convert-otel-spans-to-agent-prism-span-tree';
import type {
  OtelSpanTree,
  RelevantOpenTelemetrySpanProperties,
} from '@/components/v1/agent-prism/span-tree-type';
import { Loading } from '@/components/v1/ui/loading';
import api from '@/lib/api/api';
import { useQuery } from '@tanstack/react-query';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

function hasAtLeastOneElement<T>(arr: T[]): arr is [T, ...T[]] {
  return arr.length > 0;
}

function countTreeNodes(nodes: OtelSpanTree[] | null): number {
  if (!nodes) {
    return 0;
  }
  let count = 0;
  const stack = [...nodes];
  while (stack.length > 0) {
    const node = stack.pop()!;
    count += 1;
    if (node.queuedPhase) {
      stack.push(node.queuedPhase);
    }
    if (node.children.length > 0) {
      stack.push(...node.children);
    }
  }
  return count;
}

function summarizeRoots(nodes: OtelSpanTree[] | null): string {
  if (!nodes || nodes.length === 0) {
    return '-';
  }
  return nodes
    .map((n) => {
      const queuedMs = n.queuedPhase
        ? Math.round(n.queuedPhase.durationNs / 1e6)
        : 0;
      return `${n.spanName}[queuedMs=${queuedMs},children=${n.children.length}]`;
    })
    .join(',');
}

function isQueuedOnlyRoot(node: OtelSpanTree): boolean {
  if (!node.spanId.startsWith('__synthetic_') || !node.queuedPhase) {
    return false;
  }
  const hasRunningChild = node.children.some((c) => c.inProgress);
  return !hasRunningChild;
}

const PAGE_SIZE = 200;
const GO_ZERO_TIME = '0001-01-01T00:00:00Z';

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
  let currentPage = 0;
  let numPages = 1;

  do {
    const res = await api.v1TaskGetTrace(taskExternalId, {
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
  workflowRunExternalId: string,
): Promise<RelevantOpenTelemetrySpanProperties[]> {
  const allSpans: RelevantOpenTelemetrySpanProperties[] = [];
  let currentPage = 0;
  let numPages = 1;

  do {
    const res = await api.v1WorkflowRunGetTrace(workflowRunExternalId, {
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
  tasks?: TaskSummaryForSynthesis[];
  workflowRunCreatedAt?: string;
  workflowRunStartedAt?: string;
} & (
  | { taskRunId: string; workflowRunExternalId?: never }
  | { taskRunId?: never; workflowRunExternalId: string }
);

function buildAutocompleteContext(
  spans: RelevantOpenTelemetrySpanProperties[],
): TraceAutocompleteContext {
  const keySet = new Set<string>();
  const valuesByKey = new Map<string, Set<string>>();

  for (const span of spans) {
    if (!span.spanAttributes) {
      continue;
    }
    for (const [key, value] of Object.entries(span.spanAttributes)) {
      keySet.add(key);
      let vals = valuesByKey.get(key);
      if (!vals) {
        vals = new Set();
        valuesByKey.set(key, vals);
      }
      vals.add(value);
    }
  }

  const attributeValues = new Map<string, string[]>();
  for (const [key, vals] of valuesByKey) {
    attributeValues.set(key, [...vals].sort());
  }

  return {
    attributeKeys: [...keySet].sort(),
    attributeValues,
  };
}

export const Observability = (props: ObservabilityProps) => {
  const { isRunning, tasks, workflowRunCreatedAt, workflowRunStartedAt } =
    props;

  const queryId = props.taskRunId ?? props.workflowRunExternalId;
  const queryType = props.taskRunId ? 'task' : 'workflow-run';

  const [queryString, setQueryString] = useState('');

  const completedAtRef = useRef<number | null>(null);
  useEffect(() => {
    if (isRunning) {
      completedAtRef.current = null;
    } else if (!completedAtRef.current) {
      completedAtRef.current = Date.now();
    }
  }, [isRunning]);

  const GRACE_PERIOD_MS = 15_000;
  const inGracePeriod =
    !isRunning &&
    completedAtRef.current !== null &&
    Date.now() - completedAtRef.current < GRACE_PERIOD_MS;

  const tracesQuery = useQuery({
    queryKey: [queryType + ':trace', queryId],
    queryFn: () =>
      queryType === 'task'
        ? fetchAllSpansByTask(queryId)
        : fetchAllSpansByWorkflowRun(queryId),
    refetchInterval: isRunning || inGracePeriod ? 1000 : 10000,
  });

  const traces = tracesQuery.data;

  const autocompleteContext = useMemo(
    () => buildAutocompleteContext(traces ?? []),
    [traces],
  );

  const workflowRunTiming = useMemo(
    () => {
      if (!workflowRunCreatedAt) {
        return undefined;
      }

      const normalizedStartedAt =
        workflowRunStartedAt &&
        workflowRunStartedAt !== GO_ZERO_TIME &&
        !Number.isNaN(new Date(workflowRunStartedAt).getTime())
          ? workflowRunStartedAt
          : undefined;

      return { createdAt: workflowRunCreatedAt, startedAt: normalizedStartedAt };
    },
    [workflowRunCreatedAt, workflowRunStartedAt],
  );

  const spanTrees = useMemo(() => {
    let trees: ReturnType<typeof convertOtelSpansToOtelSpanTree> | null = null;
    const hasTraceRows = !!(traces && hasAtLeastOneElement(traces));
    const timingForSynthesis = hasTraceRows ? undefined : workflowRunTiming;
    const tasksForSynthesis = hasTraceRows ? undefined : tasks;
    const convertOptions = {
      enableTraceInProgressSynthesis: !hasTraceRows,
    };

    if (hasTraceRows) {
      trees = convertOtelSpansToOtelSpanTree(
        traces,
        tasksForSynthesis,
        timingForSynthesis,
        convertOptions,
      );
    } else {
      const pendingTasks = tasksForSynthesis?.filter(
        (t) => t.status === 'QUEUED' || t.status === 'RUNNING',
      );
      if (pendingTasks && pendingTasks.length > 0) {
        trees = convertOtelSpansToOtelSpanTree(
          undefined,
          pendingTasks,
          timingForSynthesis,
          convertOptions,
        );
      }
    }

    if (!trees || trees.length === 0) {
      return null;
    }

    trees[0].inProgress = isRunning && !isQueuedOnlyRoot(trees[0]);

    return trees;
  }, [traces, tasks, isRunning, workflowRunTiming]);

  const parsedQuery = useMemo(
    () => parseTraceQuery(queryString),
    [queryString],
  );

  const filteredTrees = useMemo(() => {
    if (!spanTrees) {
      return null;
    }
    return filterSpanTrees(spanTrees, parsedQuery);
  }, [spanTrees, parsedQuery]);

  useEffect(() => {
    const pendingTasks =
      tasks?.filter((t) => t.status === 'QUEUED' || t.status === 'RUNNING') ??
      [];
    const engineTraceCount =
      traces?.filter((t) => t.spanAttributes?.['hatchet.span_source'] === 'engine')
        .length ?? 0;

    console.log(
      `[trace-debug] input type=${queryType} id=${queryId} traces=${traces?.length ?? 0} engine=${engineTraceCount} tasks=${tasks?.length ?? 0} pending=${pendingTasks.length} isRunning=${isRunning} wrCreated=${workflowRunTiming?.createdAt ?? '-'} wrStarted=${workflowRunTiming?.startedAt ?? '-'}`,
    );
  }, [queryId, queryType, traces, tasks, isRunning, workflowRunTiming]);

  useEffect(() => {
    console.log(
      `[trace-debug] trees spanRoots=${spanTrees?.length ?? 0} spanNodes=${countTreeNodes(spanTrees)} filteredRoots=${filteredTrees?.length ?? 0} filteredNodes=${countTreeNodes(filteredTrees)} query="${queryString}" roots=${summarizeRoots(spanTrees)}`,
    );
  }, [spanTrees, filteredTrees, queryString]);

  const handleAddFilter = useCallback((key: string, value: string) => {
    const token = `${key}:${value}`;
    setQueryString((prev) => {
      const trimmed = prev.trim();
      return trimmed ? `${trimmed} ${token}` : token;
    });
  }, []);

  const handleRemoveFilter = useCallback((key: string, value: string) => {
    const token = `${key}:${value}`;
    setQueryString((prev) => {
      const parts = prev.split(/\s+/).filter((p) => p !== token);
      return parts.join(' ');
    });
  }, []);

  if (!tracesQuery.isFetched) {
    return <Loading />;
  }

  if (!filteredTrees || filteredTrees.length === 0) {
    return (
      <div className="flex flex-col gap-4">
        {spanTrees && (
          <TraceSearchInput
            value={queryString}
            onChange={setQueryString}
            autocompleteContext={autocompleteContext}
          />
        )}
        <div className="py-4 text-sm text-muted-foreground">
          {spanTrees ? (
            'No spans match the current filter.'
          ) : (
            <>
              No traces found. To collect traces, use the{' '}
              <code className="rounded bg-muted px-1 py-0.5 text-xs">
                HatchetInstrumentor
              </code>{' '}
              in your SDK.
            </>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <TraceSearchInput
        value={queryString}
        onChange={setQueryString}
        autocompleteContext={autocompleteContext}
      />
      <TaskRunTrace
        spanTrees={filteredTrees}
        isRunning={isRunning}
        activeFilters={parsedQuery}
        onAddFilter={handleAddFilter}
        onRemoveFilter={handleRemoveFilter}
      />
    </div>
  );
};
