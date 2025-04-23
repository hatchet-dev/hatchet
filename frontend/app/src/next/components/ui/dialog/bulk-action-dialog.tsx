import { DestructiveDialog } from './destructive-dialog';

interface BulkActionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: string;
  confirmButtonText: string;
  isLoading?: boolean;
  onConfirm: () => void;
  onCancel?: () => void;
  children?: React.ReactNode;
}

export function BulkActionDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmButtonText,
  isLoading = false,
  onConfirm,
  onCancel,
  children,
}: BulkActionDialogProps) {
  return (
    <DestructiveDialog
      open={open}
      onOpenChange={onOpenChange}
      title={title}
      description={description}
      confirmationText=""
      confirmButtonText={confirmButtonText}
      isLoading={isLoading}
      requireTextConfirmation={false}
      onConfirm={onConfirm}
      onCancel={onCancel}
      hideAlert={true}
    >
      {children}
    </DestructiveDialog>
  );
}
