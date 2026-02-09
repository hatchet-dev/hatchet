import { useTenantDetails } from './use-tenant';
import { usePostHog } from 'posthog-js/react';
import { useCallback } from 'react';

interface AnalyticsEvent {
  [key: string]: unknown;
}

interface UseAnalyticsReturn {
  capture: (eventName: string, properties?: AnalyticsEvent) => void;
  identify: (
    userId: string,
    properties?: AnalyticsEvent,
    setOnceProperties?: AnalyticsEvent,
  ) => void;
  isAvailable: boolean;
}

export const POSTHOG_DISTINCT_ID_LOCAL_STORAGE_KEY = 'ph__distinct_id';
export const POSTHOG_SESSION_ID_LOCAL_STORAGE_KEY = 'ph__session_id';

/**
 * Hook for PostHog analytics integration
 * Provides a clean interface for tracking events and identifying users
 */
export function useAnalytics(): UseAnalyticsReturn {
  const posthog = usePostHog();
  const { tenant } = useTenantDetails();

  const isAvailable = useCallback(() => {
    return !!posthog && (!tenant || !tenant.analyticsOptOut);
  }, [posthog, tenant]);

  const capture = useCallback(
    (eventName: string, properties?: AnalyticsEvent) => {
      if (!isAvailable()) {
        return;
      }

      try {
        posthog.capture(eventName, properties);
      } catch (error) {
        console.warn('Analytics capture failed:', error);
      }
    },
    [posthog, isAvailable],
  );

  const identify = useCallback(
    (
      userId: string,
      properties?: AnalyticsEvent,
      setOnceProperties?: AnalyticsEvent,
    ) => {
      if (!isAvailable()) {
        return;
      }

      try {
        posthog.identify(userId, properties, setOnceProperties);
      } catch (error) {
        console.warn('Analytics identify failed:', error);
      }
    },
    [posthog, isAvailable],
  );

  return {
    capture,
    identify,
    isAvailable: isAvailable(),
  };
}
