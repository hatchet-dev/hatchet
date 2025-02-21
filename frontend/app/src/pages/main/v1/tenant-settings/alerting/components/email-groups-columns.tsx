import { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '../../../../../components/molecules/data-table/data-table-column-header';
import { TenantAlertEmailGroup } from '@/lib/api';
import { DataTableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import { Badge } from '@/components/v1/ui/badge';
import RelativeDate from '@/components/v1/molecules/relative-date';

export const columns = ({
  alertTenantEmailsSet,
  onDeleteClick,
  onToggleMembersClick,
}: {
  alertTenantEmailsSet: boolean;
  onDeleteClick: (row: TenantAlertEmailGroup) => void;
  onToggleMembersClick: (val: boolean) => void;
}): ColumnDef<TenantAlertEmailGroup>[] => {
  return [
    {
      accessorKey: 'emails',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Emails" />
      ),
      cell: ({ row }) => (
        <div>
          {row.original.metadata.id == 'default' && (
            <Badge className="mr-2" variant="secondary">
              All Tenant Members
            </Badge>
          )}
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
          {row.original.metadata.id != 'default' && (
            <RelativeDate date={row.original.metadata.createdAt} />
          )}
        </div>
      ),
    },
    {
      id: 'enabled',
      cell: ({ row }) => (
        <div className="flex items-center space-x-2 justify-end">
          {row.original.metadata.id != 'default' || alertTenantEmailsSet ? (
            <Badge variant="successful">Enabled</Badge>
          ) : (
            <Badge variant="destructive">Disabled</Badge>
          )}
        </div>
      ),
    },
    {
      id: 'actions',
      cell: ({ row }) => (
        <div className="flex items-center space-x-2 justify-end mr-4">
          <DataTableRowActions
            row={row}
            actions={[
              row.original.metadata.id != 'default'
                ? {
                    label: 'Delete',
                    onClick: () => onDeleteClick(row.original),
                  }
                : {
                    label: alertTenantEmailsSet ? 'Disable' : 'Enable',
                    onClick: () => onToggleMembersClick(!alertTenantEmailsSet),
                  },
            ]}
          />
        </div>
      ),
    },
  ];
};
