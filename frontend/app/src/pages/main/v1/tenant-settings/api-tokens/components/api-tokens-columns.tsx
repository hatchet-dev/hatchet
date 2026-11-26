import { TableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import { useTenantRoles } from '@/hooks/use-tenant-roles';
import { APIToken } from '@/lib/api';

export const API_TOKEN_MANAGE_DISABLED_REASON =
  'Your role in this tenant does not allow creating or revoking API tokens.';

export function TokenActions({
  token,
  onRevokeClick,
}: {
  token: APIToken;
  onRevokeClick: (token: APIToken) => void;
}) {
  const { can } = useTenantRoles();

  return (
    <TableRowActions
      row={token}
      actions={[
        {
          label: 'Revoke',
          onClick: () => onRevokeClick(token),
          disabled: can('ApiTokenUpdateRevoke')
            ? false
            : API_TOKEN_MANAGE_DISABLED_REASON,
        },
      ]}
    />
  );
}
