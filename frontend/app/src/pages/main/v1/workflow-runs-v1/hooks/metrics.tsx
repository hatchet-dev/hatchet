import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useColumnFilters } from './column-filters';
import { useCurrentTenantId } from '@/hooks/use-tenant';

export const useMetrics = ({
  workflow,
  parentTaskExternalId,
  refetchInterval,
}: {
  workflow: string | undefined;
  parentTaskExternalId: string | undefined;
  refetchInterval: number;
}) => {
  const { tenantId } = useCurrentTenantId();
  const cf = useColumnFilters();

  const metricsQuery = useQuery({
    ...queries.v1TaskRuns.metrics(tenantId, {
      since: cf.filters.createdAfter,
      parent_task_external_id: parentTaskExternalId,
      workflow_ids: workflow ? [workflow] : [],
    }),
    placeholderData: (prev) => prev,
    refetchInterval,
  });

  const metrics = metricsQuery.data || [];

  const tenantMetricsQuery = useQuery({
    ...queries.metrics.getStepRunQueueMetrics(tenantId),
    refetchInterval,
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
