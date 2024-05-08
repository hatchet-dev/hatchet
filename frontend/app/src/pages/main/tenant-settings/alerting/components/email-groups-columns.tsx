import { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '../../../../../components/molecules/data-table/data-table-column-header';
import { TenantAlertEmailGroup } from '@/lib/api';
import { DataTableRowActions } from '@/components/molecules/data-table/data-table-row-actions';
import { Badge } from '@/components/ui/badge';
import RelativeDate from '@/components/molecules/relative-date';

export const columns = ({
  onDeleteClick,
}: {
  onDeleteClick: (row: TenantAlertEmailGroup) => void;
}): ColumnDef<TenantAlertEmailGroup>[] => {
  return [
    {
      accessorKey: 'emails',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Emails" />
      ),
      cell: ({ row }) => (
        <div>
          {row.original.emails.map((email, index) => (
            <Badge key={index} className="mr-2" variant="outline">
              {email}
            </Badge>
          ))}
        </div>
      ),
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
      id: 'actions',
      cell: ({ row }) => (
        <DataTableRowActions
          row={row}
          actions={[
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
