import api, { V1TaskSummary, V1TaskSummaryList } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import useTenant from './use-tenant';

interface RunsState {
  data?: V1TaskSummary[];
  pagination?: V1TaskSummaryList['pagination'];
  isLoading: boolean;
}

interface UseRunsOptions {
  refetchInterval?: number;
}

type RunQuery = Parameters<typeof api.v1WorkflowRunList>[1];

export default function useRuns({
  refetchInterval,
}: UseRunsOptions = {}): RunsState {
  const { tenant } = useTenant();

  const query: RunQuery = {
    since: new Date(Date.now() - 1000 * 60 * 60 * 24).toISOString(),
    only_tasks: false,
  };

  const listRunsQuery = useQuery({
    queryKey: ['v1:workflow-run:list', tenant, query],
    queryFn: async () =>
      (await api.v1WorkflowRunList(tenant?.metadata.id || '', query)).data,
    refetchInterval,
  });

  // ...queries.v1WorkflowRuns.list(tenant.metadata.id, {
  //   offset: disablePagination ? 0 : offset,
  //   limit: disablePagination ? 500 : pagination.pageSize,
  //   statuses: cf.filters.status ? [cf.filters.status] : undefined,
  //   workflow_ids: workflow ? [workflow] : [],
  //   parent_task_external_id: parentTaskExternalId,
  //       since: cf.filters.createdAfter || initialRenderTime,
  //       until: cf.filters.finishedBefore,
  //       additional_metadata: cf.filters.additionalMetadata,
  //       worker_id: workerId,
  //       only_tasks: !!workerId,
  //     }),
  //     placeholderData: (prev) => prev,
  //     refetchInterval: () => {
  //       if (Object.keys(rowSelection).length > 0) {
  //         return false;
  //       }

  //       return 5000;
  //     },
  //   });

  return {
    data: listRunsQuery.data?.rows || [],
    pagination: listRunsQuery.data?.pagination,
    isLoading: listRunsQuery.isLoading,
  };
}
