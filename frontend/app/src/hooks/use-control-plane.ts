import { controlPlaneApi } from '@/lib/api/api';
import {
  inferControlPlaneEnabled,
  writeStoredControlPlaneEnabled,
} from '@/lib/api/control-plane-status';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';

export default function useControlPlane() {
  const controlPlaneMetaQuery = useQuery({
    queryKey: ['control-plane-metadata:get'],
    retry: false,
    queryFn: async () => {
      try {
        const meta = await controlPlaneApi.metadataGet();
        const enabled = inferControlPlaneEnabled(meta.data);
        writeStoredControlPlaneEnabled(enabled);
        if (enabled) {
          console.log('🪓 Control plane active');
        }
        return meta;
      } catch (e) {
        console.error('Failed to get control plane metadata', e);
        return null;
      }
    },
    staleTime: 1000 * 60,
  });

  const isControlPlaneEnabled = useMemo(() => {
    return inferControlPlaneEnabled(controlPlaneMetaQuery.data?.data);
  }, [controlPlaneMetaQuery.data?.data]);

  return {
    isControlPlaneEnabled,
    isControlPlaneLoading: controlPlaneMetaQuery.isLoading,
    controlPlaneMeta: controlPlaneMetaQuery.data?.data,
  };
}
