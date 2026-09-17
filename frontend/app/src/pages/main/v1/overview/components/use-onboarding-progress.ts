import {
  hasQualifiedWorker,
  qualifiedRunQueryParams,
} from './onboarding-state';
import useControlPlane from '@/hooks/use-control-plane';
import { queries, WorkerStatus } from '@/lib/api';
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

  return {
    workerConnected,
    runCompleted,
    onboarded,
    isLoading:
      enabled && (workersQuery.isLoading || qualifiedRunQuery.isLoading),
  };
}
