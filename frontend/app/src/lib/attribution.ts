/**
 * First-party marketing attribution.
 *
 * The campaign that brought someone to Hatchet is almost never the page they
 * sign up on: they land on hatchet.run or docs.hatchet.run from an ad, read
 * for a while, then create an account on cloud.hatchet.run. So the ad click id
 * and campaign parameters are stashed in a cookie on `.hatchet.run` at the
 * landing, and the control plane reads it back at signup
 * (internal/attribution in hatchet-control-plane).
 *
 * Mirrored in:
 *   - hatchet-marketing/src/utils/attribution.ts  (hatchet.run)
 *   - hatchet/frontend/docs/proxy.ts              (docs.hatchet.run, edge-side)
 * Keep the payload keys and validation rules in sync — the server re-validates
 * everything, so a field that drifts here is silently dropped there.
 */
import { deleteCookie, readCookie, writeCookie } from './consent';

export const ATTRIBUTION_COOKIE = 'ht_attr';
export const ATTRIBUTION_COOKIE_MAX_AGE = 60 * 60 * 24 * 90; // 90 days

/** Ad click identifiers, in the order the server prefers them. */
const CLICK_ID_KEYS = ['gclid', 'gbraid', 'wbraid', 'msclkid'] as const;
const UTM_KEYS = [
  'utm_source',
  'utm_medium',
  'utm_campaign',
  'utm_term',
  'utm_content',
] as const;

const MAX_CLICK_ID_LENGTH = 256;
const MAX_UTM_LENGTH = 128;
const MAX_URL_LENGTH = 512;
const MAX_COOKIE_VALUE_LENGTH = 1024;

const CLICK_ID_PATTERN = /^[A-Za-z0-9_-]+$/;
const UTM_PATTERN = /^[a-zA-Z0-9 ._-]+$/;

export interface AttributionPayload {
  gclid?: string;
  gbraid?: string;
  wbraid?: string;
  msclkid?: string;
  utm_source?: string;
  utm_medium?: string;
  utm_campaign?: string;
  utm_term?: string;
  utm_content?: string;
  /** Same-site path only; the host is implied. */
  lp?: string;
  ref?: string;
}

function sanitize(
  value: string | null,
  pattern: RegExp,
  maxLength: number,
): string | undefined {
  if (!value) {
    return undefined;
  }
  const trimmed = value.trim();
  if (
    trimmed.length === 0 ||
    trimmed.length > maxLength ||
    !pattern.test(trimmed)
  ) {
    return undefined;
  }
  return trimmed;
}

/** base64url without padding, UTF-8 safe (btoa alone throws above U+00FF). */
function encodePayload(payload: AttributionPayload): string {
  const bytes = new TextEncoder().encode(JSON.stringify(payload));
  let binary = '';
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte);
  });
  return btoa(binary)
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '');
}

function decodePayload(raw: string): AttributionPayload | null {
  try {
    const padded = raw.replace(/-/g, '+').replace(/_/g, '/');
    const binary = atob(padded);
    const bytes = Uint8Array.from(binary, (c) => c.charCodeAt(0));
    const parsed: unknown = JSON.parse(new TextDecoder().decode(bytes));
    return typeof parsed === 'object' && parsed !== null
      ? (parsed as AttributionPayload)
      : null;
  } catch {
    return null;
  }
}

/** True when the payload names a paid click or a tagged campaign. */
export function hasCampaign(payload: AttributionPayload | null): boolean {
  if (!payload) {
    return false;
  }
  return [...CLICK_ID_KEYS, ...UTM_KEYS].some((key) => !!payload[key]);
}

export function buildAttributionPayload(
  search: string,
  pathname: string,
  referrer: string,
): AttributionPayload {
  const params = new URLSearchParams(search);
  const payload: AttributionPayload = {};

  for (const key of CLICK_ID_KEYS) {
    const value = sanitize(
      params.get(key),
      CLICK_ID_PATTERN,
      MAX_CLICK_ID_LENGTH,
    );
    if (value) {
      payload[key] = value;
    }
  }

  for (const key of UTM_KEYS) {
    const value = sanitize(params.get(key), UTM_PATTERN, MAX_UTM_LENGTH);
    if (value) {
      payload[key] = value;
    }
  }

  if (pathname.startsWith('/') && !pathname.startsWith('//')) {
    payload.lp = pathname.slice(0, MAX_URL_LENGTH);
  }

  // Only off-site referrers say anything; an internal one is noise. Origin
  // only: a full referrer URL can carry a path or query that belongs to the
  // referring site's user — a search term, a token, an email address — and
  // none of that is needed to know where someone came from.
  if (/^https?:\/\//.test(referrer) && referrer.length <= MAX_URL_LENGTH) {
    try {
      const url = new URL(referrer);
      if (url.hostname !== window.location.hostname) {
        payload.ref = url.origin;
      }
    } catch {
      // Unparseable referrer; drop it.
    }
  }

  return payload;
}

/** Removes the attribution cookie. Called when a visitor declines. */
export function clearAttribution() {
  deleteCookie(ATTRIBUTION_COOKIE);
}

/**
 * Records the current landing as the attribution for a later signup.
 *
 * Only writes when the URL actually carries a campaign — an ad click id or a
 * utm_* parameter. Writing for a bare path would set a 90-day marketing
 * cookie on every visitor to every page.
 *
 * First touch wins. Callers are responsible for consent; this is a marketing
 * cookie, not a strictly necessary one.
 */
export function captureAttribution({
  search = window.location.search,
  pathname = window.location.pathname,
  referrer = document.referrer,
}: {
  search?: string;
  pathname?: string;
  referrer?: string;
} = {}) {
  const candidate = buildAttributionPayload(search, pathname, referrer);
  if (!hasCampaign(candidate)) {
    return;
  }

  // First touch wins: an existing campaign keeps the credit.
  const existingRaw = readCookie(ATTRIBUTION_COOKIE);
  if (existingRaw && hasCampaign(decodePayload(existingRaw))) {
    return;
  }

  const encoded = encodePayload(candidate);
  if (encoded.length > MAX_COOKIE_VALUE_LENGTH) {
    return;
  }

  writeCookie(ATTRIBUTION_COOKIE, encoded, ATTRIBUTION_COOKIE_MAX_AGE);
}
