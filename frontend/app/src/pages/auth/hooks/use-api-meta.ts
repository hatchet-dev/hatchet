import useControlPlane from '@/hooks/use-control-plane.ts';
import api from '@/lib/api';
import { controlPlaneApi } from '@/lib/api/api.ts';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';

export default function useApiMeta() {
  const { isControlPlaneEnabled } = useControlPlane();
  const metaQuery = useQuery({
    queryKey: ['metadata:get'],
    queryFn: async () => {
      try {
        return await (isControlPlaneEnabled
          ? controlPlaneApi.metadataGet()
          : api.metadataGet());
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
