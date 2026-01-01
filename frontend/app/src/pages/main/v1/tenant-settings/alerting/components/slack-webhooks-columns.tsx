import { DataTableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import { SlackWebhook } from '@/lib/api';

export function SlackActions({
  webhook,
  onDeleteClick,
}: {
  webhook: SlackWebhook;
  onDeleteClick: (webhook: SlackWebhook) => void;
}) {
  return (
    <DataTableRowActions
      row={{ original: webhook } as any}
      actions={[
        {
          label: 'Delete',
          onClick: () => onDeleteClick(webhook),
        },
      ]}
    />
  );
}
