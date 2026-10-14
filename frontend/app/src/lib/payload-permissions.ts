import type { TenantMember } from '@/lib/api/generated/data-contracts';

// OWNER and ADMIN always see payloads; the canViewPayloads flag is ignored.
export function payloadsLockedForRole(role?: string) {
  return role === 'OWNER' || role === 'ADMIN';
}

// An unknown membership (still loading) counts as allowed: the backend omits
// payloads for restricted users regardless.
export function canViewPayloadsForMember(
  member?: Pick<TenantMember, 'role' | 'canViewPayloads'>,
): boolean {
  return (
    payloadsLockedForRole(member?.role) || member?.canViewPayloads !== false
  );
}
