import { queries } from '@/lib/api';
import { roleAllows } from '@/lib/role-permissions';
import { useAppContext } from '@/providers/app-context';
import { useQuery } from '@tanstack/react-query';

// Roles only change when the server is redeployed, so they are fetched once per tenant. The tenant is
// optional because role pickers also render outside tenant routes.
export function useTenantRoles() {
  const { tenantId, membership } = useAppContext();

  const query = useQuery({
    ...queries.roles.list(tenantId ?? ''),
    enabled: !!tenantId,
    staleTime: Infinity,
  });

  const roles = query.data?.rows ?? [];

  return {
    roles,
    isLoading: query.isLoading,
    // The server enforces every operation, so this only hides controls. A shard older than the roles
    // endpoint answers 404, and failing closed there would lock everyone out of these screens.
    can: (operation: string) =>
      query.isError || roleAllows(roles, membership, operation),
  };
}
