import { TableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import useCanWrite from '@/hooks/use-can-write';
import { SlackWebhook } from '@/lib/api';

export function SlackActions({
  webhook,
  onDeleteClick,
}: {
  webhook: SlackWebhook;
  onDeleteClick: (webhook: SlackWebhook) => void;
}) {
  const canWrite = useCanWrite();

  return (
    <TableRowActions
      row={webhook}
      actions={
        canWrite
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
