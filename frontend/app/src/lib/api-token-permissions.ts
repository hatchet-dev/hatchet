export function canManageApiTokens(role?: string): boolean {
  return role === 'OWNER' || role === 'ADMIN';
}

export const API_TOKEN_MANAGE_DISABLED_REASON =
  'Only tenant owners and admins can create or revoke API tokens.';
