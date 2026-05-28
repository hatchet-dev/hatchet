import type { User } from '@/lib/api';
import { REFERRAL_CODE_KEY, sanitizeReferralCode } from '@/lib/referral';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import { useAppContext } from '@/providers/app-context';
import { useLocation } from '@tanstack/react-router';
import posthog from 'posthog-js';
import { PostHogProvider as PhProvider, usePostHog } from 'posthog-js/react';
import { useEffect, useMemo, useState, createContext } from 'react';

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
 * - Session recording with input masking
 */
export function PostHogProvider({ children, user }: PostHogProviderProps) {
  const { meta } = useApiMeta();
  const { tenant } = useAppContext();
  const [initialized, setInitialized] = useState(false);

  const config = useMemo(() => {
    if (import.meta.env.DEV) {
      return {
        apiKey: import.meta.env.VITE_PUBLIC_POSTHOG_KEY,
        apiHost: import.meta.env.VITE_PUBLIC_POSTHOG_HOST,
      };
    }
    return meta?.posthog;
  }, [meta?.posthog]);

  useEffect(() => {
    if (initialized) {
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
      cross_subdomain_cookie: true,
    });

    setInitialized(true);
  }, [config, tenant, initialized]);

  // Handle user identification
  useEffect(() => {
    if (!initialized || !user) {
      return;
    }

    const referralCode = sanitizeReferralCode(
      localStorage.getItem(REFERRAL_CODE_KEY),
    );

    posthog.identify(`$user_${user.metadata.id}`, {
      email: user.email,
      name: user.name,
      ...(referralCode && { referral_key: referralCode }),
    });

    if (referralCode) {
      localStorage.removeItem(REFERRAL_CODE_KEY);
    }
  }, [user, initialized]);

  // Handle opt-out changes
  useEffect(() => {
    if (!initialized) {
      return;
    }

    if (tenant?.analyticsOptOut) {
      posthog.opt_out_capturing();
      posthog.stopSessionRecording?.();
    } else {
      posthog.opt_in_capturing();
    }
  }, [tenant?.analyticsOptOut, initialized]);

  const contextValue: PostHogContextValue = {
    isReady: initialized,
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
