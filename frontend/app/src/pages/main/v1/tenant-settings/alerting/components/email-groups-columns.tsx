import { TableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import { Badge } from '@/components/v1/ui/badge';
import { TenantAlertEmailGroup } from '@/lib/api';

export function EmailGroupCell({ group }: { group: TenantAlertEmailGroup }) {
  return (
    <div>
      {group.metadata.id == 'default' && (
        <Badge className="mr-2" variant="secondary">
          All Tenant Members
        </Badge>
      )}
      {group.emails.map((email, index) => (
        <Badge key={index} className="mr-2" variant="outline">
          {email}
        </Badge>
      ))}
    </div>
  );
}

export function EmailGroupStatusCell({
  group,
  alertTenantEmailsSet,
}: {
  group: TenantAlertEmailGroup;
  alertTenantEmailsSet: boolean;
}) {
  return (
    <div className="flex items-center justify-end space-x-2">
      {group.metadata.id != 'default' || alertTenantEmailsSet ? (
        <Badge variant="successful">Enabled</Badge>
      ) : (
        <Badge variant="destructive">Disabled</Badge>
      )}
    </div>
  );
}

export function EmailGroupActions({
  group,
  alertTenantEmailsSet,
  onDeleteClick,
  onToggleMembersClick,
}: {
  group: TenantAlertEmailGroup;
  alertTenantEmailsSet: boolean;
  onDeleteClick: (group: TenantAlertEmailGroup) => void;
  onToggleMembersClick: (val: boolean) => void;
}) {
  return (
    <div className="mr-4 flex items-center justify-end space-x-2">
      <TableRowActions
        row={group}
        actions={[
          group.metadata.id != 'default'
            ? {
                label: 'Delete',
                onClick: () => onDeleteClick(group),
              }
            : {
                label: alertTenantEmailsSet ? 'Disable' : 'Enable',
                onClick: () => onToggleMembersClick(!alertTenantEmailsSet),
              },
        ]}
      />
    </div>
  );
}
