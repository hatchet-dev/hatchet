import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';

export const useMetrics = ({
  workflow,
  parentTaskExternalId,
  additionalMetadata,
  createdAfter,
}: {
  workflow: string | undefined;
  parentTaskExternalId: string | undefined;
  additionalMetadata?: string[] | undefined;
  createdAfter?: string;
}) => {
  const { tenantId } = useCurrentTenantId();
  const { currentInterval } = useRefetchInterval();

  const metricsQuery = useQuery({
    ...queries.v1TaskRuns.metrics(tenantId, {
      since:
        createdAfter ||
        new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
      parent_task_external_id: parentTaskExternalId,
      workflow_ids: workflow ? [workflow] : [],
      additional_metadata: additionalMetadata,
    }),
    placeholderData: (prev) => prev,
    refetchInterval: currentInterval,
  });

  const metrics = metricsQuery.data || [];

  const tenantMetricsQuery = useQuery({
    ...queries.metrics.getStepRunQueueMetrics(tenantId),
    refetchInterval: currentInterval,
  });

  const tenantMetrics = tenantMetricsQuery.data?.queues || {};

  return {
    isLoading: metricsQuery.isLoading || tenantMetricsQuery.isLoading,
    isFetching: metricsQuery.isFetching || tenantMetricsQuery.isFetching,
    isRefetching: metricsQuery.isRefetching || tenantMetricsQuery.isRefetching,
    tenantMetrics,
    metrics,
    refetch: () => {
      tenantMetricsQuery.refetch();
      metricsQuery.refetch();
    },
  };
};
