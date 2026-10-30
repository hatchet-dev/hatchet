import {
  hasQualifiedWorker,
  qualifiedRunQueryParams,
} from './onboarding-state';
import useControlPlane from '@/hooks/use-control-plane';
import { queries, type V1TaskSummaryList } from '@/lib/api';
import { emptyGolangUUID } from '@/lib/utils';
import { useQuery } from '@tanstack/react-query';
import { useEffect } from 'react';

const POLL_INTERVAL_MS = 2000;
const SETUP_POLL_INTERVAL_MS = 5000;

const tenantOnboardedKey = (tenantId: string) =>
  `hatchet:tenant-onboarded:${tenantId}`;

function readStickyOnboarded(tenantId: string | undefined): boolean {
  if (!tenantId) {
    return false;
  }
  try {
    return localStorage.getItem(tenantOnboardedKey(tenantId)) === '1';
  } catch {
    return false;
  }
}

// Whether the tenant is set up at all, for the Overview re-entry banner. A
// tenant counts as set up once it has EVER completed a run: a run cannot
// complete without a worker having connected, so this alone proves the setup
// worked. It deliberately does not require a currently active worker, because
// workers come and go (an agent often stops the worker it started), and the
// nudge must not return every time one disconnects. Unlike
// useOnboardingProgress this is not scoped to the onboarding-selection
// timestamp.
//
// `isLoading` is true until the first answer arrives, so callers can avoid
// flashing the nudge at tenants whose state is simply not known yet. Once a
// tenant is seen as set up, that is remembered per browser: it cannot become
// un-set-up, and run retention may later delete the run that proved it.
export function useTenantOnboarded(tenantId: string | undefined): {
  onboarded: boolean;
  isLoading: boolean;
} {
  const { isSelfHosted } = useControlPlane();
  const sticky = readStickyOnboarded(tenantId);

  const completedRunQuery = useQuery({
    // Reuse the completed-run query shape with an epoch "since" so it matches
    // any completed run, ever (not scoped to the onboarding selection).
    ...queries.v1WorkflowRuns.list(
      tenantId ?? '',
      qualifiedRunQueryParams(new Date(0).toISOString()),
      isSelfHosted,
    ),
    enabled: !!tenantId && !sticky,
    // Poll only while the answer is a definite "not yet": stop once a run
    // exists, and do not keep re-issuing a list that already timed out.
    refetchInterval: (query) => {
      const data = query.state.data;
      const settled = data === 'timeout' || (data?.rows?.length ?? 0) > 0;
      return settled ? false : SETUP_POLL_INTERVAL_MS;
    },
  });

  const data = completedRunQuery.data;
  const hasCompletedRun = data !== 'timeout' && (data?.rows?.length ?? 0) > 0;

  useEffect(() => {
    if (!tenantId || !hasCompletedRun) {
      return;
    }
    try {
      localStorage.setItem(tenantOnboardedKey(tenantId), '1');
    } catch {
      // Storage can be unavailable (private mode); the API check still works.
    }
  }, [tenantId, hasCompletedRun]);

  return {
    onboarded: sticky || hasCompletedRun,
    // A failed or timed-out list means the state is unknown, not "not set
    // up", so it is reported as still loading rather than as a reason to nudge.
    isLoading:
      !sticky &&
      !!tenantId &&
      (completedRunQuery.isLoading ||
        completedRunQuery.isError ||
        data === 'timeout'),
  };
}

export type OnboardingProgress = {
  // A worker registered after the confirmed selection is ACTIVE, or the run
  // below already completed (which proves one connected, even if it has since
  // stopped: agents often stop the worker they started).
  workerConnected: boolean;
  // A COMPLETED run created after the confirmed selection exists. This is the
  // lasting "successfully onboarded" signal: unlike worker status it never
  // reverts, so it survives a refresh and a disconnected worker.
  runCompleted: boolean;
  // The first qualifying completed task, once one exists: its run id (for a
  // link to the task run) and the id of the worker that actually executed it
  // (for a link to that specific worker, not just any connected worker).
  completedRun?: { runId: string; workerId?: string };
};

// Live onboarding completion detection for the onboarding overlay. Polls
// workers and runs every 2s until a qualifying run completes, then stops.
// Nothing can qualify before a selection is confirmed, so the queries stay
// disabled until then.
export function useOnboardingProgress(
  tenantId: string | undefined,
  selectionConfirmedAt: string | undefined,
): OnboardingProgress {
  const { isSelfHosted } = useControlPlane();

  const enabled = !!tenantId && !!selectionConfirmedAt;
  const since = selectionConfirmedAt ?? new Date(0).toISOString();

  const hasRows = (data: V1TaskSummaryList | 'timeout' | undefined) =>
    data !== 'timeout' && (data?.rows?.length ?? 0) > 0;

  const qualifiedRunQuery = useQuery({
    ...queries.v1WorkflowRuns.list(
      tenantId ?? '',
      qualifiedRunQueryParams(since),
      isSelfHosted,
    ),
    enabled,
    refetchInterval: (query) =>
      hasRows(query.state.data) ? false : POLL_INTERVAL_MS,
  });
  const runCompleted = enabled && hasRows(qualifiedRunQuery.data);

  const workersQuery = useQuery({
    ...queries.workers.list(tenantId ?? ''),
    enabled,
    refetchInterval: runCompleted ? false : POLL_INTERVAL_MS,
  });
  const workerConnected =
    runCompleted ||
    hasQualifiedWorker(
      workersQuery.data?.rows ?? [],
      selectionConfirmedAt ?? null,
    );

  // The completed TASK (only_tasks: true) identifies the specific run to link
  // to. It is a separate query so completion detection above keeps its
  // workflow-level semantics.
  const completedTaskQuery = useQuery({
    ...queries.v1WorkflowRuns.list(
      tenantId ?? '',
      { ...qualifiedRunQueryParams(since), only_tasks: true },
      isSelfHosted,
    ),
    enabled: runCompleted,
    refetchInterval: (query) =>
      hasRows(query.state.data) ? false : POLL_INTERVAL_MS,
  });
  const completedRunId =
    completedTaskQuery.data !== 'timeout'
      ? completedTaskQuery.data?.rows?.[0]?.metadata.id
      : undefined;

  // The task summary does not carry the worker, so resolve the worker that
  // actually executed this run from its task events (the same source the run
  // detail page uses to link workers). Empty-UUID events are skipped.
  const runDetailsQuery = useQuery({
    ...queries.v1WorkflowRuns.details(completedRunId ?? ''),
    enabled: !!completedRunId,
  });
  const completedWorkerId = (runDetailsQuery.data?.taskEvents ?? [])
    .map((event) => event.workerId)
    .find(
      (workerId): workerId is string =>
        !!workerId && workerId !== emptyGolangUUID,
    );

  return {
    workerConnected,
    runCompleted,
    completedRun: completedRunId
      ? { runId: completedRunId, workerId: completedWorkerId }
      : undefined,
  };
}
