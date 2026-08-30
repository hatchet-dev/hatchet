import { useTenantDetails } from '@/hooks/use-tenant';
import { TenantMemberRole } from '@/lib/api';

/**
 * True when the current user's role in the active tenant allows write actions (trigger, create,
 * update, delete, cancel, replay, rerun, etc.). False only for VIEWER. Defaults to true while
 * membership hasn't loaded yet, so UI doesn't flash into a disabled state before the real role
 * is known — the backend RBAC check is the actual enforcement boundary regardless.
 */
export default function useCanWrite(): boolean {
  const { membership } = useTenantDetails();
  return membership !== TenantMemberRole.VIEWER;
}
