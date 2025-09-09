import { usePagination } from '@/hooks/use-pagination';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import api from '@/lib/api';
import { useQuery } from '@tanstack/react-query';

type UseFiltersProps = {
  key: string;
  workflowIds?: string[];
  scopes?: string[];
};

export const useFilters = ({ key, workflowIds, scopes }: UseFiltersProps) => {
  const { tenantId } = useCurrentTenantId();
  const { limit, offset, pagination, setPagination, setPageSize } =
    usePagination({
      key,
    });

  const { data, isLoading, refetch, error } = useQuery({
    queryKey: ['v1:filter:list', tenantId, key],
    queryFn: async () => {
      const response = await api.v1FilterList(tenantId, {
        offset,
        limit,
        workflowIds,
        scopes,
      });

      return response.data;
    },
    refetchInterval: 10000,
  });

  const filters = data?.rows ?? [];

  return {
    filters,
    isLoading,
    refetch,
    error,
    pagination,
    setPagination,
    setPageSize,
  };
};
