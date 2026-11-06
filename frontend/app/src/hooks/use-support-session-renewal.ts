import { controlPlaneApi } from '@/lib/api/api';
import { SupportSession } from '@/lib/api/generated/control-plane/data-contracts';
import {
  isSupportSessionMarkedExpired,
  useSupportSession,
} from '@/lib/support-session';
import { supportSessionQueryOptions } from '@/lib/support-session-renewal';
import { useQuery } from '@tanstack/react-query';

function renewSupportSession(token: string): Promise<SupportSession> {
  return controlPlaneApi
    .supportSessionRenew({ headers: { Authorization: `Bearer ${token}` } })
    .then((r) => r.data);
}

/**
 * Keeps the support token fresh through the query-client refresh triggers
 * (poll while visible, focus, visibilitychange, online). A token past its
 * expiry is still worth renewing: the grant decides, not the clock.
 */
export function useSupportSessionRenewal() {
  const { session, isSupport } = useSupportSession();

  useQuery({
    ...supportSessionQueryOptions(session?.grant.id, renewSupportSession),
    enabled: isSupport && !isSupportSessionMarkedExpired(),
  });
}
