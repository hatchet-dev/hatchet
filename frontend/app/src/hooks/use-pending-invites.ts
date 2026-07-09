import useCloud from '@/hooks/use-cloud';
import useControlPlane from '@/hooks/use-control-plane';
import api, { TenantInvite } from '@/lib/api';
import { cloudApi, controlPlaneApi } from '@/lib/api/api';
import { OrganizationInvite } from '@/lib/api/generated/cloud/data-contracts';
import { OrganizationInviteTenant } from '@/lib/api/generated/control-plane/data-contracts';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback } from 'react';

// The cloud client's OrganizationInvite lacks the control-plane-only `tenants`
// field. It is absent (not `[]`) when there are no tenant grants, the invite is
// for an OWNER, or the server is older — never assume it exists.
export type PendingOrganizationInvite = OrganizationInvite & {
  tenants?: OrganizationInviteTenant[];
};

export const pendingInvitesQuery = (
  isCloudEnabled: boolean,
  isControlPlaneEnabled: boolean,
) => ({
  queryKey: ['pending-invites', isCloudEnabled, isControlPlaneEnabled],
  queryFn: async () => {
    const [tenantInvitesRes, orgInvitesRes] = await Promise.allSettled([
      isControlPlaneEnabled
        ? controlPlaneApi.userListTenantInvites()
        : api.userListTenantInvites(),
      isCloudEnabled || isControlPlaneEnabled
        ? isControlPlaneEnabled
          ? controlPlaneApi.userListOrganizationInvites()
          : cloudApi.userListOrganizationInvites()
        : Promise.resolve({ data: { rows: [] } }),
    ]);

    const tenantInvites: TenantInvite[] =
      tenantInvitesRes.status === 'fulfilled'
        ? tenantInvitesRes.value.data.rows || []
        : [];
    const organizationInvites: PendingOrganizationInvite[] =
      orgInvitesRes.status === 'fulfilled'
        ? orgInvitesRes.value.data.rows || []
        : [];

    const tenantCount = tenantInvites.length;
    const orgCount = organizationInvites.length;

    return {
      inviteCount: tenantCount + orgCount,
      tenantInvites,
      organizationInvites,
    };
  },
  refetchInterval: 30_000,
  staleTime: 30_000,
});

export const usePendingInvites = (opts?: {
  isCloudEnabled?: boolean;
  isControlPlaneEnabled?: boolean;
}) => {
  const { isCloudEnabled, isCloudLoading } = useCloud();
  const { isControlPlaneEnabled } = useControlPlane();
  const queryClient = useQueryClient();
  const resolvedIsCloudEnabled = opts?.isCloudEnabled ?? isCloudEnabled;
  const resolvedIsControlPlaneEnabled =
    opts?.isControlPlaneEnabled ?? isControlPlaneEnabled;

  const query = useQuery(
    pendingInvitesQuery(resolvedIsCloudEnabled, resolvedIsControlPlaneEnabled),
  );

  const invalidate = useCallback(() => {
    queryClient.invalidateQueries({
      queryKey: ['pending-invites'],
    });
  }, [queryClient]);

  const get = useCallback(
    () =>
      query
        .refetch({
          cancelRefetch: false,
        })
        .then((result) => {
          if (result.isSuccess) {
            return result.data;
          }

          throw result.error;
        }),
    [query],
  );

  return {
    pendingInvitesQuery: query,
    isLoading: isCloudLoading || query.isLoading,
    isLoaded: query.isSuccess,
    invalidate,
    get,
  };
};
