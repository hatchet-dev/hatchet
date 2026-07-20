import api, { TenantInvite } from '@/lib/api';
import { controlPlaneApi, fetchControlPlaneStatus } from '@/lib/api/api';
import { OrganizationInvite } from '@/lib/api/generated/control-plane/data-contracts';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback } from 'react';

export type PendingOrganizationInvite = OrganizationInvite;

export const pendingInvitesQuery = () => ({
  queryKey: ['pending-invites'],
  queryFn: async () => {
    const { isControlPlaneEnabled } = await fetchControlPlaneStatus();
    const [tenantInvitesRes, orgInvitesRes] = await Promise.allSettled([
      isControlPlaneEnabled
        ? controlPlaneApi.userListTenantInvites()
        : api.userListTenantInvites(),
      isControlPlaneEnabled
        ? controlPlaneApi.userListOrganizationInvites()
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

export const usePendingInvites = () => {
  const queryClient = useQueryClient();

  const query = useQuery(pendingInvitesQuery());

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
    isLoading: query.isLoading,
    isLoaded: query.isSuccess,
    invalidate,
    get,
  };
};
