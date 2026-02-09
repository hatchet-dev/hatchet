import { ButtonProps, Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Spinner } from '@/components/v1/ui/loading.tsx';

interface ConfirmDialogProps {
  title: string;
  description: string | JSX.Element;

  submitLabel: string;
  submitVariant?: ButtonProps['variant'];
  submitDisabled?: boolean;
  cancelLabel?: string;
  className?: string;
  onSubmit: () => void;
  onCancel: () => void;
  isLoading: boolean;
  isOpen: boolean;
}

export function ConfirmDialog({
  className,
  title,
  description,
  submitLabel,
  submitVariant = 'destructive',
  submitDisabled = false,
  cancelLabel = 'Cancel',
  isOpen,
  ...props
}: ConfirmDialogProps) {
  return (
    <Dialog open={isOpen}>
      <DialogContent className="w-fit min-w-[500px] max-w-[80%]">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>
        <div>
          <div className="mb-4 text-sm text-foreground">{description}</div>
          <div className="flex flex-row justify-end gap-4">
            <Button
              variant="ghost"
              onClick={() => {
                props.onCancel();
              }}
            >
              {cancelLabel}
            </Button>
            <Button
              variant={submitVariant}
              onClick={props.onSubmit}
              disabled={props.isLoading || submitDisabled}
            >
              {props.isLoading && <Spinner />}
              {submitLabel}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
