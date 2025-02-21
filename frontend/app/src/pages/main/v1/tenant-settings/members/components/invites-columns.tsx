import { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '../../../../../components/molecules/data-table/data-table-column-header';
import { TenantInvite } from '@/lib/api';
import { DataTableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import { capitalize } from '@/lib/utils';
import RelativeDate from '@/components/v1/molecules/relative-date';

export const columns = ({
  onEditClick,
  onDeleteClick,
}: {
  onEditClick: (row: TenantInvite) => void;
  onDeleteClick: (row: TenantInvite) => void;
}): ColumnDef<TenantInvite>[] => {
  return [
    {
      accessorKey: 'email',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Email" />
      ),
      cell: ({ row }) => <div>{row.getValue('email')}</div>,
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: 'role',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Role" />
      ),
      cell: ({ row }) => <div>{capitalize(row.getValue('role'))}</div>,
    },
    {
      accessorKey: 'created',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Created" />
      ),
      cell: ({ row }) => (
        <div>
          <RelativeDate date={row.original.metadata.createdAt} />
        </div>
      ),
    },
    {
      accessorKey: 'Expires',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Expires" />
      ),
      cell: ({ row }) => {
        return (
          <div>
            <RelativeDate date={row.original.expires} />
          </div>
        );
      },
    },
    {
      id: 'actions',
      cell: ({ row }) => (
        <DataTableRowActions
          row={row}
          actions={[
            {
              label: 'Edit role',
              onClick: () => onEditClick(row.original),
            },
            {
              label: 'Delete',
              onClick: () => onDeleteClick(row.original),
            },
          ]}
        />
      ),
    },
  ];
};
