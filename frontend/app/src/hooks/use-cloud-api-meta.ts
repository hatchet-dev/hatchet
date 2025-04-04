import { cloudApi } from '@/lib/api/api';
import { useApiError } from '@/lib/hooks';
import { useQuery } from '@tanstack/react-query';
import { AxiosError } from 'axios';

export default function useCloudApiMeta() {
  const { handleApiError } = useApiError({});

  const cloudMetaQuery = useQuery({
    queryKey: ['cloud-metadata:get'],
    retry: false,
    queryFn: async () => {
      try {
        const meta = await cloudApi.metadataGet();
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

  return cloudMetaQuery.data;
}
