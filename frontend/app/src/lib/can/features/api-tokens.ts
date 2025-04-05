import { PermissionSet, RejectReason } from '@/lib/can';
import { TenantMemberRole } from '@/lib/api';

export const apiTokens: PermissionSet = {
  manage:
    () =>
    ({ membership }) => {
      const allowed = [
        TenantMemberRole.OWNER,
        TenantMemberRole.ADMIN,
        TenantMemberRole.MEMBER,
      ];
      if (!membership || !allowed.includes(membership)) {
        return {
          allowed: false,
          reason: RejectReason.ROLE_REQUIRED,
          message:
            'You must be an owner, admin, or member to manage API tokens.',
        };
      }

      return {
        allowed: true,
      };
    },
};
