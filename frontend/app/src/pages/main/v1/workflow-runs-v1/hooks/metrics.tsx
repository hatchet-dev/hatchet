import { queries } from '@/lib/api';
import { useTenant } from '@/lib/atoms';
import { useQuery } from '@tanstack/react-query';
import invariant from 'tiny-invariant';
import { useColumnFilters } from './column-filters';

export const useMetrics = ({
  workflow,
  parentTaskExternalId,
  refetchInterval,
}: {
  workflow: string | undefined;
  parentTaskExternalId: string | undefined;
  refetchInterval: number;
}) => {
  const { tenant } = useTenant();
  invariant(tenant);

  const cf = useColumnFilters();

  const metricsQuery = useQuery({
    ...queries.v1TaskRuns.metrics(tenant.metadata.id, {
      since: cf.filters.createdAfter,
      parent_task_external_id: parentTaskExternalId,
      workflow_ids: workflow ? [workflow] : [],
    }),
    placeholderData: (prev) => prev,
    refetchInterval,
  });

  const metrics = metricsQuery.data || [];

  const tenantMetricsQuery = useQuery({
    ...queries.metrics.getStepRunQueueMetrics(tenant.metadata.id),
    refetchInterval,
  });

  const tenantMetrics = tenantMetricsQuery.data?.queues || {};

  return {
    isLoading:
      metricsQuery.isLoading ||
      metricsQuery.isFetching ||
      tenantMetricsQuery.isLoading ||
      tenantMetricsQuery.isFetching,
    tenantMetrics,
    metrics,
    refetch: () => {
      tenantMetricsQuery.refetch();
      metricsQuery.refetch();
    },
  };
};
