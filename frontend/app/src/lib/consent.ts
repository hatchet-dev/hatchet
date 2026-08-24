/**
 * Region-scoped consent defaults.
 *
 * Visitors in the EEA, the UK and Switzerland get no non-essential storage
 * until they say yes. Everywhere else analytics is on by default.
 *
 * cloud.hatchet.run has no edge geo header of its own, so the country comes
 * from the `ht_region` cookie that hatchet.run and docs.hatchet.run publish on
 * `.hatchet.run`. A visitor who lands here directly has no cookie, which
 * resolves to "restricted" — the same behavior this app had before regions
 * existed, so a miss is never a regression.
 *
 * The same country list is mirrored in:
 *   - hatchet-marketing/src/constants/consent.ts  (hatchet.run)
 *   - hatchet/frontend/docs/lib/consent.ts        (docs.hatchet.run)
 * Keep the copies in sync.
 */

/** EEA-27 + the UK + Switzerland, as ISO 3166-1 alpha-2 codes. */
export const RESTRICTED_REGIONS = [
  // EU-27
  'AT',
  'BE',
  'BG',
  'HR',
  'CY',
  'CZ',
  'DK',
  'EE',
  'FI',
  'FR',
  'DE',
  'GR',
  'HU',
  'IE',
  'IT',
  'LV',
  'LT',
  'LU',
  'MT',
  'NL',
  'PL',
  'PT',
  'RO',
  'SK',
  'SI',
  'ES',
  'SE',
  // Non-EU EEA
  'IS',
  'LI',
  'NO',
  // UK + Switzerland
  'GB',
  'CH',
] as const;

export type ConsentRegion = 'restricted' | 'unrestricted';
export type ConsentStatus = 'granted' | 'denied';

export const REGION_COOKIE = 'ht_region';
export const CONSENT_COOKIE = 'ht_consent';

export function resolveRegion(
  country: string | null | undefined,
): ConsentRegion {
  if (!country) {
    return 'restricted';
  }
  const code = country.trim().toUpperCase();
  return (RESTRICTED_REGIONS as readonly string[]).includes(code)
    ? 'restricted'
    : 'unrestricted';
}

const SHARED_COOKIE_DOMAIN = 'hatchet.run';

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

export function writeCookie(
  name: string,
  value: string,
  maxAgeSeconds: number,
) {
  if (typeof document === 'undefined') {
    return;
  }
  const domain = sharedCookieDomain(window.location.hostname);
  const parts = [
    `${name}=${encodeURIComponent(value)}`,
    'path=/',
    `max-age=${maxAgeSeconds}`,
    'SameSite=Lax',
  ];
  if (domain) {
    parts.push(`domain=${domain}`, 'Secure');
  }
  document.cookie = parts.join('; ');
}

export function readCookie(name: string): string | undefined {
  if (typeof document === 'undefined') {
    return undefined;
  }
  const match = document.cookie.match(
    new RegExp(
      `(?:^|;\\s*)${name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}=([^;]*)`,
    ),
  );
  return match ? decodeURIComponent(match[1]) : undefined;
}

export interface ConsentDecision {
  region: ConsentRegion;
  /** The visitor's explicit choice, or the default for their region. */
  status: ConsentStatus;
  hasExplicitChoice: boolean;
}

/**
 * Resolves consent from the cookies shared across hatchet.run,
 * docs.hatchet.run and cloud.hatchet.run. Tenant-level `analyticsOptOut`
 * always wins over whatever this returns.
 */
export function readConsentDecision(): ConsentDecision {
  const region = resolveRegion(readCookie(REGION_COOKIE));
  const stored = readCookie(CONSENT_COOKIE);
  const explicit =
    stored === 'granted' || stored === 'denied' ? stored : undefined;

  return {
    region,
    status: explicit ?? (region === 'restricted' ? 'denied' : 'granted'),
    hasExplicitChoice: explicit !== undefined,
  };
}

export function deleteCookie(name: string) {
  if (typeof document === 'undefined') {
    return;
  }
  const domain = sharedCookieDomain(window.location.hostname);
  const parts = [`${name}=`, 'path=/', 'max-age=0', 'SameSite=Lax'];
  if (domain) {
    parts.push(`domain=${domain}`, 'Secure');
  }
  document.cookie = parts.join('; ');
}
