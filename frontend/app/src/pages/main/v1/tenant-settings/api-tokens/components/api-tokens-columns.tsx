import { TableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import useCanWrite from '@/hooks/use-can-write';
import { APIToken } from '@/lib/api';

export function TokenActions({
  token,
  onRevokeClick,
}: {
  token: APIToken;
  onRevokeClick: (token: APIToken) => void;
}) {
  const canWrite = useCanWrite();

  return (
    <TableRowActions
      row={token}
      actions={
        canWrite
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
