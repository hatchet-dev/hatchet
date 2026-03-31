import posthog from 'posthog-js';

const useFeatureFlags = () => {
  const isFeatureEnabled = (flagName: string): boolean =>
    posthog.isFeatureEnabled(flagName) ?? false;

  return {
    isFeatureEnabled,
  };
};

export const useIsFeatureEnabled = (flagName: string): boolean => {
  const { isFeatureEnabled } = useFeatureFlags();
  return isFeatureEnabled(flagName);
};
