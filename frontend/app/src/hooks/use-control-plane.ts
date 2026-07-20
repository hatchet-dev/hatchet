import { cloudApi, controlPlaneMetaQuery } from '@/lib/api/api';
import { inferControlPlaneEnabled } from '@/lib/api/control-plane-status';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';

export default function useControlPlane(tenantId?: string) {
  const result = useQuery({
    ...controlPlaneMetaQuery,
    refetchOnMount: 'always',
  });

  const isControlPlaneEnabled = useMemo(
    () => inferControlPlaneEnabled(result.data?.data),
    [result.data?.data],
  );

  const featureFlagsQuery = useQuery({
    queryKey: ['feature-flags:list', tenantId],
    retry: false,
    enabled: isControlPlaneEnabled && !!tenantId,
    queryFn: async () => {
      if (!tenantId) {
        return null;
      }

      try {
        return await cloudApi.featureFlagsList(tenantId);
      } catch {
        return null;
      }
    },
    staleTime: 1000 * 60,
  });

  return {
    isControlPlaneEnabled,
    isControlPlaneLoading: result.isLoading,
    isControlPlaneLoaded: result.isSuccess,
    controlPlaneMeta: result.data?.data,
    controlPlaneCapabilities: isControlPlaneEnabled
      ? {
          canBill: result.data?.data?.canBill ?? false,
          canLinkGithub: true,
          metricsEnabled: true,
          requireBillingForManagedCompute: true,
          inactivityLogoutMs: result.data?.data?.inactivityLogoutMs,
        }
      : null,
    featureFlags: featureFlagsQuery.data?.data || null,
  };
}
