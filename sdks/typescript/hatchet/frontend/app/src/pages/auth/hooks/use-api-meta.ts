import api from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { useQuery } from '@tanstack/react-query';
import { AxiosError } from 'axios';
import { useMemo } from 'react';

export default function useApiMeta() {
  const { handleApiError } = useApiError({});

  const metaQuery = useQuery({
    queryKey: ['metadata:get'],
    queryFn: async () => {
      const meta = await api.metadataGet();
      return meta;
    },
    staleTime: 1000 * 60,
  });

  if (metaQuery.isError) {
    handleApiError(metaQuery.error as AxiosError);
  }

  const data = useMemo(() => {
    return metaQuery.data?.data;
  }, [metaQuery.data]);

  return {
    data: data,
    isLoading: false,
  };
}
