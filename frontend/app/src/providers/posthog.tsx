import { REFERRAL_CODE_KEY, sanitizeReferralCode } from '@/lib/referral';
import { clearUtmParams, readUtmParams } from '@/lib/utm';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import { useAppContext } from '@/providers/app-context';
import { useLocation } from '@tanstack/react-router';
import posthog from 'posthog-js';
import { PostHogProvider as PhProvider, usePostHog } from 'posthog-js/react';
import { useContext, useEffect, useMemo, useState, createContext } from 'react';

let tenantAnalyticsOptOut = false;

interface PostHogContextValue {
  isReady: boolean;
}

const PostHogContext = createContext<PostHogContextValue>({ isReady: false });

interface PostHogProviderProps {
  children: React.ReactNode;
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
export function PostHogProvider({ children }: PostHogProviderProps) {
  const { meta } = useApiMeta();
  const { tenant, user } = useAppContext();
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

    if (!config?.apiKey) {
      return;
    }

    console.info('Initializing Analytics, opt out in settings.');

    posthog.init(config.apiKey, {
      api_host: config.apiHost || 'https://us.i.posthog.com',
      cookieless_mode: 'on_reject',
      opt_out_capturing_by_default: true,
      person_profiles: 'identified_only',
      capture_pageview: false,
      capture_pageleave: true,
      session_recording: {
        maskAllInputs: true,
        maskTextSelector: '*',
      },
      persistence: 'localStorage+cookie',
      cross_subdomain_cookie: true,
      before_send: (event) => (tenantAnalyticsOptOut ? null : event),
    });

    if (posthog.get_explicit_consent_status() === 'pending') {
      posthog.opt_out_capturing();
    }

    const utms = readUtmParams();
    if (utms) {
      posthog.register(utms);
    }

    setInitialized(true);
  }, [config, tenant, initialized]);

  useEffect(() => {
    if (!initialized || !tenant) {
      return;
    }

    if (tenant.analyticsOptOut) {
      tenantAnalyticsOptOut = true;
      posthog.opt_out_capturing();
      posthog.stopSessionRecording?.();
      return;
    }

    tenantAnalyticsOptOut = false;
    if (posthog.get_explicit_consent_status() !== 'granted') {
      posthog.opt_in_capturing();
    }

    if (!user) {
      return;
    }

    let referralCode: string | null = null;
    try {
      referralCode = sanitizeReferralCode(
        localStorage.getItem(REFERRAL_CODE_KEY),
      );
    } catch {
      // noop
    }
    const utms = readUtmParams();
    if (utms) {
      posthog.register(utms);
    }

    posthog.identify(`$user_${user.metadata.id}`, {
      email: user.email,
      name: user.name,
      ...(referralCode && { referral_key: referralCode }),
    });

    if (referralCode) {
      localStorage.removeItem(REFERRAL_CODE_KEY);
    }
    if (utms) {
      clearUtmParams();
    }
  }, [user, tenant, initialized]);

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
  const { isReady } = useContext(PostHogContext);

  useEffect(() => {
    if (!isReady || !posthogClient) {
      return;
    }

    let url = window.origin + location.pathname;
    if (location.searchStr) {
      url = `${url}?${location.searchStr}`;
    }

    posthogClient.capture('$pageview', { $current_url: url });
    // eslint-disable-next-line react-hooks/exhaustive-deps -- intentionally exclude location.search to avoid firing pageviews on query param changes
  }, [isReady, location.pathname, posthogClient]);

  return null;
}
