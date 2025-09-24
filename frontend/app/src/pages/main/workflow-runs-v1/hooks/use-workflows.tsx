import { useCurrentTenantId } from '@/hooks/use-tenant';
import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';

export const useWorkflows = () => {
  const { tenantId } = useCurrentTenantId();

  const { data } = useQuery({
    ...queries.workflows.list(tenantId, { limit: 200 }),
  });

  return data?.rows ?? [];
};
