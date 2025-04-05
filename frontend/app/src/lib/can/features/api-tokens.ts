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

      if (!membership) {
        return {
          allowed: false,
          reason: RejectReason.ROLE_REQUIRED,
          message:
            'No membership found. You need to be a tenant member to manage API tokens.',
        };
      }

      console.log('membership', membership);

      if (!allowed.includes(membership)) {
        return {
          allowed: false,
          reason: RejectReason.ROLE_REQUIRED,
          message:
            'You need to have Member or Admin role to manage API tokens.',
        };
      }

      return {
        allowed: true,
      };
    },
};
