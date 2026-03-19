import { TaskRunTrace } from './task-run-trace';
import { filterSpanTrees } from './trace-search/filter';
import { parseTraceQuery } from './trace-search/parser';
import { TraceSearchInput } from './trace-search/trace-search-input';
import type { TraceAutocompleteContext } from './trace-search/types';
import {
  convertOtelSpansToOtelSpanTree,
  type TaskSummaryForSynthesis,
} from '@/components/v1/agent-prism/convert-otel-spans-to-agent-prism-span-tree';
import type { RelevantOpenTelemetrySpanProperties } from '@/components/v1/agent-prism/span-tree-type';
import { Loading } from '@/components/v1/ui/loading';
import api from '@/lib/api/api';
import { useQuery } from '@tanstack/react-query';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

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
  const { isRunning, tasks } = props;

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

  useEffect(() => {
    if (!traces || traces.length === 0) {
      return;
    }

    const engine: RelevantOpenTelemetrySpanProperties[] = [];
    const sdk: RelevantOpenTelemetrySpanProperties[] = [];

    for (const span of traces) {
      if (span.spanAttributes?.['hatchet.span_source'] === 'engine') {
        engine.push(span);
      } else {
        sdk.push(span);
      }
    }

    const lines: string[] = [];
    lines.push(
      `[trace-debug] poll — ${traces.length} spans (${engine.length} engine, ${sdk.length} sdk)`,
    );
    for (const s of traces) {
      const src =
        s.spanAttributes?.['hatchet.span_source'] === 'engine' ? 'E' : 'S';
      lines.push(
        `  ${src} ${s.spanName} task_name=${s.spanAttributes?.['hatchet.task_name'] ?? ''} step_name=${s.spanAttributes?.['hatchet.step_name'] ?? ''} step_run_id=${s.spanAttributes?.['hatchet.step_run_id'] ?? ''} parent=${s.parentSpanId ?? ''} status=${s.statusCode}`,
      );
    }
    console.log(lines.join('\n'));
  }, [traces]);

  const autocompleteContext = useMemo(
    () => buildAutocompleteContext(traces ?? []),
    [traces],
  );

  const spanTrees = useMemo(() => {
    let trees: ReturnType<typeof convertOtelSpansToOtelSpanTree> | null = null;

    if (traces && hasAtLeastOneElement(traces)) {
      trees = convertOtelSpansToOtelSpanTree(traces, tasks);
    } else {
      const pendingTasks = tasks?.filter(
        (t) => t.status === 'QUEUED' || t.status === 'RUNNING',
      );
      if (pendingTasks && pendingTasks.length > 0) {
        trees = convertOtelSpansToOtelSpanTree(undefined, pendingTasks);
      }
    }

    if (trees && trees.length > 0) {
      trees[0].inProgress = isRunning;
    }

    return trees;
  }, [traces, tasks, isRunning]);

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
