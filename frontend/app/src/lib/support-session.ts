import {
  SupportGrant,
  SupportGrantRole,
  SupportSession,
} from '@/lib/api/generated/control-plane/data-contracts';
import { isRetryableRequestError } from '@/lib/query-retry';
import { Query } from '@tanstack/react-query';
import { isAxiosError } from 'axios';
import { useSyncExternalStore } from 'react';

// Redeeming a support link signs the staff member in with a control-plane
// cookie session. From there support tokens are fetched per tenant like
// exchange tokens and cached in memory, so a support tenant opens in any tab
// and nothing about the session touches browser storage.

export const SUPPORT_GRANTS_QUERY_KEY = ['support-grants'] as const;
export const SUPPORT_TOKEN_QUERY_KEY_PREFIX = 'support-token';

// Refresh the token 60 seconds before it actually expires.
const TOKEN_EXPIRY_BUFFER_MS = 60_000;

export function supportGrantsQueryOptions(
  fetchFn: () => Promise<SupportGrant[]>,
) {
  return {
    queryKey: SUPPORT_GRANTS_QUERY_KEY,
    queryFn: fetchFn,
    staleTime: 30_000,
    gcTime: 60 * 60 * 1000,
  };
}

export function findSupportGrant(
  grants: readonly SupportGrant[] | undefined,
  tenantId: string | undefined,
): SupportGrant | null {
  if (!grants || !tenantId) {
    return null;
  }
  return grants.find((grant) => grant.tenantId === tenantId) ?? null;
}

export function supportTokenQueryKey(tenantId: string) {
  return [SUPPORT_TOKEN_QUERY_KEY_PREFIX, tenantId] as const;
}

/**
 * Token fetch for one grant. A 401/403 means the grant is revoked or expired,
 * which is remembered so the banner can explain instead of the tenant
 * silently vanishing.
 */
export function supportTokenQueryOptions(
  grant: SupportGrant,
  fetchFn: () => Promise<SupportSession>,
) {
  return {
    queryKey: supportTokenQueryKey(grant.tenantId),
    queryFn: async (): Promise<SupportSession> => {
      try {
        return await fetchFn();
      } catch (err) {
        const status = isAxiosError(err) ? err.response?.status : undefined;
        if (status === 401 || status === 403) {
          markSupportSessionEnded(grant);
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
        expiry - query.state.dataUpdatedAt - TOKEN_EXPIRY_BUFFER_MS,
      );
    },
    gcTime: 60 * 60 * 1000,
    retry: (failureCount: number, error: unknown) =>
      failureCount < 3 && isRetryableRequestError(error),
  };
}

// Grants that ended while the tab was on their tenant, by tenant id. Kept in
// memory only: a reload re-reads the live grant list.
const ended = new Map<string, SupportGrant>();
const listeners = new Set<() => void>();
let endedVersion = 0;

function notify() {
  endedVersion += 1;
  listeners.forEach((listener) => listener());
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function markSupportSessionEnded(grant: SupportGrant) {
  const current = ended.get(grant.tenantId);
  if (current?.id === grant.id) {
    return;
  }
  ended.set(grant.tenantId, grant);
  notify();
}

export function getEndedSupportGrant(
  tenantId: string | undefined,
): SupportGrant | null {
  return (tenantId && ended.get(tenantId)) || null;
}

export function clearEndedSupportGrants() {
  if (ended.size === 0) {
    return;
  }
  ended.clear();
  notify();
}

/**
 * The support session on a tenant, from the live grants and the ended ones.
 * A newer grant on the same tenant (hat support run again) supersedes an
 * ended one.
 */
export function resolveSupportSession(
  tenantId: string | undefined,
  grants: readonly SupportGrant[] | undefined,
):
  | { grant: SupportGrant; isSupport: true; isExpired: boolean }
  | { grant: null; isSupport: false; isExpired: false } {
  const live = findSupportGrant(grants, tenantId);
  const lapsed = getEndedSupportGrant(tenantId);

  if (live && live.id !== lapsed?.id) {
    return { grant: live, isSupport: true, isExpired: false };
  }
  if (lapsed) {
    return { grant: lapsed, isSupport: true, isExpired: true };
  }
  return { grant: null, isSupport: false, isExpired: false };
}

export function useEndedSupportGrants() {
  return useSyncExternalStore(
    subscribe,
    () => endedVersion,
    () => endedVersion,
  );
}

export function supportRoleLabel(role: SupportGrantRole): string {
  return role === SupportGrantRole.SUPPORT_OPERATOR ? 'operator' : 'viewer';
}
