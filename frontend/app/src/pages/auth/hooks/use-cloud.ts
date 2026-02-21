import { cloudApi } from '@/lib/api/api';
import { APICloudMetadata } from '@/lib/api/generated/cloud/data-contracts';
import { useApiError } from '@/lib/hooks';
import { useQuery } from '@tanstack/react-query';
import { AxiosError } from 'axios';
import { useMemo } from 'react';

const metadataIndicatesCloudEnabled = (cloudMeta: APICloudMetadata) => {
  // @ts-expect-error errors is returned when this is oss
  return !!cloudMeta && !cloudMeta?.errors;
};

export default function useCloud(tenantId?: string) {
  const { handleApiError } = useApiError({});

  const cloudMetaQuery = useQuery({
    queryKey: ['cloud-metadata:get'],
    retry: false,
    queryFn: async () => {
      try {
        const meta = await cloudApi.metadataGet();
        if (!metadataIndicatesCloudEnabled(meta.data)) {
          console.log('\x1b[33m🪓 Thanks for self-hosting Hatchet!\x1b[0m');
          console.log('For support, please contact support@hatchet.run,');
          console.log(
            'Join our Discord server at https://hatchet.run/discord,',
          );
          console.log('or visit https://docs.hatchet.run/self-hosting');
        } else {
          console.log('🪓☁️');
        }

        return meta;
      } catch (e) {
        console.error('Failed to get cloud metadata', e);
        return null;
      }
    },
    staleTime: 1000 * 60,
  });

  if (cloudMetaQuery.isError) {
    handleApiError(cloudMetaQuery.error as AxiosError);
  }

  const isCloudEnabled = useMemo(() => {
    return metadataIndicatesCloudEnabled(cloudMetaQuery.data?.data || {});
  }, [cloudMetaQuery.data?.data]);

  const featureFlagsQuery = useQuery({
    queryKey: ['feature-flags:list', tenantId],
    retry: false,
    enabled: isCloudEnabled && !!tenantId,
    queryFn: async () => {
      try {
        // tenantId is guaranteed by `enabled`
        return await cloudApi.featureFlagsList(tenantId as string);
      } catch (e) {
        return null;
      }
    },
    staleTime: 1000 * 60,
  });

  if (featureFlagsQuery.isError) {
    handleApiError(featureFlagsQuery.error as AxiosError);
  }

  return {
    cloud: cloudMetaQuery.data?.data,
    isCloudEnabled,
    isCloudLoading: cloudMetaQuery.isLoading,
    featureFlags: featureFlagsQuery.data?.data,
  };
}
