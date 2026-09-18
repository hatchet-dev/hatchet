import {
  hasQualifiedWorker,
  qualifiedRunQueryParams,
} from './onboarding-state';
import useControlPlane from '@/hooks/use-control-plane';
import { queries } from '@/lib/api';
import { emptyGolangUUID } from '@/lib/utils';
import { useQuery } from '@tanstack/react-query';
import { useEffect, useRef } from 'react';

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
    // Poll only while the tenant is not yet set up; stop once it is.
    refetchInterval: (query) => {
      const data = query.state.data;
      const done = data !== 'timeout' && (data?.rows?.length ?? 0) > 0;
      return done ? false : SETUP_POLL_INTERVAL_MS;
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
    // A list timeout (self-hosted) means the state is unknown, not "not set
    // up", so it is reported as still loading rather than as a reason to nudge.
    isLoading:
      !sticky &&
      !!tenantId &&
      (completedRunQuery.isLoading || data === 'timeout'),
  };
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
  // Completion latches for a given tenant + selection: once both conditions
  // were met they stay met, even if the worker then disconnects (agents often
  // stop the worker after the first run). Otherwise Next / Start exploring
  // would disappear again right after appearing.
  const latchKey = `${tenantId ?? ''}:${selectionConfirmedAt ?? ''}`;
  const latchedKeyRef = useRef<string | null>(null);

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

  if (workerConnected && runCompleted) {
    latchedKeyRef.current = latchKey;
  }
  const onboarded = latchedKeyRef.current === latchKey;
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
    // Poll only until the completed task row shows up, then stop.
    refetchInterval: (query) => {
      const data = query.state.data;
      const found = data !== 'timeout' && (data?.rows?.length ?? 0) > 0;
      return found ? false : POLL_INTERVAL_MS;
    },
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
