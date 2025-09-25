import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';

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
    isLoading,
    isFetching,
    isRefetching,
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

  const { data: queueMetricsRaw } = useQuery({
    ...queries.metrics.getStepRunQueueMetrics(tenantId),
    refetchInterval: 5000,
    enabled: showQueueMetrics,
  });

  const queueMetrics = queueMetricsRaw?.queues || {};

  return {
    runStatusCounts,
    isLoading,
    isFetching,
    isRefetching,
    refetch,
    queueMetrics,
  };
};
