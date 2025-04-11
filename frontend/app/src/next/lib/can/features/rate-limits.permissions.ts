import { PermissionSet, RejectReason } from '@/next/lib/can';
import { TenantMemberRole } from '@/next/lib/api';
import { RANK } from '@/next/lib/can/features/members.permissions';

export const rateLimits: PermissionSet = {
  manage:
    () =>
    ({ membership }) => {
      if (!membership || RANK[membership] < RANK[TenantMemberRole.MEMBER]) {
        return {
          allowed: false,
          reason: RejectReason.ROLE_REQUIRED,
          message:
            'You must be an owner, admin, or member to manage rate limits.',
        };
      }

      return {
        allowed: true,
      };
    },
};
