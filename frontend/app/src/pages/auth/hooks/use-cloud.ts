import { cloudApi } from '@/lib/api/api';
import {
  APICloudMetadata,
  FeatureFlags,
} from '@/lib/api/generated/cloud/data-contracts';
import { useApiError } from '@/lib/hooks';
import { useQuery } from '@tanstack/react-query';
import { AxiosError } from 'axios';

const metadataIndicatesCloudEnabled = (cloudMeta: APICloudMetadata) => {
  // @ts-expect-error errors is returned when this is oss
  return !!cloudMeta && !cloudMeta?.errors;
};

export const getCloudMetadataQuery = {
  queryKey: ['cloud-metadata:get'],
  retry: false,
  queryFn: async () => {
    try {
      const { data: meta } = await cloudApi.metadataGet();
      const isCloudEnabled = metadataIndicatesCloudEnabled(meta);
      if (isCloudEnabled) {
        console.log('🪓☁️');

        return {
          ...meta,
          isCloudEnabled,
        };
      }

      console.log('\x1b[33m🪓 Thanks for self-hosting Hatchet!\x1b[0m');
      console.log('For support, please contact support@hatchet.run,');
      console.log('Join our Discord server at https://hatchet.run/discord,');
      console.log('or visit https://docs.hatchet.run/self-hosting');
    } catch (e) {
      console.error('Failed to get cloud metadata', e);
    }

    return {
      isCloudEnabled: false,
    } as const;
  },
  staleTime: 1000 * 60,
};

type UseCloudReturn =
  | {
      isCloudLoaded: false;
      isCloudEnabled: false;
      isCloudLoading: boolean;
      featureFlags: FeatureFlags | null;
      cloud: null;
    }
  | {
      isCloudLoaded: true;
      isCloudEnabled: true;
      isCloudLoading: boolean;
      featureFlags: FeatureFlags | null;
      cloud: APICloudMetadata;
    }
  | {
      isCloudLoaded: true;
      isCloudEnabled: false;
      isCloudLoading: boolean;
      featureFlags: FeatureFlags | null;
      cloud: null;
    };

export default function useCloud(tenantId?: string): UseCloudReturn {
  const { handleApiError } = useApiError();

  const cloudMetaQuery = useQuery(getCloudMetadataQuery);

  if (cloudMetaQuery.isError) {
    handleApiError(cloudMetaQuery.error as AxiosError);
  }

  const featureFlagsQuery = useQuery({
    queryKey: ['feature-flags:list', tenantId],
    retry: false,
    enabled: cloudMetaQuery.data?.isCloudEnabled && !!tenantId,
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

  if (cloudMetaQuery.data && cloudMetaQuery.data.isCloudEnabled) {
    return {
      cloud: cloudMetaQuery.data,
      isCloudEnabled: cloudMetaQuery.data.isCloudEnabled,
      isCloudLoaded: true,
      isCloudLoading: cloudMetaQuery.isLoading,
      featureFlags: featureFlagsQuery.data?.data || null,
    };
  }

  return {
    isCloudEnabled: false,
    isCloudLoaded: cloudMetaQuery.isSuccess,
    isCloudLoading: cloudMetaQuery.isLoading,
    featureFlags: featureFlagsQuery.data?.data || null,
    cloud: cloudMetaQuery?.data?.isCloudEnabled ? cloudMetaQuery.data : null,
  };
}
