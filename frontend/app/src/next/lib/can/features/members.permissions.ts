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
  updateRole:
    (params: { targetMember: TenantMember; newRole: TenantMemberRole }) =>
    ({ membership, user }) => {
      if (!membership) {
        return {
          allowed: false,
          rejectReason: RejectReason.ROLE_REQUIRED,
          message: 'You must be logged in to update member roles',
        };
      }

      // Cannot update your own role
      if (user?.email === params.targetMember.user.email) {
        return {
          allowed: false,
          message: 'You cannot update your own role',
        };
      }

      // Must be at least ADMIN to update roles
      if (RANK[membership] < RANK[TenantMemberRole.ADMIN]) {
        return {
          allowed: false,
          message: 'You do not have permission to update member roles',
        };
      }

      // Cannot elevate someone to a role higher than your own
      if (RANK[params.newRole] > RANK[membership]) {
        return {
          allowed: false,
          message: `You cannot assign the ${params.newRole.toLowerCase()} role`,
        };
      }

      // Cannot modify the role of someone with equal or higher rank (unless you're owner)
      if (
        RANK[params.targetMember.role] >= RANK[membership] &&
        membership !== TenantMemberRole.OWNER
      ) {
        return {
          allowed: false,
          message: 'You cannot modify the role of this member',
        };
      }

      return { allowed: true };
    },
};
