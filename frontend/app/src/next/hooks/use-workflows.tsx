import api from '@/next/lib/api';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import useTenant from './use-tenant';

interface UseWorkflowsOptions {
  refetchInterval?: number;
}

export default function useWorkflows({
  refetchInterval,
}: UseWorkflowsOptions = {}) {
  const { tenant } = useTenant();
  const queryClient = useQueryClient();

  const listWorkflowsQuery = useQuery({
    queryKey: ['workflows', 'list', tenant?.metadata.id || 'tenant', ''],
    queryFn: async () => {
      if (!tenant) {
        return { rows: [], pagination: { current_page: 0, num_pages: 0 } };
      }

      const res = await api.workflowList(tenant.metadata.id);

      return res.data;
    },
    refetchInterval,
  });

  const invalidate = async () => {
    await queryClient.invalidateQueries({
      queryKey: ['workflows', 'list', tenant?.metadata.id || 'tenant', ''],
    });
  };

  return {
    data: listWorkflowsQuery.data?.rows || [],
    pagination: listWorkflowsQuery.data?.pagination,
    isLoading: listWorkflowsQuery.isLoading,
    invalidate,
  };
}
