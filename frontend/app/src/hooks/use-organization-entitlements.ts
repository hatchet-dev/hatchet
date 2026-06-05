import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { useUserUniverse } from '@/providers/user-universe';
import { useQuery } from '@tanstack/react-query';

/**
 * Loads an organization's entitlement limit state (seats and tenants) and
 * exposes whether another member can be invited or another tenant created.
 *
 * Gating is UX-only; the control-plane API remains authoritative and will
 * reject over-limit invites/tenant creation with a 403.
 */
export function useOrganizationEntitlements(organizationId?: string | null) {
  const { isCloudEnabled } = useUserUniverse();
  const orgApi = useOrganizationApi();

  const query = useQuery({
    ...orgApi.organizationEntitlementsGetQuery(organizationId ?? ''),
    enabled: isCloudEnabled && !!organizationId,
  });

  const entitlements = query.data;

  // Default to allowing the action while data is unknown (no org, not cloud, or
  // still loading) so non-cloud and in-flight states never block the form. The
  // backend is the source of truth and will reject if the limit is exceeded.
  const canInviteUser = entitlements ? entitlements.users.canCreate : true;
  const canCreateTenant = entitlements ? entitlements.tenants.canCreate : true;

  return {
    entitlements,
    canInviteUser,
    canCreateTenant,
    isLoading: query.isLoading,
    isFetched: query.isFetched,
  };
}
