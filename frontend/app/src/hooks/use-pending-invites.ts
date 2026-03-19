import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { useTenantApi } from '@/lib/api/tenant-wrapper';
import useCloud from '@/hooks/use-cloud';
import useControlPlane from '@/hooks/use-control-plane';
import { useQuery } from '@tanstack/react-query';

export const usePendingInvites = () => {
  const { isCloudEnabled, isCloudLoading } = useCloud();
  const { isControlPlaneEnabled, isControlPlaneLoading } = useControlPlane();
  const orgApi = useOrganizationApi();
  const tenantApi = useTenantApi();
  const hasOrgInvites = isCloudEnabled || isControlPlaneEnabled;

  const query = useQuery({
    queryKey: ['pending-invites', isCloudEnabled, isControlPlaneEnabled],
    enabled: !isCloudLoading && !isControlPlaneLoading,
    queryFn: async () => {
      const [tenantInvites, orgInvites] = await Promise.allSettled([
        tenantApi.userListTenantInvites(),
        hasOrgInvites
          ? orgApi.userListOrganizationInvites()
          : Promise.resolve({ data: { rows: [] } }),
      ]);

      const tenantCount =
        tenantInvites.status === 'fulfilled'
          ? tenantInvites.value.data.rows?.length || 0
          : 0;
      const orgCount =
        orgInvites.status === 'fulfilled'
          ? orgInvites.value.data.rows?.length || 0
          : 0;

      return tenantCount + orgCount;
    },
    refetchInterval: 30000,
  });

  return {
    pendingInvitesQuery: query,
    isLoading: isCloudLoading || isControlPlaneLoading || query.isLoading,
  };
};
