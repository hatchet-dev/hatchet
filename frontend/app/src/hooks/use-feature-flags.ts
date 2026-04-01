import api from '@/lib/api';
import { FeatureFlagId } from '@/lib/api/generated/data-contracts';
import { useAppContext } from '@/providers/app-context';
import { useQuery } from '@tanstack/react-query';

export { FeatureFlagId };

export const useIsFeatureEnabled = (
  flagName: FeatureFlagId,
  // controls default behavior if PostHog is not initialized. if `true`, then the feature will be enabled
  // this is useful for features that are being rolled out incrementally on Cloud, but should be enabled by default
  // on the OSS regardless of whether or not PostHog is set up or if we've removed the flag
  isEnabledIfNoPosthog: boolean,
): boolean => {
  const { tenantId } = useAppContext();

  const { data } = useQuery({
    queryKey: ['feature-flag', tenantId, flagName, isEnabledIfNoPosthog],
    queryFn: async () => {
      if (!tenantId) {
        return { isEnabled: isEnabledIfNoPosthog };
      }

      const res = await api.tenantFeatureFlagEvaluate(tenantId, {
        featureFlagId: flagName,
        isEnabledIfNoPosthog,
      });
      return res.data;
    },
    enabled: !!tenantId,
  });

  return data?.isEnabled ?? isEnabledIfNoPosthog;
};
