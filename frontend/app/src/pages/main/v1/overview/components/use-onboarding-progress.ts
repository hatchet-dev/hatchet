import {
  hasQualifiedWorker,
  qualifiedRunQueryParams,
} from './onboarding-state';
import useControlPlane from '@/hooks/use-control-plane';
import { queries, WorkerStatus } from '@/lib/api';
import { emptyGolangUUID } from '@/lib/utils';
import { useQuery } from '@tanstack/react-query';
import { useRef } from 'react';

const POLL_INTERVAL_MS = 2000;
const SETUP_POLL_INTERVAL_MS = 20000;

// Whether the tenant is set up at all: it has an active worker AND has ever
// completed a run. Unlike useOnboardingProgress, this is NOT scoped to the
// onboarding-selection timestamp, so the Overview re-entry banner reflects the
// tenant's real state (a tenant with a live worker and past runs is set up,
// even if it never went through the new onboarding).
export function useTenantOnboarded(tenantId: string | undefined): boolean {
  const { isSelfHosted } = useControlPlane();
  const enabled = !!tenantId;

  const workersQuery = useQuery({
    ...queries.workers.list(tenantId ?? ''),
    enabled,
    refetchInterval: SETUP_POLL_INTERVAL_MS,
  });

  const completedRunQuery = useQuery({
    // Reuse the completed-run query shape with an epoch "since" so it matches
    // any completed run, ever (not scoped to the onboarding selection).
    ...queries.v1WorkflowRuns.list(
      tenantId ?? '',
      qualifiedRunQueryParams(new Date(0).toISOString()),
      isSelfHosted,
    ),
    enabled,
    refetchInterval: SETUP_POLL_INTERVAL_MS,
  });

  const hasActiveWorker = (workersQuery.data?.rows ?? []).some(
    (worker) => worker.status === WorkerStatus.ACTIVE,
  );
  const hasCompletedRun =
    completedRunQuery.data !== 'timeout' &&
    (completedRunQuery.data?.rows?.length ?? 0) > 0;

  return hasActiveWorker && hasCompletedRun;
}

export type OnboardingProgress = {
  // An ACTIVE worker registered after the confirmed selection exists.
  workerConnected: boolean;
  // A COMPLETED run created after the confirmed selection exists.
  runCompleted: boolean;
  // Both of the above. This is the single "successfully onboarded" signal.
  onboarded: boolean;
  // The first qualifying completed task, once one exists: its run id (for a
  // link to the task run) and the id of the worker that actually executed it
  // (for a link to that specific worker, not just any connected worker).
  completedRun?: { runId: string; workerId?: string };
  isLoading: boolean;
};

// Shared onboarding completion detection, extracted so both the onboarding
// modal (live status) and the Overview re-entry surfaces can consume it
// without duplicating the worker/run qualification logic. The detection
// itself lives in onboarding-state.ts and is reused verbatim here.
//
// Polls workers and runs every 2s while onboarding is incomplete, then
// stops once both conditions are met. The run query is only enabled once a
// selection has been confirmed, mirroring the inline flow: without a
// confirmation timestamp nothing can qualify.
export function useOnboardingProgress(
  tenantId: string | undefined,
  selectionConfirmedAt: string | undefined,
): OnboardingProgress {
  const { isSelfHosted } = useControlPlane();

  // Written during render below so the refetchInterval callbacks can read
  // the latest combined state and stop polling once onboarded.
  const onboardedRef = useRef(false);

  const enabled = !!tenantId && !!selectionConfirmedAt;

  const workersQuery = useQuery({
    ...queries.workers.list(tenantId ?? ''),
    enabled,
    refetchInterval: () => (onboardedRef.current ? false : POLL_INTERVAL_MS),
  });

  const qualifiedRunQuery = useQuery({
    ...queries.v1WorkflowRuns.list(
      tenantId ?? '',
      qualifiedRunQueryParams(
        selectionConfirmedAt ?? new Date(0).toISOString(),
      ),
      isSelfHosted,
    ),
    enabled,
    refetchInterval: () => (onboardedRef.current ? false : POLL_INTERVAL_MS),
  });

  const workerConnected = hasQualifiedWorker(
    workersQuery.data?.rows ?? [],
    selectionConfirmedAt ?? null,
  );

  const runCompleted =
    !!selectionConfirmedAt &&
    qualifiedRunQuery.data !== 'timeout' &&
    (qualifiedRunQuery.data?.rows?.length ?? 0) > 0;

  const onboarded = workerConnected && runCompleted;
  onboardedRef.current = onboarded;

  // Once a run has completed, fetch the completed TASK (only_tasks: true) so we
  // can surface the specific run and the worker that executed it. This is a
  // separate query from the completion check so that detection semantics stay
  // unchanged; it is only enabled once runCompleted, and stops polling after
  // it resolves a row.
  const completedTaskQuery = useQuery({
    ...queries.v1WorkflowRuns.list(
      tenantId ?? '',
      {
        ...qualifiedRunQueryParams(
          selectionConfirmedAt ?? new Date(0).toISOString(),
        ),
        only_tasks: true,
      },
      isSelfHosted,
    ),
    enabled: enabled && runCompleted,
    refetchInterval: POLL_INTERVAL_MS,
  });

  const completedTaskRow =
    completedTaskQuery.data !== 'timeout'
      ? completedTaskQuery.data?.rows?.[0]
      : undefined;
  const completedRunId = completedTaskRow?.metadata.id;

  // The task summary does not carry the worker, so resolve the worker that
  // actually executed this run from its task events (the same source the run
  // detail page uses to link workers). Empty-UUID events are skipped.
  const runDetailsQuery = useQuery({
    ...queries.v1WorkflowRuns.details(completedRunId ?? ''),
    enabled: enabled && !!completedRunId,
  });
  const completedWorkerId = (runDetailsQuery.data?.taskEvents ?? [])
    .map((event) => event.workerId)
    .find(
      (workerId): workerId is string =>
        !!workerId && workerId !== emptyGolangUUID,
    );

  const completedRun = completedRunId
    ? { runId: completedRunId, workerId: completedWorkerId }
    : undefined;

  return {
    workerConnected,
    runCompleted,
    onboarded,
    completedRun,
    isLoading:
      enabled && (workersQuery.isLoading || qualifiedRunQuery.isLoading),
  };
}
