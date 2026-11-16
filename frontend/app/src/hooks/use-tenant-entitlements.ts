import { queries } from '@/lib/api';
import { useAppContext } from '@/providers/app-context';
import { useQuery } from '@tanstack/react-query';

/** The active tenant's engine feature entitlements; org-level limits are in useOrganizationEntitlements. */
export function useTenantEntitlements() {
  // optional: the shell also renders on routes without a tenant, e.g. /tenants
  const { tenantId } = useAppContext();

  const { data, isLoading, error } = useQuery({
    ...queries.tenantEntitlements.get(tenantId ?? ''),
    enabled: !!tenantId,
  });

  return { entitlements: data, isLoading, error };
}
