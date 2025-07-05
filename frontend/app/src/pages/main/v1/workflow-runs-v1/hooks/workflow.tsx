import { useCurrentTenantId } from '@/hooks/use-tenant';
import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';

export const useWorkflow = () => {
  const { tenantId } = useCurrentTenantId();

  const {
    data: workflowKeys,
    isLoading: workflowKeysIsLoading,
    error: workflowKeysError,
  } = useQuery({
    ...queries.workflows.list(tenantId, { limit: 200 }),
  });

  return {
    workflowKeys,
    workflowKeysIsLoading,
    workflowKeysError,
  };
};
