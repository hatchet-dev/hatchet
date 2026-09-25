import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { queries } from '@/lib/api';
import { withPolling } from '@/lib/api/polling';
import { useQuery } from '@tanstack/react-query';

export const useMetrics = ({
  workflowIds,
  parentTaskExternalId,
  additionalMetadata,
  createdAfter,
  createdBefore,
  showQueueMetrics,
}: {
  workflowIds: string[] | undefined;
  parentTaskExternalId: string | undefined;
  additionalMetadata?: string[] | undefined;
  createdAfter?: string;
  createdBefore?: string;
  showQueueMetrics: boolean;
}) => {
  const { tenantId } = useCurrentTenantId();
  const { refetchInterval } = useRefetchInterval();

  const {
    data: rawStatusCounts,
    isLoading: isStatusCountsLoading,
    isFetching: isStatusCountsFetching,
    isRefetching: isStatusCountsRefetching,
    refetch,
  } = useQuery({
    ...withPolling(
      queries.v1TaskRuns.metrics(tenantId, {
        since:
          createdAfter ||
          new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
        until: createdBefore,
        parent_task_external_id: parentTaskExternalId,
        workflow_ids: workflowIds ?? [],
        additional_metadata: additionalMetadata,
      }),
      refetchInterval,
    ),
    placeholderData: (prev) => prev,
  });

  const runStatusCounts = rawStatusCounts || [];

  const { data: queueMetricsRaw, isLoading: isQueueMetricsLoading } = useQuery({
    ...withPolling(
      queries.metrics.getStepRunQueueMetrics(tenantId),
      refetchInterval,
    ),
    enabled: showQueueMetrics,
  });

  const queueMetrics = queueMetricsRaw?.queues || {};

  return {
    runStatusCounts,
    isStatusCountsRefetching,
    isStatusCountsLoading,
    isStatusCountsFetching,
    isQueueMetricsLoading,
    refetch,
    queueMetrics,
  };
};
