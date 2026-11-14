import { TableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import { useTenantDetails } from '@/hooks/use-tenant';
import { APIToken } from '@/lib/api';
import {
  API_TOKEN_MANAGE_DISABLED_REASON,
  canManageApiTokens,
} from '@/lib/api-token-permissions';

export function TokenActions({
  token,
  onRevokeClick,
}: {
  token: APIToken;
  onRevokeClick: (token: APIToken) => void;
}) {
  const { membership } = useTenantDetails();

  return (
    <TableRowActions
      row={token}
      actions={[
        {
          label: 'Revoke',
          onClick: () => onRevokeClick(token),
          disabled: canManageApiTokens(membership)
            ? false
            : API_TOKEN_MANAGE_DISABLED_REASON,
        },
      ]}
    />
  );
}
