import useCloud from '@/hooks/use-cloud';
import useControlPlane from '@/hooks/use-control-plane';
import api from '@/lib/api';
import { cloudApi, controlPlaneApi } from '@/lib/api/api';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback } from 'react';

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
      isCloudEnabled
        ? isControlPlaneEnabled
          ? controlPlaneApi.userListOrganizationInvites()
          : cloudApi.userListOrganizationInvites()
        : Promise.resolve({ data: { rows: [] } }),
    ]);

    const tenantInvites =
      tenantInvitesRes.status === 'fulfilled'
        ? tenantInvitesRes.value.data.rows || []
        : [];
    const organizationInvites =
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
  refetchInterval: 30000,
});

export const usePendingInvites = () => {
  const { isCloudEnabled, isCloudLoading } = useCloud();
  const { isControlPlaneEnabled } = useControlPlane();
  const queryClient = useQueryClient();

  const query = useQuery(pendingInvitesQuery(isCloudEnabled, isControlPlaneEnabled));

  const invalidate = useCallback(() => {
    queryClient.resetQueries({
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
