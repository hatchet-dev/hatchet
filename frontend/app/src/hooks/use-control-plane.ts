import { controlPlaneMetaQuery } from '@/lib/api/api';
import { inferControlPlaneEnabled } from '@/lib/api/control-plane-status';
import { useSupportSession } from '@/lib/support-session';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';

export default function useControlPlane() {
  const { isSupport } = useSupportSession();
  const result = useQuery({
    ...controlPlaneMetaQuery,
    refetchOnMount: 'always',
    enabled: !isSupport,
  });

  const isControlPlaneEnabled = useMemo(
    () => !isSupport && inferControlPlaneEnabled(result.data?.data),
    [isSupport, result.data?.data],
  );

  const controlPlaneMeta = isSupport ? undefined : result.data?.data;

  return {
    isControlPlaneEnabled,
    isSelfHosted: !isControlPlaneEnabled,
    canBill: !!controlPlaneMeta?.canBill,
    isControlPlaneLoading: !isSupport && result.isLoading,
    controlPlaneMeta,
  };
}
