import { useLocalStorageState } from '@/hooks/use-local-storage-state';
import { useEffect } from 'react';

const RECENT_TENANTS_KEY = 'hatchet:recent-tenants';
const MAX_RECENT_TENANTS = 3;

/**
 * Tracks the tenants the user most recently switched to, newest first.
 * The current tenant is recorded as it changes but is never returned.
 */
export function useRecentTenants(currentTenantId: string | undefined) {
  const [ids, setIds] = useLocalStorageState<string[]>(RECENT_TENANTS_KEY, []);

  useEffect(() => {
    if (!currentTenantId || ids[0] === currentTenantId) {
      return;
    }
    setIds((prev) =>
      [currentTenantId, ...prev.filter((id) => id !== currentTenantId)].slice(
        0,
        MAX_RECENT_TENANTS + 1,
      ),
    );
    // Only record when the current tenant changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentTenantId]);

  return ids
    .filter((id) => id !== currentTenantId)
    .slice(0, MAX_RECENT_TENANTS);
}
