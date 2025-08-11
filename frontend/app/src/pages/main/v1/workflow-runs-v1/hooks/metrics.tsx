import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useCurrentTenantId } from '@/hooks/use-tenant';

export const useMetrics = ({
  workflow,
  parentTaskExternalId,
  createdAfter,
  refetchInterval,
  pauseRefetch = false,
}: {
  workflow: string | undefined;
  parentTaskExternalId: string | undefined;
  createdAfter?: string;
  refetchInterval: number;
  pauseRefetch?: boolean;
}) => {
  const { tenantId } = useCurrentTenantId();

  const effectiveRefetchInterval = pauseRefetch ? false : refetchInterval;

  const metricsQuery = useQuery({
    ...queries.v1TaskRuns.metrics(tenantId, {
      since: createdAfter || new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
      parent_task_external_id: parentTaskExternalId,
      workflow_ids: workflow ? [workflow] : [],
    }),
    placeholderData: (prev) => prev,
    refetchInterval: effectiveRefetchInterval,
  });

  const metrics = metricsQuery.data || [];

  const tenantMetricsQuery = useQuery({
    ...queries.metrics.getStepRunQueueMetrics(tenantId),
    refetchInterval: effectiveRefetchInterval,
  });

  const tenantMetrics = tenantMetricsQuery.data?.queues || {};

  return {
    isLoading: metricsQuery.isLoading || tenantMetricsQuery.isLoading,
    isFetching: metricsQuery.isFetching || tenantMetricsQuery.isFetching,
    tenantMetrics,
    metrics,
    refetch: () => {
      tenantMetricsQuery.refetch();
      metricsQuery.refetch();
    },
  };
};
