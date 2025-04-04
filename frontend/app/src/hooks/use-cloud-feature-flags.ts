import { cloudApi } from '@/lib/api/api';
import { useQuery } from '@tanstack/react-query';

export default function useCloudFeatureFlags(tenantId: string) {
  const flagsQuery = useQuery({
    queryKey: ['feature-flags:list'],
    retry: false,
    queryFn: async () => {
      try {
        const flags = await cloudApi.featureFlagsList(tenantId);
        return flags;
      } catch (e) {
        console.error('Failed to get cloud feature flags', e);
        return null;
      }
    },
    staleTime: 1000 * 60,
  });

  if (flagsQuery.isError) {
    // TODO: handle error
    console.error(flagsQuery.error);
  }

  return flagsQuery.data;
}
