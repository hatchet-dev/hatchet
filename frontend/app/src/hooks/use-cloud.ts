import useControlPlane from '@/hooks/use-control-plane';
import { cloudApi } from '@/lib/api/api';
import {
  APICloudMetadata,
  FeatureFlags,
} from '@/lib/api/generated/cloud/data-contracts';
import { useQuery } from '@tanstack/react-query';

type UseCloudReturn = {
  featureFlags: FeatureFlags | null;
  cloud: APICloudMetadata | null;
};

export default function useCloud(tenantId?: string): UseCloudReturn {
  const { isControlPlaneEnabled, controlPlaneMeta } = useControlPlane();

  const featureFlagsQuery = useQuery({
    queryKey: ['feature-flags:list', tenantId],
    retry: false,
    enabled: isControlPlaneEnabled && !!tenantId,
    queryFn: async () => {
      try {
        // This shouldn't be possible because of the `enabled` above, and yet, Josh found it happening at runtime
        if (tenantId === undefined) {
          return null;
        }
        // tenantId is guaranteed by `enabled`
        return await cloudApi.featureFlagsList(tenantId as string);
      } catch (e) {
        return null;
      }
    },
    staleTime: 1000 * 60,
  });

  return {
    featureFlags: featureFlagsQuery.data?.data || null,
    cloud: isControlPlaneEnabled
      ? {
          canBill: controlPlaneMeta?.canBill ?? false,
          canLinkGithub: true,
          metricsEnabled: true,
          requireBillingForManagedCompute: true,
          inactivityLogoutMs: controlPlaneMeta?.inactivityLogoutMs,
        }
      : null,
  };
}
