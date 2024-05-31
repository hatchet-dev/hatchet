import { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '../../../../../components/molecules/data-table/data-table-column-header';
import { TenantAlertEmailGroup } from '@/lib/api';
import { DataTableRowActions } from '@/components/molecules/data-table/data-table-row-actions';
import { Badge } from '@/components/ui/badge';
import RelativeDate from '@/components/molecules/relative-date';
import { Switch } from '@/components/ui/switch';
import { Label } from '@radix-ui/react-label';

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
      id: 'actions',
      cell: ({ row }) => (
        <div className="flex items-center space-x-2 justify-end">
          {row.original.metadata.id != 'default' ? (
            <DataTableRowActions
              row={row}
              actions={[
                {
                  label: 'Delete',
                  onClick: () => onDeleteClick(row.original),
                },
              ]}
            />
          ) : (
            <>
              <Switch
                id="eta"
                checked={alertTenantEmailsSet}
                aria-label="Toggle Member Email Alerts"
                onClick={() => {
                  onToggleMembersClick(!alertTenantEmailsSet);
                }}
              />
              <Label htmlFor="eta">
                {alertTenantEmailsSet ? 'Enabled' : 'Disabled'}
              </Label>
            </>
          )}
        </div>
      ),
    },
  ];
};
