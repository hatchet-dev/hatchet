"use client";

import { usePostHog } from "posthog-js/react";
import { useEffect } from "react";

// Supports exact hostnames or wildcards like "*.onhatchet.run"
// Wildcards match both the base domain and all subdomains
const CROSS_DOMAIN_TARGETS = ["*.onhatchet.run"];

function matchesHostnamePattern(hostname: string, pattern: string): boolean {
  if (pattern.startsWith("*.")) {
    const baseDomain = pattern.slice(2);
    return hostname === baseDomain || hostname.endsWith(`.${baseDomain}`);
  }
  return hostname === pattern;
}

function shouldHandleLink(href: string): boolean {
  // Relative URLs are always same-domain, skip them
  if (!href.startsWith("http://") && !href.startsWith("https://")) {
    return false;
  }

  try {
    const url = new URL(href);
    return CROSS_DOMAIN_TARGETS.some((pattern) =>
      matchesHostnamePattern(url.hostname, pattern),
    );
  } catch (error) {
    console.error("Invalid URL in link href:", href, error);
    return false;
  }
}

function appendTrackingParams(
  href: string,
  sessionId: string,
  distinctId: string,
): string {
  const url = new URL(href);
  const trackingParams = `session_id=${sessionId}&distinct_id=${distinctId}`;

  if (url.hash && url.hash.length > 1) {
    url.hash += `&${trackingParams}`;
  } else {
    url.hash = trackingParams;
  }

  return url.toString();
}

/**
 * Global click handler that intercepts clicks on links to cross-domain targets
 * and appends PostHog session/distinct IDs for cross-domain tracking.
 *
 * This handles both Markdown-generated links and raw HTML <a> tags in MDX.
 */
export function CrossDomainLinkHandler({
  children,
}: {
  children: React.ReactNode;
}) {
  const posthog = usePostHog();

  useEffect(() => {
    function handleClick(event: MouseEvent) {
      if (!posthog) return;

      const target = event.target as HTMLElement;
      const anchor = target.closest("a");

      if (!anchor) return;

      const href = anchor.getAttribute("href");
      if (!href || !shouldHandleLink(href)) return;

      const sessionId = posthog.get_session_id();
      const distinctId = posthog.get_distinct_id();

      if (!sessionId || !distinctId) return;

      event.preventDefault();

      const newHref = appendTrackingParams(href, sessionId, distinctId);

      // Respect the original link's target attribute
      const linkTarget = anchor.getAttribute("target");

      if (linkTarget === "_blank") {
        // Preserve rel attribute if present, otherwise use safe defaults
        const rel = anchor.getAttribute("rel") || "noopener";
        window.open(newHref, "_blank", rel);
      } else {
        // Navigate in the same window
        window.location.href = newHref;
      }
    }

    document.addEventListener("click", handleClick);
    return () => document.removeEventListener("click", handleClick);
  }, [posthog]);

  return <>{children}</>;
}
