import {
  hasQualifiedWorker,
  qualifiedRunQueryParams,
} from './onboarding-state';
import useControlPlane from '@/hooks/use-control-plane';
import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useRef } from 'react';

const POLL_INTERVAL_MS = 2000;

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
