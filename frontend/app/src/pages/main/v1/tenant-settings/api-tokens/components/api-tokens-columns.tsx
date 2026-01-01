import { DataTableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import { APIToken } from '@/lib/api';

export function TokenActions({
  token,
  onRevokeClick,
}: {
  token: APIToken;
  onRevokeClick: (token: APIToken) => void;
}) {
  return (
    <DataTableRowActions
      row={{ original: token } as any}
      actions={[
        {
          label: 'Revoke',
          onClick: () => onRevokeClick(token),
        },
      ]}
    />
  );
}
