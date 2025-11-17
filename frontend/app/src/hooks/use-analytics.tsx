import { useCallback } from 'react';
import { useTenantDetails } from './use-tenant';

interface AnalyticsEvent {
  [key: string]: any;
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
  const { tenant } = useTenantDetails();

  const isAvailable = useCallback(() => {
    // Check if PostHog is loaded and user hasn't opted out
    return (
      typeof window !== 'undefined' &&
      (window as any).posthog &&
      (!tenant || !tenant.analyticsOptOut)
    );
  }, [tenant]);

  const capture = useCallback(
    (eventName: string, properties?: AnalyticsEvent) => {
      if (!isAvailable()) {
        return;
      }

      try {
        (window as any).posthog.capture(eventName, properties);
      } catch (error) {
        console.warn('Analytics capture failed:', error);
      }
    },
    [isAvailable],
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
        (window as any).posthog.identify(userId, properties, setOnceProperties);
      } catch (error) {
        console.warn('Analytics identify failed:', error);
      }
    },
    [isAvailable],
  );

  return {
    capture,
    identify,
    isAvailable: isAvailable(),
  };
}
