import { controlPlaneMetaQuery } from '@/lib/api/api';
import { inferControlPlaneEnabled } from '@/lib/api/control-plane-status';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';

export default function useControlPlane() {
  const result = useQuery({
    ...controlPlaneMetaQuery,
    refetchOnMount: 'always',
  });

  const isControlPlaneEnabled = useMemo(
    () => inferControlPlaneEnabled(result.data?.data),
    [result.data?.data],
  );

  const controlPlaneMeta = result.data?.data;

  return {
    isControlPlaneEnabled,
    isSelfHosted: !isControlPlaneEnabled,
    canBill: !!controlPlaneMeta?.canBill,
    isControlPlaneLoading: result.isLoading,
    controlPlaneMeta,
  };
}
