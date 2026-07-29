import { TenantExchangeToken } from '@/lib/api/generated/control-plane/data-contracts';
import { isRetryableRequestError, NonRetryableError } from '@/lib/query-retry';
import { Query } from '@tanstack/react-query';
import { isAxiosError } from 'axios';

// Refresh the token 60 seconds before it actually expires.
const EXPIRY_BUFFER_MS = 60_000;
const RETRY_BASE_DELAY_MS = 1_000;
const RETRY_MAX_DELAY_MS = 10_000;
export const EXCHANGE_TOKEN_QUERY_KEY_PREFIX = 'exchange-token';

/**
 * Thrown by the exchange-token interceptor when the token could not be
 * fetched. Extends NonRetryableError because the token layer has its own
 * retry budget — retrying the outer request would just restart the whole
 * token retry sequence.
 */
export class ExchangeTokenFetchError extends NonRetryableError {
  readonly cause: unknown;

  // Mirrors the underlying HTTP status (if any) so getApiErrorStatus works.
  readonly status?: number;

  constructor(tenantId: string, cause: unknown) {
    super(`failed to fetch exchange token for tenant ${tenantId}`);
    this.name = 'ExchangeTokenFetchError';
    this.cause = cause;
    if (isAxiosError(cause)) {
      this.status = cause.response?.status;
    }
  }
}

function shouldRetryTokenFetch(failureCount: number, error: unknown) {
  if (failureCount >= 5) {
    return false;
  }

  // Unlike other 4xx, 404 gets a couple of quick retries: the token endpoint
  // has been observed to intermittently 404 in production.
  if (isAxiosError(error) && error.response?.status === 404) {
    return failureCount < 2;
  }

  return isRetryableRequestError(error);
}

export function exchangeTokenQueryKey(tenantId: string) {
  return [EXCHANGE_TOKEN_QUERY_KEY_PREFIX, tenantId] as const;
}

/**
 * Returns React Query options for fetching/caching an exchange token.
 *
 * `fetchFn` is called only when no valid token exists in the React Query cache
 * or the cached token is stale. Callers supply it so that this module stays
 * free of circular imports (it doesn't need to import `controlPlaneApi`).
 *
 * `staleTime` is derived from the token's own `expiresAt` so React Query
 * automatically re-fetches just before expiry without any manual scheduling.
 */
export function exchangeTokenQueryOptions(
  tenantId: string,
  fetchFn: () => Promise<TenantExchangeToken>,
) {
  return {
    queryKey: exchangeTokenQueryKey(tenantId),
    queryFn: (): Promise<TenantExchangeToken> => fetchFn(),
    staleTime: (query: Query<TenantExchangeToken>) => {
      const data = query.state.data;
      if (!data) {
        return 0;
      }
      const expiry = new Date(data.expiresAt).getTime();
      // staleTime is a duration measured from dataUpdatedAt, so it must be
      // computed relative to that timestamp. Subtracting Date.now() here
      // would double-count elapsed time and mark tokens stale at half their
      // intended lifetime.
      return Math.max(0, expiry - query.state.dataUpdatedAt - EXPIRY_BUFFER_MS);
    },
    gcTime: 60 * 60 * 1000,
    retry: shouldRetryTokenFetch,
    retryDelay: (attemptIndex: number) =>
      Math.min(RETRY_BASE_DELAY_MS * 2 ** attemptIndex, RETRY_MAX_DELAY_MS),
  };
}
