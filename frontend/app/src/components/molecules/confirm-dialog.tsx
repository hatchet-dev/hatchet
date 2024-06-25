import { Button, ButtonProps } from '@/components/ui/button';
import { Spinner } from '@/components/ui/loading.tsx';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';

interface ConfirmDialogProps {
  title: string;
  description: string | JSX.Element;

  submitLabel: string;
  submitVariant?: ButtonProps['variant'];
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
  cancelLabel = 'Cancel',
  isOpen,
  ...props
}: ConfirmDialogProps) {
  return (
    <Dialog open={isOpen}>
      <DialogContent className="w-fit max-w-[80%] min-w-[500px]">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>
        <div>
          <div className="text-sm text-foreground mb-4">{description}</div>
          <div className="flex flex-row gap-4 justify-end">
            <Button
              variant="ghost"
              onClick={() => {
                props.onCancel();
              }}
            >
              {cancelLabel}
            </Button>
            <Button variant={submitVariant} onClick={props.onSubmit}>
              {props.isLoading && <Spinner />}
              {submitLabel}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
