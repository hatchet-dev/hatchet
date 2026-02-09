import { TableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import { SlackWebhook } from '@/lib/api';

export function SlackActions({
  webhook,
  onDeleteClick,
}: {
  webhook: SlackWebhook;
  onDeleteClick: (webhook: SlackWebhook) => void;
}) {
  return (
    <TableRowActions
      row={webhook}
      actions={[
        {
          label: 'Delete',
          onClick: () => onDeleteClick(webhook),
        },
      ]}
    />
  );
}
