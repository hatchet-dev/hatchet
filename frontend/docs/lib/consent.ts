/**
 * Region-scoped consent defaults, shared by the cookie banner, PostHog and the
 * Google tag.
 *
 * Visitors in the EEA, the UK and Switzerland get no non-essential storage
 * until they say yes. Everywhere else analytics is on by default and the
 * banner offers an opt-out instead of asking permission.
 *
 * The same country list is mirrored in:
 *   - hatchet-marketing/src/constants/consent.ts  (hatchet.run)
 *   - hatchet/frontend/app/src/lib/consent.ts     (cloud.hatchet.run)
 * Keep the copies in sync — a country that drifts out of one list silently
 * changes its defaults.
 */

/** EEA-27 + the UK + Switzerland, as ISO 3166-1 alpha-2 codes. */
export const RESTRICTED_REGIONS = [
  // EU-27
  "AT", "BE", "BG", "HR", "CY", "CZ", "DK", "EE", "FI", "FR",
  "DE", "GR", "HU", "IE", "IT", "LV", "LT", "LU", "MT", "NL",
  "PL", "PT", "RO", "SK", "SI", "ES", "SE",
  // Non-EU EEA
  "IS", "LI", "NO",
  // UK + Switzerland
  "GB", "CH",
] as const;

export type ConsentRegion = "restricted" | "unrestricted";
export type ConsentStatus = "granted" | "denied";

/**
 * Unknown country resolves to "restricted". A geo lookup that fails must not
 * turn consent on for someone it can't place.
 */
export function resolveRegion(country: string | null | undefined): ConsentRegion {
  if (!country) return "restricted";
  const code = country.trim().toUpperCase();
  return (RESTRICTED_REGIONS as readonly string[]).includes(code)
    ? "restricted"
    : "unrestricted";
}

/** ISO country code for the current visitor, set by proxy.ts at the edge. */
export const REGION_COOKIE = "ht_region";
/** "granted" | "denied", written only when the visitor makes an explicit choice. */
export const CONSENT_COOKIE = "ht_consent";

/**
 * 30 days rather than something shorter, because cloud.hatchet.run cannot
 * refresh this: only hatchet.run and docs.hatchet.run sit behind an edge that
 * resolves the country, so a returning customer who goes straight to the
 * dashboard would otherwise fall back to "restricted" and be counted
 * cookielessly for no good reason.
 *
 * The cost is staleness — someone resolved here who then relocates keeps the
 * old country until they next load a marketing or docs page, which re-sets it.
 * An explicit choice in `ht_consent` always wins over the regional default, so
 * this only affects visitors who have never answered the banner.
 */
export const REGION_COOKIE_MAX_AGE = 60 * 60 * 24 * 30; // 30 days
export const CONSENT_COOKIE_MAX_AGE = 60 * 60 * 24 * 365; // 1 year

const SHARED_COOKIE_DOMAIN = "hatchet.run";

/** `.hatchet.run` on production hosts, undefined on localhost/previews. */
export function sharedCookieDomain(hostname: string): string | undefined {
  if (
    hostname === SHARED_COOKIE_DOMAIN ||
    hostname.endsWith(`.${SHARED_COOKIE_DOMAIN}`)
  ) {
    return `.${SHARED_COOKIE_DOMAIN}`;
  }
  return undefined;
}

export function readCookie(name: string): string | undefined {
  if (typeof document === "undefined") return undefined;
  const match = document.cookie.match(
    new RegExp(`(?:^|;\\s*)${name.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}=([^;]*)`),
  );
  return match ? decodeURIComponent(match[1]) : undefined;
}

export function writeCookie(
  name: string,
  value: string,
  { maxAgeSeconds }: { maxAgeSeconds: number },
) {
  if (typeof document === "undefined") return;
  const domain = sharedCookieDomain(window.location.hostname);
  const parts = [
    `${name}=${encodeURIComponent(value)}`,
    "path=/",
    `max-age=${maxAgeSeconds}`,
    "SameSite=Lax",
  ];
  if (domain) parts.push(`domain=${domain}`, "Secure");
  // eslint-disable-next-line no-restricted-syntax
  document.cookie = parts.join("; ");
}

/** The consent we act on when the visitor has not made an explicit choice. */
export function defaultStatusForRegion(region: ConsentRegion): ConsentStatus {
  return region === "restricted" ? "denied" : "granted";
}
