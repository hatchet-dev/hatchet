import { useCurrentTenantId } from '@/hooks/use-tenant';
import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';

export function useTenantEntitlements() {
  const { tenantId } = useCurrentTenantId();

  const { data, isLoading, error } = useQuery({
    ...queries.tenantEntitlements.get(tenantId),
    enabled: !!tenantId,
  });

  return { entitlements: data, isLoading, error };
}
