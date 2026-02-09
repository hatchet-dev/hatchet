import type { User } from '@/lib/api';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import { useAppContext } from '@/providers/app-context';
import { useLocation } from '@tanstack/react-router';
import posthog from 'posthog-js';
import { PostHogProvider as PhProvider, usePostHog } from 'posthog-js/react';
import { useEffect, useRef, useMemo, createContext } from 'react';

const CROSS_DOMAIN_SESSION_ID_KEY = 'session_id';
const CROSS_DOMAIN_DISTINCT_ID_KEY = 'distinct_id';

interface PostHogContextValue {
  isReady: boolean;
}

const PostHogContext = createContext<PostHogContextValue>({ isReady: false });

interface PostHogProviderProps {
  children: React.ReactNode;
  user?: User;
}

/**
 * PostHog Analytics Provider for the Hatchet App
 *
 * Features:
 * - Config from API meta endpoint (or env vars in dev)
 * - User identification with email/name
 * - Tenant-level analytics opt-out
 * - Cross-domain tracking via URL hash bootstrap
 * - Session recording with input masking
 */
export function PostHogProvider({ children, user }: PostHogProviderProps) {
  const { meta } = useApiMeta();
  const { tenant } = useAppContext();
  const initializedRef = useRef(false);

  const config = useMemo(() => {
    if (import.meta.env.DEV) {
      return {
        apiKey: import.meta.env.VITE_PUBLIC_POSTHOG_KEY,
        apiHost: import.meta.env.VITE_PUBLIC_POSTHOG_HOST,
      };
    }
    return meta?.posthog;
  }, [meta?.posthog]);

  // Check for cross-domain tracking params in URL hash
  const bootstrapIds = useMemo(() => {
    if (typeof window === 'undefined') {
      return null;
    }

    const hashParams = new URLSearchParams(window.location.hash.substring(1));
    const sessionId = hashParams.get(CROSS_DOMAIN_SESSION_ID_KEY);
    const distinctId = hashParams.get(CROSS_DOMAIN_DISTINCT_ID_KEY);

    if (sessionId && distinctId) {
      return { sessionID: sessionId, distinctID: distinctId };
    }
    return null;
  }, []);

  useEffect(() => {
    if (initializedRef.current) {
      return;
    }

    if (tenant?.analyticsOptOut) {
      console.info(
        'Skipping Analytics initialization due to opt-out, we respect user privacy.',
      );
      return;
    }

    // Need config and tenant to initialize
    if (!config?.apiKey || !tenant) {
      return;
    }

    console.info('Initializing Analytics, opt out in settings.');

    posthog.init(config.apiKey, {
      api_host: config.apiHost || 'https://us.i.posthog.com',
      person_profiles: 'identified_only',
      capture_pageleave: true,
      session_recording: {
        maskAllInputs: true,
        maskTextSelector: '*',
      },
      persistence: 'localStorage+cookie',
      bootstrap: bootstrapIds || undefined,
    });

    initializedRef.current = true;
  }, [config, tenant, bootstrapIds]);

  // Handle user identification
  useEffect(() => {
    if (!initializedRef.current || !user) {
      return;
    }

    const ref = localStorage.getItem('ref');
    if (ref) {
      posthog.identify(ref);
    }

    posthog.identify(user.metadata.id, {
      email: user.email,
      name: user.name,
    });
  }, [user]);

  // Handle opt-out changes
  useEffect(() => {
    if (!initializedRef.current) {
      return;
    }

    if (tenant?.analyticsOptOut) {
      posthog.opt_out_capturing();
      posthog.stopSessionRecording?.();
    } else {
      posthog.opt_in_capturing();
    }
  }, [tenant?.analyticsOptOut]);

  const contextValue: PostHogContextValue = {
    isReady: initializedRef.current,
  };

  return (
    <PostHogContext.Provider value={contextValue}>
      <PhProvider client={posthog}>
        <PostHogPageView />
        {children}
      </PhProvider>
    </PostHogContext.Provider>
  );
}

function PostHogPageView() {
  const location = useLocation();
  const posthogClient = usePostHog();

  useEffect(() => {
    if (!posthogClient) {
      return;
    }

    let url = window.origin + location.pathname;
    if (location.searchStr) {
      url = `${url}?${location.searchStr}`;
    }

    posthogClient.capture('$pageview', { $current_url: url });
    // eslint-disable-next-line react-hooks/exhaustive-deps -- intentionally exclude location.search to avoid firing pageviews on query param changes
  }, [location.pathname, posthogClient]);

  return null;
}
