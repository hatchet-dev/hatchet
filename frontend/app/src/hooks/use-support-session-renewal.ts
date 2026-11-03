import { controlPlaneApi, getApiErrorStatus } from '@/lib/api/api';
import { SupportSession } from '@/lib/api/generated/control-plane/data-contracts';
import { isRetryableRequestError } from '@/lib/query-retry';
import {
  getSupportSession,
  isSupportSessionExpired,
  markSupportSessionExpired,
  setSupportSession,
  SUPPORT_SESSION_QUERY_KEY_PREFIX,
  SUPPORT_SESSION_RENEW_BEFORE_MS,
  useSupportSession,
} from '@/lib/support-session';
import { Query, useQuery } from '@tanstack/react-query';

/**
 * Renews the support token once it is within SUPPORT_SESSION_RENEW_BEFORE_MS of expiry,
 * through the query-client refresh triggers. A rejected renewal marks the session
 * expired, which disables this query.
 */
export function useSupportSessionRenewal() {
  const { session, isSupport, isExpired } = useSupportSession();

  useQuery({
    queryKey: [SUPPORT_SESSION_QUERY_KEY_PREFIX, session?.grant.id],
    enabled: isSupport && !isExpired,
    queryFn: async (): Promise<SupportSession> => {
      const current = getSupportSession();
      if (!current) {
        throw new Error('no support session');
      }

      const expiry = new Date(current.expiresAt).getTime();
      if (expiry - Date.now() > SUPPORT_SESSION_RENEW_BEFORE_MS) {
        return current;
      }

      try {
        const renewed = (
          await controlPlaneApi.supportSessionRenew({
            headers: { Authorization: `Bearer ${current.token}` },
          })
        ).data;
        setSupportSession(renewed);
        return renewed;
      } catch (err) {
        const status = getApiErrorStatus(err);
        if (
          status === 401 ||
          status === 403 ||
          isSupportSessionExpired(current)
        ) {
          markSupportSessionExpired();
        }
        throw err;
      }
    },
    staleTime: (query: Query<SupportSession>) => {
      const data = query.state.data;
      if (!data) {
        return 0;
      }
      const expiry = new Date(data.expiresAt).getTime();
      return Math.max(
        0,
        expiry - query.state.dataUpdatedAt - SUPPORT_SESSION_RENEW_BEFORE_MS,
      );
    },
    gcTime: Infinity,
    retry: (failureCount, error) =>
      failureCount < 3 && isRetryableRequestError(error),
  });
}
