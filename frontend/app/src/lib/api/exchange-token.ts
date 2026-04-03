import { TenantExchangeToken } from '@/lib/api/generated/control-plane/data-contracts';
import { Query } from '@tanstack/react-query';

// Refresh the token 60 seconds before it actually expires.
const EXPIRY_BUFFER_MS = 60_000;

export function exchangeTokenQueryKey(tenantId: string) {
  return ['exchange-token', tenantId] as const;
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
      return Math.max(0, expiry - Date.now() - EXPIRY_BUFFER_MS);
    },
    gcTime: 60 * 60 * 1000,
    retry: 1,
  };
}
