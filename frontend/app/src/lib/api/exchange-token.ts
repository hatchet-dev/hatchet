import queryClient from '@/query-client';

export interface TenantTokenData {
  token: string;
  apiUrl: string;
  expiresAt: string;
}

const STORAGE_PREFIX = 'hatchet_xt_';
// Refresh the token 60 seconds before it actually expires so there's
// no window where we use an about-to-expire token.
const EXPIRY_BUFFER_MS = 60_000;

// ── localStorage helpers ──────────────────────────────────────────────────────

function storageKey(tenantId: string): string {
  return `${STORAGE_PREFIX}${tenantId}`;
}

function readStored(tenantId: string): TenantTokenData | null {
  try {
    const raw = localStorage.getItem(storageKey(tenantId));
    if (!raw) return null;
    const stored: TenantTokenData = JSON.parse(raw);
    const expiry = new Date(stored.expiresAt).getTime();
    if (Date.now() >= expiry - EXPIRY_BUFFER_MS) {
      // Evict expired/nearly-expired tokens eagerly so the next read
      // triggers a fresh fetch instead of returning a stale entry.
      localStorage.removeItem(storageKey(tenantId));
      return null;
    }
    return stored;
  } catch {
    // Ignore parse errors and unavailable storage (private browsing, etc.)
    return null;
  }
}

function writeStored(tenantId: string, token: TenantTokenData): void {
  try {
    localStorage.setItem(storageKey(tenantId), JSON.stringify(token));
  } catch {
    // Ignore write failures (private browsing, quota exceeded, etc.)
  }
}

export function clearStoredExchangeToken(tenantId: string): void {
  try {
    localStorage.removeItem(storageKey(tenantId));
  } catch {
    // Ignore
  }
}

// ── React Query integration ───────────────────────────────────────────────────

export function exchangeTokenQueryKey(tenantId: string) {
  return ['exchange-token', tenantId] as const;
}

/**
 * Returns React Query options for fetching/caching an exchange token.
 *
 * `fetchFn` is called only when no valid token exists in localStorage or the
 * React Query cache; callers supply it so that this module stays free of
 * circular imports (it doesn't need to import `controlPlaneApi` from api.ts).
 *
 * Deduplication is handled by React Query — concurrent calls to
 * `queryClient.fetchQuery(exchangeTokenQueryOptions(...))` for the same tenant
 * share a single in-flight request.
 */
export function exchangeTokenQueryOptions(
  tenantId: string,
  fetchFn: () => Promise<TenantTokenData>,
) {
  // TODO-CONTROL-PLANE: this can be removed, we can assume that the apiUrl contains https
  const normalizeApiUrl = (apiUrl: string): string =>
    /^https?:\/\//i.test(apiUrl) ? apiUrl : `http://${apiUrl}`;

  // TODO-CONTROL-PLANE: let's use zod here for typing and unmarshaling?
  const getTokenDataFromQueryState = (
    query: unknown,
  ): TenantTokenData | undefined => {
    if (!query || typeof query !== 'object') {
      return undefined;
    }

    const state = Reflect.get(query, 'state');
    if (!state || typeof state !== 'object') {
      return undefined;
    }

    const data = Reflect.get(state, 'data');
    if (!data || typeof data !== 'object') {
      return undefined;
    }

    const token = Reflect.get(data, 'token');
    const apiUrl = Reflect.get(data, 'apiUrl');
    const expiresAt = Reflect.get(data, 'expiresAt');

    if (
      typeof token !== 'string' ||
      typeof apiUrl !== 'string' ||
      typeof expiresAt !== 'string'
    ) {
      return undefined;
    }

    return {
      token,
      apiUrl,
      expiresAt,
    };
  };

  return {
    queryKey: exchangeTokenQueryKey(tenantId),
    queryFn: async (): Promise<TenantTokenData> => {
      console.log('[exchange-token queryFn] start', { tenantId });

      // Prefer the localStorage copy (survives page refreshes) over a
      // network round-trip when the stored token is still valid.
      const stored = readStored(tenantId);
      if (stored) {
        console.log('[exchange-token queryFn] returning localStorage hit', {
          tenantId,
          apiUrl: stored.apiUrl,
          expiresAt: stored.expiresAt,
        });
        return stored;
      }

      console.log(
        '[exchange-token queryFn] no valid cached token — fetching from CP',
        { tenantId },
      );
      try {
        let token = await fetchFn();
        // Normalise the API URL before storing — the backend may omit the
        // protocol (e.g. "localhost:8888").  We normalise here so that every
        // cached copy (memory + localStorage) already has a valid absolute URL.
        token = { ...token, apiUrl: normalizeApiUrl(token.apiUrl) };
        console.log('[exchange-token queryFn] fetched token', {
          tenantId,
          apiUrl: token.apiUrl,
          expiresAt: token.expiresAt,
        });
        writeStored(tenantId, token);
        return token;
      } catch (err) {
        console.error('[exchange-token queryFn] fetch failed', {
          tenantId,
          err,
        });
        throw err;
      }
    },
    // Dynamically compute staleTime from the token's own expiry so React Query
    // will re-run the queryFn (and refresh the token) just before it expires.
    // TODO-CONTROL-PLANE: why unknown type here?
    staleTime: (query: unknown) => {
      const data = getTokenDataFromQueryState(query);
      if (!data) return 0;
      const expiry = new Date(data.expiresAt).getTime();
      return Math.max(0, expiry - Date.now() - EXPIRY_BUFFER_MS);
    },
    gcTime: 60 * 60 * 1000, // keep in memory for up to 1 hour
    retry: 1,
  };
}

/**
 * Fetches an exchange token for `tenantId`, using `fetchFn` to acquire a new
 * one from the control plane when the cached copy is absent or expired.
 *
 * Wraps `queryClient.fetchQuery` so that:
 *  - concurrent calls for the same tenant are automatically deduplicated
 *  - the result is cached in React Query memory with a staleTime derived from
 *    the token's `expiresAt` field
 *  - localStorage is checked first so the token survives page refreshes
 */
export function fetchExchangeToken(
  tenantId: string,
  fetchFn: () => Promise<TenantTokenData>,
): Promise<TenantTokenData> {
  return queryClient.fetchQuery(exchangeTokenQueryOptions(tenantId, fetchFn));
}
