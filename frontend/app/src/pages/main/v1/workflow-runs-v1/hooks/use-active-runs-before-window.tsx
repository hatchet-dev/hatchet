import { FilterActions } from './use-runs-table-filters';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { queries, V1TaskStatus } from '@/lib/api';
import { withPolling } from '@/lib/api/polling';
import { getRetentionBoundary } from '@/lib/utils/retention';
import { useQuery } from '@tanstack/react-query';
import { useCallback, useMemo } from 'react';

const ACTIVE_STATUSES = [V1TaskStatus.QUEUED, V1TaskStatus.RUNNING];

const MIN_REFETCH_INTERVAL_MS = 60 * 1000;

// The boundary is rounded so the query key only changes once per minute.
const BOUNDARY_ROUNDING_MS = 60 * 1000;

// A window starting this close to the retention boundary already shows
// everything that is retained.
const RETENTION_TOLERANCE_MS = 5 * 60 * 1000;

export type ActiveRunsBeforeWindow = {
  count: number;
  capped: boolean;
  showActiveRuns: () => void;
};

export const useActiveRunsBeforeWindow = ({
  enabled,
  filters,
}: {
  enabled: boolean;
  filters: FilterActions;
}): ActiveRunsBeforeWindow | null => {
  const { tenantId } = useCurrentTenantId();
  const { refetchInterval } = useRefetchInterval();
  const { apiFilters, retentionPeriod, showActiveRunsSince } = filters;

  const boundaryMs = retentionPeriod
    ? getRetentionBoundary(retentionPeriod)?.getTime()
    : undefined;
  const retentionStart =
    boundaryMs === undefined
      ? undefined
      : new Date(
          Math.ceil(boundaryMs / BOUNDARY_ROUNDING_MS) * BOUNDARY_ROUNDING_MS,
        ).toISOString();

  const windowReachesRetention =
    boundaryMs !== undefined &&
    new Date(apiFilters.since).getTime() - boundaryMs <= RETENTION_TOLERANCE_MS;

  const statuses = useMemo(
    () =>
      (apiFilters.statuses ?? ACTIVE_STATUSES).filter((status) =>
        ACTIVE_STATUSES.includes(status),
      ),
    [apiFilters.statuses],
  );

  const isEnabled =
    enabled &&
    retentionStart !== undefined &&
    !windowReachesRetention &&
    statuses.length > 0;

  const query = useQuery({
    ...withPolling(
      queries.v1WorkflowRuns.activeCount(tenantId, {
        since: retentionStart ?? '',
        before: apiFilters.since,
        statuses,
        running_filter: apiFilters.runningFilter,
        workflow_ids: apiFilters.workflowIds,
        additional_metadata: apiFilters.additionalMetadata,
        additional_metadata_operator: apiFilters.additionalMetadataOperator,
        idempotency_keys: apiFilters.idempotencyKeys,
      }),
      refetchInterval === false
        ? false
        : Math.max(refetchInterval, MIN_REFETCH_INTERVAL_MS),
    ),
    enabled: isEnabled,
    placeholderData: (prev) => prev,
  });

  const showActiveRuns = useCallback(() => {
    if (retentionStart) {
      showActiveRunsSince(retentionStart, statuses);
    }
  }, [retentionStart, showActiveRunsSince, statuses]);

  if (!isEnabled || !query.data) {
    return null;
  }

  return {
    count: query.data.count,
    capped: query.data.capped,
    showActiveRuns,
  };
};
