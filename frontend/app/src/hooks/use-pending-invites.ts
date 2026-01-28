import api from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import useCloud from '@/pages/auth/hooks/use-cloud';
import { useQuery } from '@tanstack/react-query';

export const usePendingInvites = () => {
  const { isCloudEnabled, isCloudLoading } = useCloud();

  const query = useQuery({
    queryKey: ['pending-invites', isCloudEnabled],
    queryFn: async () => {
      const [tenantInvites, orgInvites] = await Promise.allSettled([
        api.userListTenantInvites(),
        isCloudEnabled
          ? cloudApi.userListOrganizationInvites()
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
    isLoading: isCloudLoading || query.isLoading,
  };
};
