import posthog from 'posthog-js';
import { useCallback } from 'react';

export const FEATURE_FLAGS = ['tenant-log-workflow-filter-enabled'] as const;

type FeatureFlag = (typeof FEATURE_FLAGS)[number];

const useFeatureFlags = () => {
  const isAvailable = useCallback(() => {
    return !!posthog;
  }, []);

  const isFeatureEnabled = (
    flagName: FeatureFlag,
    isEnabledIfNoPosthog: boolean,
  ): boolean => {
    if (!isAvailable()) {
      return isEnabledIfNoPosthog;
    }

    return posthog.isFeatureEnabled(flagName) ?? false;
  };

  return {
    isFeatureEnabled,
  };
};

export const useIsFeatureEnabled = (
  flagName: FeatureFlag,
  // controls default behavior if PostHog is not initialized. if `true`, then the feature will be enabled
  isEnabledIfNoPosthog: boolean,
): boolean => {
  const { isFeatureEnabled } = useFeatureFlags();
  return isFeatureEnabled(flagName, isEnabledIfNoPosthog);
};
