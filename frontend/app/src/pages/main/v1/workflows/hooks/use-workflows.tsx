import { useQuery } from '@tanstack/react-query';
import { queries } from '@/lib/api';
import { usePagination } from '@/hooks/use-pagination';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';

type UseWorkflowsProps = {
  key: string;
};

export const useWorkflows = ({ key }: UseWorkflowsProps) => {
  const { tenantId } = useCurrentTenantId();
  const { currentInterval } = useRefetchInterval();
  const { pagination, setPagination, setPageSize, offset, limit } =
    usePagination({
      key,
    });

  const listWorkflowQuery = useQuery({
    ...queries.workflows.list(tenantId, {
      limit,
      offset,
    }),
    refetchInterval: currentInterval,
    placeholderData: (data) => data,
  });

  const workflows = listWorkflowQuery.data?.rows || [];
  const numWorkflows = listWorkflowQuery.data?.pagination?.num_pages || 0;

  return {
    workflows,
    numWorkflows,
    isLoading: listWorkflowQuery.isLoading,
    isRefetching: listWorkflowQuery.isRefetching,
    error: listWorkflowQuery.error,
    refetch: listWorkflowQuery.refetch,
    pagination,
    setPagination,
    setPageSize,
  };
};
