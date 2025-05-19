import { cloudApi } from '@/lib/api/api';
import { useApiError } from '@/lib/hooks';
import { useQuery } from '@tanstack/react-query';
import { AxiosError } from 'axios';

export default function useCloudFeatureFlags(tenantId: string) {
  const { handleApiError } = useApiError({});

  const flagsQuery = useQuery({
    queryKey: ['feature-flags:list'],
    retry: false,
    queryFn: async () => {
      try {
        return await cloudApi.featureFlagsList(tenantId);
      } catch (e) {
        console.error('Failed to get cloud feature flags', e);
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
