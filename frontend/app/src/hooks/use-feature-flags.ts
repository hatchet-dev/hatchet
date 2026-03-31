import posthog from 'posthog-js';
import { useCallback } from 'react';

export const FEATURE_FLAGS = ['tenant-log-workflow-filter-enabled'] as const;

type FeatureFlag = (typeof FEATURE_FLAGS)[number];

const useFeatureFlags = () => {
  const isAvailable = useCallback(() => {
    return !!posthog;
  }, []);

  const isFeatureEnabled = (flagName: FeatureFlag): boolean => {
    if (!isAvailable()) {
      return false;
    }

    return posthog.isFeatureEnabled(flagName) ?? false;
  };

  return {
    isFeatureEnabled,
  };
};

export const useIsFeatureEnabled = (flagName: FeatureFlag): boolean => {
  const { isFeatureEnabled } = useFeatureFlags();
  return isFeatureEnabled(flagName);
};
