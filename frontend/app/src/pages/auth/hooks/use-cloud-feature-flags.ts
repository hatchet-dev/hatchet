import { cloudApi } from '@/lib/api/api';
import { useApiError } from '@/lib/hooks';
import useCloudApiMeta from '@/pages/auth/hooks/use-cloud-api-meta';
import { useQuery } from '@tanstack/react-query';
import { AxiosError } from 'axios';

export default function useCloudFeatureFlags(tenantId: string) {
  const { handleApiError } = useApiError({});
  const { isCloudEnabled } = useCloudApiMeta();

  const flagsQuery = useQuery({
    queryKey: ['feature-flags:list', tenantId],
    retry: false,
    enabled: isCloudEnabled && !!tenantId,
    queryFn: async () => {
      try {
        const flags = await cloudApi.featureFlagsList(tenantId);
        return flags;
      } catch (e) {
        return null;
      }
    },
    staleTime: 1000 * 60,
  });

  if (flagsQuery.isError) {
    handleApiError(flagsQuery.error as AxiosError);
  }

  return flagsQuery.data;
}
