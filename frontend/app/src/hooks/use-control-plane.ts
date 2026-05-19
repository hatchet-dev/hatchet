import { controlPlaneMetaQuery } from '@/lib/api/api';
import { inferControlPlaneEnabled } from '@/lib/api/control-plane-status';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';

export default function useControlPlane() {
  const result = useQuery(controlPlaneMetaQuery);

  const isControlPlaneEnabled = useMemo(
    () => inferControlPlaneEnabled(result.data?.data),
    [result.data?.data],
  );

  return {
    isControlPlaneEnabled,
    isControlPlaneLoading: result.isLoading,
    controlPlaneMeta: result.data?.data,
  };
}
