"use client";

import { usePathname, useSearchParams } from "next/navigation";
import posthog from "posthog-js";
import { PostHogProvider as PhProvider, usePostHog } from "posthog-js/react";
import { useEffect, useRef } from "react";
import { useConsent } from "@/context/ConsentContext";

export function PostHogProvider({ children }: { children: React.ReactNode }) {
  const { hasConsent } = useConsent();
  const initializedRef = useRef(false);

  useEffect(() => {
    const key = process.env.NEXT_PUBLIC_POSTHOG_KEY;

    if (!key)
      return console.error("PostHog key is not set in environment variables");

    if (!hasConsent) {
      posthog.stopSessionRecording();
      posthog.opt_out_capturing();
      posthog.reset();

      return;
    }

    // Initialize PostHog on first run
    if (!initializedRef.current) {
      posthog.init(key, {
        api_host:
          process.env.NEXT_PUBLIC_POSTHOG_HOST || "https://us.i.posthog.com",
        person_profiles: "identified_only",
        capture_pageleave: true,
        capture_exceptions: {
          capture_unhandled_errors: true,
          capture_unhandled_rejections: true,
          capture_console_errors: false, // handle these manually
        },
        session_recording: {
          maskAllInputs: false,
          maskInputOptions: { password: true },
        },
        persistence: "localStorage+cookie",
        before_send: (event) => {
          // You can customize exception events for better grouping
          return event;
        },
      });
      initializedRef.current = true;
    }

    posthog.opt_in_capturing();
    posthog.startSessionRecording();
  }, [hasConsent]);

  return (
    <PhProvider client={posthog}>
      <PostHogPageView />
      {children}
    </PhProvider>
  );
}

function PostHogPageView() {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const posthog = usePostHog();

  useEffect(() => {
    if (pathname && posthog) {
      let url = window.origin + pathname;
      if (searchParams.toString()) {
        url = `${url}?${searchParams.toString()}`;
      }

      posthog.capture("$pageview", { $current_url: url });
    }
  }, [pathname, searchParams, posthog]);

  return null;
}
