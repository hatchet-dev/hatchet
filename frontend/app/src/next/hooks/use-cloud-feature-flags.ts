import { cloudApi } from '@/lib/api/api';
import { useQuery } from '@tanstack/react-query';
import { useToast } from './utils/use-toast';

export default function useCloudFeatureFlags(tenantId: string) {
  const { toast } = useToast();

  const flagsQuery = useQuery({
    queryKey: ['feature-flags:list', tenantId],
    retry: false,
    queryFn: async () => {
      try {
        const flags = await cloudApi.featureFlagsList(tenantId);
        return flags;
      } catch (error) {
        toast({
          title: 'Error fetching feature flags',
          
          variant: 'destructive',
          error,
        });
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
