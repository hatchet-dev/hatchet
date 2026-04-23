import { useCurrentTenantId } from '@/hooks/use-tenant';
import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useDebounce } from 'use-debounce';

export const useWorkflows = (name?: string) => {
  const { tenantId } = useCurrentTenantId();
  const [debouncedName] = useDebounce(name, 300);

  const { data } = useQuery({
    ...queries.workflows.list(tenantId, {
      limit: 200,
      name: debouncedName || undefined,
    }),
    staleTime: 30_000,
  });

  return data?.rows ?? [];
};
