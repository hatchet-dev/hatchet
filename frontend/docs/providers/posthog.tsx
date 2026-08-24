"use client";

import { usePathname, useSearchParams } from "next/navigation";
import posthog from "posthog-js";
import { PostHogProvider as PhProvider, usePostHog } from "posthog-js/react";
import {
  createContext,
  Suspense,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import { useConsent } from "@/context/ConsentContext";
import { REGION_COOKIE, readCookie, resolveRegion } from "@/lib/consent";

const PostHogReadyContext = createContext(false);

export function PostHogProvider({ children }: { children: React.ReactNode }) {
  const { consentStatus, hasExplicitChoice, isHydrated } = useConsent();
  const initializedRef = useRef(false);
  const [isReady, setIsReady] = useState(false);

  useEffect(() => {
    const key = process.env.NEXT_PUBLIC_POSTHOG_KEY;

    if (!key)
      return console.error("PostHog key is not set in environment variables");

    if (initializedRef.current) return;
    // Wait for the consent layer, so a stored opt-out is applied before init
    // rather than a tick after it.
    if (!isHydrated) return;

    // Read the region cookie directly rather than through useConsent: child
    // effects run before the provider's, so the context value is still on its
    // "restricted" placeholder at this point.
    const region = resolveRegion(readCookie(REGION_COOKIE));

    posthog.init(key, {
      api_host:
        process.env.NEXT_PUBLIC_POSTHOG_HOST || "https://us.i.posthog.com",
      // Visitors who reject the cookie banner are counted without device
      // storage: PostHog's servers hash ip + user agent + a daily salt into an
      // anonymous distinct_id. Requires "Cookieless server hash mode" in the
      // project settings, otherwise cookieless events are dropped at ingestion.
      cookieless_mode: "on_reject",
      // Region-scoped default: EEA/UK/CH visitors are counted cookielessly
      // until they accept, everyone else is counted normally until they opt
      // out. In on_reject mode, pending consent only captures (cookieless)
      // when the default is opt-out; without this, pre-banner events in
      // restricted regions would be dropped entirely.
      opt_out_capturing_by_default: region === "restricted",
      person_profiles: "identified_only",
      // Pageviews are captured manually by PostHogPageView.
      capture_pageview: false,
      capture_pageleave: true,
      capture_exceptions: {
        capture_unhandled_errors: true,
        capture_unhandled_rejections: true,
        capture_console_errors: false, // handle these manually
      },
      disable_session_recording: true,
      persistence: "localStorage+cookie",
      cross_subdomain_cookie: true,
      before_send: (event) => {
        return event;
      },
      loaded: () => {
        setIsReady(true);
      },
    });
    initializedRef.current = true;
  }, [isHydrated]);

  useEffect(() => {
    // Only an explicit choice moves PostHog off the region default set at
    // init; without this, an unrestricted visitor who never touched the banner
    // would be recorded as having granted explicit consent.
    if (!initializedRef.current || !hasExplicitChoice) return;

    const explicitConsent = posthog.get_explicit_consent_status();
    if (consentStatus === "granted" && explicitConsent !== "granted") {
      posthog.opt_in_capturing();
    } else if (consentStatus === "denied" && explicitConsent !== "denied") {
      posthog.opt_out_capturing();
    }
  }, [consentStatus, hasExplicitChoice]);

  return (
    <PostHogReadyContext.Provider value={isReady}>
      <PhProvider client={posthog}>
        <Suspense fallback={null}>
          <PostHogPageView />
        </Suspense>
        {children}
      </PhProvider>
    </PostHogReadyContext.Provider>
  );
}

function PostHogPageView() {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const posthog = usePostHog();
  // Wait for init: this effect runs before the parent provider's init effect
  // on first mount, and pre-init captures are dropped.
  const isReady = useContext(PostHogReadyContext);

  useEffect(() => {
    if (!isReady) return;

    if (pathname && posthog) {
      let url = window.origin + pathname;
      if (searchParams.toString()) {
        url = `${url}?${searchParams.toString()}`;
      }

      posthog.capture("$pageview", { $current_url: url });
    }
  }, [isReady, pathname, searchParams, posthog]);

  return null;
}
