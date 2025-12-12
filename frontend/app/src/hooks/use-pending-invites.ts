import api from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import useCloudApiMeta from '@/pages/auth/hooks/use-cloud-api-meta';
import { useQuery } from '@tanstack/react-query';

export const usePendingInvites = () => {
  const { data: cloudMeta } = useCloudApiMeta();

  const query = useQuery({
    queryKey: ['pending-invites'],
    queryFn: async () => {
      const [tenantInvites, orgInvites] = await Promise.allSettled([
        api.userListTenantInvites(),
        cloudMeta?.data
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
    refetchInterval: 30000, // Refetch every 30 seconds
    enabled: true,
  });

  return {
    pendingInvitesQuery: query,
  };
};
