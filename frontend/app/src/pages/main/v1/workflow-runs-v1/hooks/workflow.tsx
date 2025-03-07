import { queries } from '@/lib/api';
import { TenantContextType } from '@/lib/outlet';
import { useQuery } from '@tanstack/react-query';
import { useOutletContext } from 'react-router-dom';
import invariant from 'tiny-invariant';

export const useWorkflow = () => {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const {
    data: workflowKeys,
    isLoading: workflowKeysIsLoading,
    error: workflowKeysError,
  } = useQuery({
    ...queries.workflows.list(tenant.metadata.id, { limit: 200 }),
  });

  return {
    workflowKeys,
    workflowKeysIsLoading,
    workflowKeysError,
  };
};
