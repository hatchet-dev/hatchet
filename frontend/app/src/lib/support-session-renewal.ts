import { SupportSession } from '@/lib/api/generated/control-plane/data-contracts';
import { isRetryableRequestError } from '@/lib/query-retry';
import {
  getSupportSession,
  markSupportSessionExpired,
  setSupportSession,
  SUPPORT_SESSION_QUERY_KEY_PREFIX,
  SUPPORT_SESSION_RENEW_BEFORE_MS,
} from '@/lib/support-session';
import { Query } from '@tanstack/react-query';
import { isAxiosError } from 'axios';

export type RenewSupportSession = (token: string) => Promise<SupportSession>;

export function supportSessionNeedsRenewal(
  session: SupportSession,
  now: number = Date.now(),
): boolean {
  const expiry = new Date(session.expiresAt).getTime();
  return (
    Number.isNaN(expiry) || expiry - now <= SUPPORT_SESSION_RENEW_BEFORE_MS
  );
}

/**
 * Shared by the renewal hook and the request interceptor so a renewal in
 * flight is never duplicated. The control plane renews a token that expired
 * while the tab was hidden, as long as the grant is active, so only a
 * rejection marks the session expired.
 */
export function supportSessionQueryOptions(
  grantId: string | undefined,
  renew: RenewSupportSession,
) {
  return {
    queryKey: [SUPPORT_SESSION_QUERY_KEY_PREFIX, grantId],
    queryFn: async (): Promise<SupportSession> => {
      const current = getSupportSession();
      if (!current) {
        throw new Error('no support session');
      }

      if (!supportSessionNeedsRenewal(current)) {
        return current;
      }

      try {
        const renewed = await renew(current.token);
        setSupportSession(renewed);
        return renewed;
      } catch (err) {
        const status = isAxiosError(err) ? err.response?.status : undefined;
        if (status === 401 || status === 403) {
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
    retry: (failureCount: number, error: unknown) =>
      failureCount < 3 && isRetryableRequestError(error),
  };
}
