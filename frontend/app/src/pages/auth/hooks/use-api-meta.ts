import api from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { useQuery } from '@tanstack/react-query';
import { AxiosError } from 'axios';

export default function useApiMeta() {
  const { handleApiError } = useApiError({});

  const metaQuery = useQuery({
    queryKey: ['metadata:get'],
    queryFn: async () => {
      const meta = await api.metadataGet();
      return meta;
    },
  });

  if (metaQuery.isError) {
    handleApiError(metaQuery.error as AxiosError);
  }

  return metaQuery;
}
