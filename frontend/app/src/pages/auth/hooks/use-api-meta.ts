import api from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';

export default function useApiMeta() {
  const metaQuery = useQuery({
    queryKey: ['metadata:get'],
    queryFn: async () => {
      try {
        return await api.metadataGet();
      } catch (e) {
        console.error('Failed to get API metadata', e);
        return null;
      }
    },
    retry: false,
    staleTime: 1000 * 60,
  });

  const data = useMemo(() => {
    return metaQuery.data?.data;
  }, [metaQuery.data]);

  return {
    meta: data,
    isLoading: false,
  };
}
