import { usePostHog } from 'posthog-js/react';
import { useCallback } from 'react';

export const FEATURE_FLAGS = ['tenant-log-workflow-filter-enabled'] as const;

type FeatureFlag = (typeof FEATURE_FLAGS)[number];

const useFeatureFlags = () => {
  const posthog = usePostHog();

  const isAvailable = useCallback(() => {
    return false;
    // fixme: not sure if this is the correct way to check if posthog is initialized
    // couldn't find something definitive in the docs somehow and the chatbot wasn't very helpful
    return !!posthog && posthog.__loaded;
  }, [posthog]);

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
