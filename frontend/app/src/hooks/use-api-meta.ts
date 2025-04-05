import api from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import { useQuery } from '@tanstack/react-query';
import { useEffect, useState } from 'react';

export default function useApiMeta() {
  const [refetchInterval, setRefetchInterval] = useState<number | undefined>(
    undefined,
  );

  const metaQuery = useQuery({
    queryKey: ['metadata:get'],
    queryFn: async () => (await api.metadataGet()).data,
    staleTime: 1000 * 60 * 60, // 1 hour
    retryDelay: 150,
    retry: 2,
    refetchInterval,
  });

  const { data: version } = useQuery({
    queryKey: ['info:version'],
    queryFn: async () => (await api.infoGetVersion()).data,
    retryDelay: 150,
    retry: 2,
    refetchInterval,
  });

  const { data: cloudMeta } = useQuery({
    queryKey: ['cloud-metadata:get'],
    queryFn: async () => {
      try {
        const meta = (await cloudApi.metadataGet()).data;
        return meta;
      } catch (e) {
        console.error('Failed to get cloud metadata', e);
        return;
      }
    },
    staleTime: 1000 * 60,
    retryDelay: 150,
    retry: 2,
    refetchInterval,
  });

  useEffect(() => {
    setRefetchInterval(metaQuery.isError ? 15000 : undefined);
  }, [metaQuery.isError]);

  return {
    oss: metaQuery.data,
    cloud: cloudMeta,
    isLoading: metaQuery.isLoading,
    hasFailed: metaQuery.isError && metaQuery.error,
    refetchInterval,
    version: version?.version,
  };
}
