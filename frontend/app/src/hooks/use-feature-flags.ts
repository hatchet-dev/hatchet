import posthog from 'posthog-js';

export const FEATURE_FLAGS = ['tenant-log-workflow-filter-enabled'] as const;

const useFeatureFlags = () => {
  const isFeatureEnabled = (
    flagName: (typeof FEATURE_FLAGS)[number],
  ): boolean => posthog.isFeatureEnabled(flagName) ?? false;

  return {
    isFeatureEnabled,
  };
};

export const useIsFeatureEnabled = (
  flagName: (typeof FEATURE_FLAGS)[number],
): boolean => {
  const { isFeatureEnabled } = useFeatureFlags();
  return isFeatureEnabled(flagName);
};
