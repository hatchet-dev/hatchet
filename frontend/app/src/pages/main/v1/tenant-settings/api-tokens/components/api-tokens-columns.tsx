import { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '../../../../../components/molecules/data-table/data-table-column-header';
import { APIToken } from '@/lib/api';
import { DataTableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import RelativeDate from '@/components/v1/molecules/relative-date';

export const columns = ({
  onRevokeClick,
}: {
  onRevokeClick: (row: APIToken) => void;
}): ColumnDef<APIToken>[] => {
  return [
    {
      accessorKey: 'name',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Name" />
      ),
      cell: ({ row }) => <div>{row.getValue('name')}</div>,
      enableSorting: false,
      enableHiding: false,
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
          <div>{new Date(row.original.expiresAt).toLocaleDateString()}</div>
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
              label: 'Revoke',
              onClick: () => onRevokeClick(row.original),
            },
          ]}
        />
      ),
    },
  ];
};
