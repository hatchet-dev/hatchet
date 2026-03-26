import { useRunDetailSearch } from '../../../../hooks/use-run-detail-search';
import { TaskRunTrace } from './task-run-trace';
import { hasOnlyEngineSpans, isQueuedOnlyRoot } from './utils/span-tree-utils';
import {
  ATTR,
  SPAN,
  convertOtelSpansToOtelSpanTree,
  findSubtreeByTaskRunId,
  type TaskSummaryForSynthesis,
} from '@/components/v1/agent-prism/convert-otel-spans-to-agent-prism-span-tree';
import type {
  OtelSpanTree,
  RelevantOpenTelemetrySpanProperties,
} from '@/components/v1/agent-prism/span-tree-type';
import {
  filterSpanTrees,
  parseTraceQuery,
  TraceSearchInput,
  type TraceAutocompleteContext,
} from '@/components/v1/cloud/observability/trace-search';
import { DocsButton } from '@/components/v1/docs/docs-button';
import { Loading } from '@/components/v1/ui/loading';
import { OnboardingCard } from '@/components/v1/ui/onboarding-card';
import api from '@/lib/api/api';
import { docsPages } from '@/lib/generated/docs';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import useCloud from '@/pages/auth/hooks/use-cloud';
import { appRoutes } from '@/router';
import { useQuery } from '@tanstack/react-query';
import { useParams } from '@tanstack/react-router';
import { Activity } from 'lucide-react';
import { useCallback, useEffect, useMemo, useState } from 'react';

function hasAtLeastOneElement<T>(arr: T[]): arr is [T, ...T[]] {
  return arr.length > 0;
}

function pruneOtherWorkflowRuns(
  node: OtelSpanTree,
  currentRunId: string,
): OtelSpanTree {
  const filteredChildren = node.children
    .filter((child) => {
      if (child.spanName === SPAN.ENGINE_WORKFLOW_RUN) {
        const childRunId = child.spanAttributes?.[ATTR.WORKFLOW_RUN_ID];
        return childRunId === currentRunId;
      }
      return true;
    })
    .map((child) => pruneOtherWorkflowRuns(child, currentRunId));
  return { ...node, children: filteredChildren };
}

const GO_ZERO_TIME = '0001-01-01T00:00:00Z';

const pickSpan = (
  span: RelevantOpenTelemetrySpanProperties,
): RelevantOpenTelemetrySpanProperties => ({
  spanId: span.spanId,
  parentSpanId: span.parentSpanId,
  spanName: span.spanName,
  statusCode: span.statusCode,
  statusMessage: span.statusMessage,
  durationNs: span.durationNs,
  createdAt: span.createdAt,
  spanAttributes: span.spanAttributes,
});

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
  const nameSet = new Set<string>();

  for (const span of spans) {
    if (span.spanName) {
      nameSet.add(span.spanName);
    }
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
    spanNames: [...nameSet].sort(),
  };
}

