import { TableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import { APIToken } from '@/lib/api';

export function TokenActions({
  token,
  onRevokeClick,
}: {
  token: APIToken;
  onRevokeClick: (token: APIToken) => void;
}) {
  return (
    <TableRowActions
      row={token}
      actions={[
        {
          label: 'Revoke',
          onClick: () => onRevokeClick(token),
        },
      ]}
    />
  );
}
