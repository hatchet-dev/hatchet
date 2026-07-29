import useControlPlane from '@/hooks/use-control-plane';
import { cloudApi, fetchControlPlaneStatus } from '@/lib/api/api';
import {
  APICloudMetadata,
  FeatureFlags,
} from '@/lib/api/generated/cloud/data-contracts';
import { useQuery } from '@tanstack/react-query';

export const metadataIndicatesLegacyCloudEnabled = (
  cloudMeta: APICloudMetadata,
) => {
  // @ts-expect-error errors is returned when this is oss
  return !!cloudMeta && !cloudMeta?.errors;
};

// Detects the legacy (non-control-plane) hosted-cloud backend. This is a
// distinct concept from `useCloud().isCloudEnabled`, which is `true` under the
// control plane too — see the note there. Callers that mean "is the legacy
// cloud backend active?" (route loaders, cloud metadata consumers) read
// `isLegacyCloudEnabled`; callers that mean "should cloud-style features be
// available?" read `useCloud().isCloudEnabled`.
export const getCloudMetadataQuery = {
  queryKey: ['cloud-metadata:get'],
  retry: false,
  queryFn: async () => {
    // Under the control plane there is no legacy cloud backend: the control
    // plane is the source of truth, so `/api/v1/cloud/metadata` (which 403s) is
    // never hit.
    const { isControlPlaneEnabled } = await fetchControlPlaneStatus();
    if (isControlPlaneEnabled) {
      return { isLegacyCloudEnabled: false } as const;
    }

    try {
      const { data: meta } = await cloudApi.metadataGet();
      const isLegacyCloudEnabled = metadataIndicatesLegacyCloudEnabled(meta);
      if (isLegacyCloudEnabled) {
        console.log('🪓☁️');

        return {
          ...meta,
          isLegacyCloudEnabled,
        };
      }

      console.log('\x1b[33m🪓 Thanks for self-hosting Hatchet!\x1b[0m');
      console.log('For support, please contact support@hatchet.run,');
      console.log('Join our Discord server at https://hatchet.run/discord,');
      console.log('or visit https://docs.hatchet.run/self-hosting');
    } catch (e) {
      console.error('Failed to get cloud metadata', e);
    }

    return {
      isLegacyCloudEnabled: false,
    } as const;
  },
  staleTime: 1000 * 60,
};

type UseCloudReturn =
  | {
      isCloudLoaded: false;
      isCloudEnabled: false;
      isCloudLoading: boolean;
      featureFlags: FeatureFlags | null;
      cloud: null;
    }
  | {
      isCloudLoaded: true;
      isCloudEnabled: true;
      isCloudLoading: boolean;
      featureFlags: FeatureFlags | null;
      cloud: APICloudMetadata;
    }
  | {
      isCloudLoaded: true;
      isCloudEnabled: false;
      isCloudLoading: boolean;
      featureFlags: FeatureFlags | null;
      cloud: null;
    };

// `isCloudEnabled` here is the *feature* sense: "should cloud-style features
// (billing, GitHub linking, metrics) be available?". It is `true` under the
// control plane, which provides those features, even though there is no legacy
// cloud backend (`isLegacyCloudEnabled` is `false` then). Outside the control
// plane the two coincide.
export default function useCloud(tenantId?: string): UseCloudReturn {
  const cloudMetaQuery = useQuery(getCloudMetadataQuery);
  const { isControlPlaneEnabled, controlPlaneMeta } = useControlPlane();

  const featureFlagsQuery = useQuery({
    queryKey: ['feature-flags:list', tenantId],
    retry: false,
    enabled:
      (isControlPlaneEnabled || cloudMetaQuery.data?.isLegacyCloudEnabled) &&
      !!tenantId,
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

  if (isControlPlaneEnabled) {
    return {
      cloud: {
        canBill: controlPlaneMeta?.canBill ?? false,
        canLinkGithub: true,
        metricsEnabled: true,
        requireBillingForManagedCompute: true,
        inactivityLogoutMs: controlPlaneMeta?.inactivityLogoutMs,
      },
      isCloudEnabled: true,
      isCloudLoaded: true,
      isCloudLoading: false,
      featureFlags: featureFlagsQuery.data?.data || null,
    };
  }

  if (cloudMetaQuery.data && cloudMetaQuery.data.isLegacyCloudEnabled) {
    return {
      cloud: cloudMetaQuery.data,
      isCloudEnabled: cloudMetaQuery.data.isLegacyCloudEnabled,
      isCloudLoaded: true,
      isCloudLoading: cloudMetaQuery.isLoading,
      featureFlags: featureFlagsQuery.data?.data || null,
    };
  }

  return {
    isCloudEnabled: false,
    isCloudLoaded: cloudMetaQuery.isSuccess,
    isCloudLoading: cloudMetaQuery.isLoading,
    featureFlags: featureFlagsQuery.data?.data || null,
    cloud: cloudMetaQuery?.data?.isLegacyCloudEnabled
      ? cloudMetaQuery.data
      : null,
  };
}
