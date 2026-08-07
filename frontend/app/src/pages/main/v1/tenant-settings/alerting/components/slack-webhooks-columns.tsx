import { TableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import { useIsTenantAdmin } from '@/hooks/use-tenant';
import { SlackWebhook } from '@/lib/api';

export function SlackActions({
  webhook,
  onDeleteClick,
}: {
  webhook: SlackWebhook;
  onDeleteClick: (webhook: SlackWebhook) => void;
}) {
  const isTenantAdmin = useIsTenantAdmin();

  return (
    <TableRowActions
      row={webhook}
      actions={
        isTenantAdmin
          ? [
              {
                label: 'Delete',
                onClick: () => onDeleteClick(webhook),
              },
            ]
          : []
      }
    />
  );
}
