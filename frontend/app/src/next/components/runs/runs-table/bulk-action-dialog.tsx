import { BulkActionDialog } from '@/next/components/ui/dialog/bulk-action-dialog';

interface BulkActionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  action: 'replay' | 'cancel';
  isLoading: boolean;
  onConfirm: () => void;
}

export function RunsBulkActionDialog({
  open,
  onOpenChange,
  action,
  isLoading,
  onConfirm,
}: BulkActionDialogProps) {
  const title = action === 'replay' ? 'Replay All Runs' : 'Cancel All Runs';
  const description =
    action === 'replay'
      ? 'Are you sure you want to replay all runs? This will create new runs for all tasks in the current view.'
      : 'Are you sure you want to cancel all runs? This will stop all running and queued tasks in the current view.';

  return (
    <BulkActionDialog
      open={open}
      onOpenChange={onOpenChange}
      title={title}
      description={description}
      confirmButtonText={title}
      isLoading={isLoading}
      onConfirm={onConfirm}
    />
  );
}
