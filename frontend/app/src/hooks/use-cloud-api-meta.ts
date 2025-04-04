import { cloudApi } from '@/lib/api/api';
import { useQuery } from '@tanstack/react-query';

export default function useCloudApiMeta() {
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
    // TODO: handle error
    console.error(cloudMetaQuery.error);
  }

  return cloudMetaQuery.data;
}
