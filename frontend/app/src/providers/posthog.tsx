import { readConsentDecision } from '@/lib/consent';
import { REFERRAL_CODE_KEY, sanitizeReferralCode } from '@/lib/referral';
import { clearUtmParams, readUtmParams } from '@/lib/utm';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import { useAppContext } from '@/providers/app-context';
import { useLocation } from '@tanstack/react-router';
import posthog from 'posthog-js';
import { PostHogProvider as PhProvider, usePostHog } from 'posthog-js/react';
import {
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  createContext,
} from 'react';

let tenantAnalyticsOptOut = false;
let capturePaused = false;

interface PostHogContextValue {
  isReady: boolean;
  isCapturePaused: boolean;
}

const PostHogContext = createContext<PostHogContextValue>({
  isReady: false,
  isCapturePaused: false,
});

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
  const { tenant, tenantId, user, isUserLoading, isUserUniverseLoaded } =
    useAppContext();
  const [initialized, setInitialized] = useState(false);
  const [syncedTenantId, setSyncedTenantId] = useState<string>();
  const hadUserRef = useRef(false);

  tenantAnalyticsOptOut = !!tenant?.analyticsOptOut;
  const isCapturePaused =
    isUserLoading ||
    (!!user &&
      (!isUserUniverseLoaded ||
        (!!tenantId && tenant?.metadata.id !== tenantId) ||
        (!!tenant && tenant.metadata.id !== syncedTenantId)));
  capturePaused = isCapturePaused;

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

    // Consent travels with the visitor from hatchet.run / docs.hatchet.run via
    // cookies on `.hatchet.run`; a direct landing with no cookie resolves to
    // "restricted", which is how this app behaved before regions existed.
    const consent = readConsentDecision();

    posthog.init(config.apiKey, {
      api_host: config.apiHost || 'https://us.i.posthog.com',
      // Rejecting still yields aggregate, cookieless counting.
      cookieless_mode: 'on_reject',
      // EEA/UK/CH visitors are counted cookielessly until they accept;
      // everyone else is counted normally until they opt out.
      opt_out_capturing_by_default: consent.status === 'denied',
      person_profiles: 'identified_only',
      capture_pageview: false,
      capture_pageleave: true,
      session_recording: {
        maskAllInputs: true,
        maskTextSelector: '*',
      },
      persistence: 'localStorage+cookie',
      cross_subdomain_cookie: true,
      before_send: (event) =>
        tenantAnalyticsOptOut || capturePaused ? null : event,
    });

    if (
      consent.status === 'denied' &&
      posthog.get_explicit_consent_status() !== 'denied'
    ) {
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
      if (posthog.get_explicit_consent_status() !== 'denied') {
        posthog.opt_out_capturing();
      }
      posthog.stopSessionRecording?.();
      setSyncedTenantId(tenant.metadata.id);
      return;
    }

    // A decline made on any Hatchet property keeps capture off here too.
    if (readConsentDecision().status === 'denied') {
      if (posthog.get_explicit_consent_status() !== 'denied') {
        posthog.opt_out_capturing();
      }
      setSyncedTenantId(tenant.metadata.id);
      return;
    }

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
    setSyncedTenantId(tenant.metadata.id);
  }, [user, tenant, initialized]);

  useEffect(() => {
    if (user) {
      hadUserRef.current = true;
      return;
    }
    if (!initialized || isUserLoading || !hadUserRef.current) {
      return;
    }
    hadUserRef.current = false;
    setSyncedTenantId(undefined);
    if (posthog.get_explicit_consent_status() !== 'denied') {
      posthog.opt_out_capturing();
    }
  }, [user, isUserLoading, initialized]);

  const contextValue: PostHogContextValue = {
    isReady: initialized,
    isCapturePaused,
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
  const { isReady, isCapturePaused } = useContext(PostHogContext);

  useEffect(() => {
    if (!isReady || isCapturePaused || !posthogClient) {
      return;
    }

    let url = window.origin + location.pathname;
    if (location.searchStr) {
      url = `${url}?${location.searchStr}`;
    }

    posthogClient.capture('$pageview', { $current_url: url });
    // eslint-disable-next-line react-hooks/exhaustive-deps -- intentionally exclude location.search to avoid firing pageviews on query param changes
  }, [isReady, isCapturePaused, location.pathname, posthogClient]);

  return null;
}
