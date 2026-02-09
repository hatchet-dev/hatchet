import { TableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import { TenantInvite } from '@/lib/api';

export function InviteActions({
  invite,
  onEditClick,
  onDeleteClick,
}: {
  invite: TenantInvite;
  onEditClick: (invite: TenantInvite) => void;
  onDeleteClick: (invite: TenantInvite) => void;
}) {
  return (
    <TableRowActions
      row={invite}
      actions={[
        {
          label: 'Edit role',
          onClick: () => onEditClick(invite),
        },
        {
          label: 'Delete',
          onClick: () => onDeleteClick(invite),
        },
      ]}
    />
  );
}
