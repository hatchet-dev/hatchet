import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useEffect, useState } from 'react';

export default function useApiMeta() {
  const [refetchInterval, setRefetchInterval] = useState<number | undefined>(
    undefined,
  );

  const metaQuery = useQuery({
    ...queries.metadata.get,
    retryDelay: 150,
    retry: 2,
    refetchInterval,
  });

  useEffect(() => {
    setRefetchInterval(metaQuery.isError ? 15000 : undefined);
  }, [metaQuery.isError]);

  return {
    data: metaQuery.data,
    isLoading: metaQuery.isLoading,
    hasFailed: metaQuery.isError && metaQuery.error,
    refetchInterval,
  };
}
