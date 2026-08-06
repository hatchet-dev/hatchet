import { TableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import { useIsTenantAdmin } from '@/hooks/use-tenant';
import { APIToken } from '@/lib/api';

export function TokenActions({
  token,
  onRevokeClick,
}: {
  token: APIToken;
  onRevokeClick: (token: APIToken) => void;
}) {
  const isTenantAdmin = useIsTenantAdmin();

  return (
    <TableRowActions
      row={token}
      actions={
        isTenantAdmin
          ? [
              {
                label: 'Revoke',
                onClick: () => onRevokeClick(token),
              },
            ]
          : []
      }
    />
  );
}
