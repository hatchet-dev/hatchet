import useCloud from '@/hooks/use-cloud';
import useControlPlane from '@/hooks/use-control-plane';
import api, { TenantInvite } from '@/lib/api';
import { cloudApi, controlPlaneApi } from '@/lib/api/api';
import { OrganizationInvite } from '@/lib/api/generated/cloud/data-contracts';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback } from 'react';

type PendingInvitesData = {
  inviteCount: number;
  tenantInvites: TenantInvite[];
  organizationInvites: OrganizationInvite[];
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
    const organizationInvites: OrganizationInvite[] =
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
  refetchInterval: 60_000,
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

  // Drop a processed invite from the cache synchronously so consumers of
  // inviteCount (auto-open effect, notification badge) don't act on a stale
  // count while the post-accept refetch is still in flight.
  const removeInviteFromCache = useCallback(
    (inviteId: string) => {
      queryClient.setQueriesData<PendingInvitesData>(
        { queryKey: ['pending-invites'] },
        (data) => {
          if (!data) {
            return data;
          }
          const tenantInvites = data.tenantInvites.filter(
            (inv) => inv.metadata.id !== inviteId,
          );
          const organizationInvites = data.organizationInvites.filter(
            (inv) => inv.metadata.id !== inviteId,
          );
          return {
            inviteCount: tenantInvites.length + organizationInvites.length,
            tenantInvites,
            organizationInvites,
          };
        },
      );
    },
    [queryClient],
  );

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
    removeInviteFromCache,
    get,
  };
};
