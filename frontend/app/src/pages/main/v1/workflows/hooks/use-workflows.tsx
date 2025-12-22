import { nameKey } from '../components/workflow-columns';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { usePagination } from '@/hooks/use-pagination';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { useZodColumnFilters } from '@/hooks/use-zod-column-filters';
import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useDebounce } from 'use-debounce';
import { z } from 'zod';

type UseWorkflowsProps = {
  key: string;
};

const workflowQuerySchema = z
  .object({
    s: z.string().optional(), // search
  })
  .default({ s: undefined });

export const useWorkflows = ({ key }: UseWorkflowsProps) => {
  const { tenantId } = useCurrentTenantId();
  const { refetchInterval } = useRefetchInterval();
  const { pagination, setPagination, setPageSize, offset, limit } =
    usePagination({
      key,
    });

  const paramKey = `workflows-${key}`;

  const {
    state: { s: search },
    columnFilters,
    setColumnFilters,
    resetFilters,
  } = useZodColumnFilters(workflowQuerySchema, paramKey, { s: nameKey });

  const [debouncedSearch] = useDebounce(search, 300);

  const listWorkflowQuery = useQuery({
    ...queries.workflows.list(tenantId, {
      limit,
      offset,
      name: debouncedSearch,
    }),
    refetchInterval,
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
    columnFilters,
    setColumnFilters,
    resetFilters,
  };
};
