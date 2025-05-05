import { PermissionSet, RejectReason } from '@/next/lib/can';
import { TenantMember, TenantMemberRole } from '@/lib/api';

export const RANK: Record<TenantMemberRole, number> = {
  [TenantMemberRole.MEMBER]: 0,
  [TenantMemberRole.ADMIN]: 1,
  [TenantMemberRole.OWNER]: 2,
};

export const members: PermissionSet = {
  view:
    () =>
    ({ membership }) => {
      if (!membership) {
        return {
          allowed: false,
          rejectReason: RejectReason.ROLE_REQUIRED,
          message: 'You must be logged in to view members',
        };
      }

      if (RANK[membership] >= RANK[TenantMemberRole.ADMIN]) {
        return { allowed: true };
      }

      return {
        allowed: false,
        rejectReason: RejectReason.ROLE_REQUIRED,
        message: 'You do not have permission to view members',
      };
    },
  remove:
    (targetMember: TenantMember) =>
    ({ membership, user }) => {
      if (!membership) {
        return {
          allowed: false,
          rejectReason: RejectReason.ROLE_REQUIRED,
          message: 'You must be logged in to remove members',
        };
      }

      // No one can remove themselves
      if (user?.email === targetMember.user.email) {
        return {
          allowed: false,
          message: 'You cannot remove yourself',
        };
      }

      // Logic for role-based permissions:
      if (RANK[membership] >= RANK[targetMember.role]) {
        return { allowed: true };
      }
      // Default deny for unhandled cases
      return {
        allowed: false,
        message: 'You do not have permission to remove this member',
      };
    },
  invite:
    (role: TenantMemberRole) =>
    ({ membership }) => {
      if (!membership) {
        return {
          allowed: false,
          rejectReason: RejectReason.ROLE_REQUIRED,
          message: 'You must be logged in to invite members',
        };
      }

      if (
        RANK[membership] >= RANK[role] &&
        RANK[membership] >= RANK[TenantMemberRole.ADMIN]
      ) {
        return { allowed: true };
      }

      return {
        allowed: false,
        message: 'You do not have permission to invite members',
      };
    },
};
