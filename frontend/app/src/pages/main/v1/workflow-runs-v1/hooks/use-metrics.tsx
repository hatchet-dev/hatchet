import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';

export const useMetrics = ({
  workflow,
  parentTaskExternalId,
  additionalMetadata,
  createdAfter,
  showQueueMetrics,
}: {
  workflow: string | undefined;
  parentTaskExternalId: string | undefined;
  additionalMetadata?: string[] | undefined;
  createdAfter?: string;
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
    ...queries.v1TaskRuns.metrics(tenantId, {
      since:
        createdAfter ||
        new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
      parent_task_external_id: parentTaskExternalId,
      workflow_ids: workflow ? [workflow] : [],
      additional_metadata: additionalMetadata,
    }),
    placeholderData: (prev) => prev,
    refetchInterval,
  });

  const runStatusCounts = rawStatusCounts || [];

  const { data: queueMetricsRaw, isLoading: isQueueMetricsLoading } = useQuery({
    ...queries.metrics.getStepRunQueueMetrics(tenantId),
    refetchInterval: 5000,
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