export const Observability = (props: ObservabilityProps) => {
  const { isRunning, tasks, workflowRunCreatedAt, workflowRunStartedAt } =
    props;

  const runExternalId = props.taskRunId ?? props.workflowRunExternalId;
  const { tenant } = useParams({ from: appRoutes.tenantRoute.to });
  const { isCloudEnabled } = useCloud(tenant);
  const { meta } = useApiMeta();

  const { queryString, setQueryString } = useRunDetailSearch();

  const GRACE_PERIOD_MS = 15_000;
  const [showInContext, setShowInContext] = useState(false);
  const [inGracePeriod, setInGracePeriod] = useState(false);
  useEffect(() => {
    if (isRunning) {
      setInGracePeriod(false);
      return;
    }
    setInGracePeriod(true);
    const timer = setTimeout(() => setInGracePeriod(false), GRACE_PERIOD_MS);
    return () => clearTimeout(timer);
  }, [isRunning]);

  const tracesQuery = useQuery({
    queryKey: [tenant, runExternalId],
    queryFn: async () => {
      const res = await api.v1ObservabilityGetTrace(tenant, {
        run_external_id: runExternalId,
        limit: 1_000,
      });

      return res.data?.rows?.map(pickSpan);
    },
    refetchInterval: isRunning || inGracePeriod ? 5000 : false,
  });

  const traces = tracesQuery.data;

  const autocompleteContext = useMemo(
    () => buildAutocompleteContext(traces ?? []),
    [traces],
  );

  const hasMultipleWorkflowRuns = useMemo(() => {
    if (!traces) {
      return false;
    }
    const runIds = new Set<string>();
    for (const span of traces) {
      const id = span.spanAttributes?.[ATTR.WORKFLOW_RUN_ID];
      if (id) {
        runIds.add(id);
      }
      if (runIds.size > 1) {
        return true;
      }
    }
    return false;
  }, [traces]);

  const showContextToggle = !!props.taskRunId || hasMultipleWorkflowRuns;

  const workflowRunTiming = useMemo(() => {
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
  }, [workflowRunCreatedAt, workflowRunStartedAt]);

  const spanTrees = useMemo(() => {
    let trees: ReturnType<typeof convertOtelSpansToOtelSpanTree> | null = null;
    const hasTraceRows = !!(traces && hasAtLeastOneElement(traces));
    const timingForSynthesis = hasTraceRows ? undefined : workflowRunTiming;
    const convertOptions = {
      enableTraceInProgressSynthesis: !hasTraceRows,
    };

    if (hasTraceRows) {
      trees = convertOtelSpansToOtelSpanTree(
        traces,
        tasks,
        timingForSynthesis,
        convertOptions,
      );
    } else {
      const pendingTasks = tasks?.filter(
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

    if (!showInContext) {
      if (props.taskRunId) {
        const subtree = findSubtreeByTaskRunId(trees, props.taskRunId);
        if (subtree) {
          subtree.inProgress = isRunning && !isQueuedOnlyRoot(subtree);
          return [subtree];
        }
      } else if (props.workflowRunExternalId) {
        trees = trees.map((t) =>
          pruneOtherWorkflowRuns(t, props.workflowRunExternalId),
        );
      }
    }

    trees[0].inProgress = isRunning && !isQueuedOnlyRoot(trees[0]);

    return trees;
  }, [
    traces,
    workflowRunTiming,
    showInContext,
    props.taskRunId,
    props.workflowRunExternalId,
    isRunning,
    tasks,
  ]);

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

  const handleAddFilter = useCallback(
    (key: string, value: string) => {
      const token = `${key}:${value}`;
      const trimmed = queryString.trim();
      setQueryString(trimmed ? `${trimmed} ${token}` : token);
    },
    [queryString, setQueryString],
  );

  const handleRemoveFilter = useCallback(
    (key: string, value: string) => {
      const token = `${key}:${value}`;
      const parts = queryString.split(/\s+/).filter((p) => p !== token);
      setQueryString(parts.join(' '));
    },
    [queryString, setQueryString],
  );

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
        {spanTrees ? (
          <div className="py-4 text-sm text-muted-foreground">
            No spans match the current filter.
          </div>
        ) : !isCloudEnabled && !meta?.observabilityEnabled ? (
          <OnboardingCard
            icon={<Activity className="size-4" />}
            title="Enable Observability"
            description={
              <>
                Trace collection is not enabled on this instance. Set{' '}
                <code className="rounded bg-muted px-1 py-0.5 text-xs">
                  SERVER_OBSERVABILITY_ENABLED=true
                </code>{' '}
                in your server configuration to start collecting traces.
              </>
            }
            actions={
              <DocsButton
                doc={docsPages['self-hosting']['configuration-options']}
                label="View setup guide"
                variant="text"
              />
            }
          />
        ) : (
          <OnboardingCard
            icon={<Activity className="size-4" />}
            title="No traces found"
            description={
              <>
                To collect traces, use the{' '}
                <code className="rounded bg-muted px-1 py-0.5 text-xs">
                  HatchetInstrumentor
                </code>{' '}
                in your SDK.
              </>
            }
            actions={
              <DocsButton
                doc={docsPages.v1.opentelemetry}
                label="View instrumentation docs"
                variant="text"
              />
            }
          />
        )}
      </div>
    );
  }

  const onlyEngineSpans = !isRunning && hasOnlyEngineSpans(filteredTrees);

  return (
    <div className="flex flex-col gap-4">
      <TraceSearchInput
        value={queryString}
        onChange={setQueryString}
        autocompleteContext={autocompleteContext}
      />
      {onlyEngineSpans && (
        <OnboardingCard
          variant="info"
          dismissible
          dismissKey="hatchet:dismiss-traces-enrichment-hint"
          icon={<Activity className="size-4" />}
          title="Enrich your traces"
          description={
            <>
              These traces only contain engine-generated spans. Add the{' '}
              <code className="rounded bg-muted px-1 py-0.5 text-xs">
                HatchetInstrumentor
              </code>{' '}
              to your SDK to capture custom spans from your application code.
            </>
          }
          actions={
            <DocsButton
              doc={docsPages.v1.opentelemetry}
              label="View instrumentation docs"
              variant="text"
            />
          }
        />
      )}
      <TaskRunTrace
        spanTrees={filteredTrees}
        isRunning={isRunning}
        activeFilters={parsedQuery}
        onAddFilter={handleAddFilter}
        onRemoveFilter={handleRemoveFilter}
        showInContext={showContextToggle ? showInContext : undefined}
        onToggleShowInContext={
          showContextToggle ? () => setShowInContext((v) => !v) : undefined
        }
        contextTaskRunId={props.taskRunId}
        onClearFilters={queryString ? () => setQueryString('') : undefined}
      />
    </div>
  );
};
