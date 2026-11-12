import useControlPlane from '@/hooks/use-control-plane';
import { supportGrantsQuery } from '@/lib/api/api';
import { SupportGrant } from '@/lib/api/generated/control-plane/data-contracts';
import {
  markSupportSessionEnded,
  resolveSupportSession,
  useEndedSupportGrants,
} from '@/lib/support-session';
import { useQuery } from '@tanstack/react-query';
import { useEffect, useRef } from 'react';

/**
 * The signed-in user's active support grants, polled so a revocation from hat
 * or another tab reaches the banner without a request to the shard.
 */
export function useSupportGrants() {
  const { isControlPlaneEnabled, isControlPlaneLoading } = useControlPlane();
  const query = useQuery({
    ...supportGrantsQuery,
    enabled: isControlPlaneEnabled && !isControlPlaneLoading,
    refetchInterval: 30_000,
  });

  // A grant that drops out of the list ended: remember it so its tenant shows
  // the expired banner rather than a bare 403.
  const previous = useRef<SupportGrant[] | undefined>(undefined);
  useEffect(() => {
    const current = query.data;
    if (!current) {
      return;
    }
    previous.current?.forEach((grant) => {
      if (!current.some((g) => g.id === grant.id)) {
        markSupportSessionEnded(grant);
      }
    });
    previous.current = current;
  }, [query.data]);

  return query;
}

export function useSupportSession(tenantId: string | undefined) {
  const grants = useSupportGrants();
  useEndedSupportGrants();

  return {
    ...resolveSupportSession(tenantId, grants.data),
    hasSupportGrants: (grants.data?.length ?? 0) > 0,
  };
}
