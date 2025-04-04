import {
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Button } from '@/components/v1/ui/button';
import { cn } from '@/lib/utils';
import { Spinner } from '@/components/v1/ui/loading';

interface DeleteWebhookWorkerDialogProps {
  onSubmit: () => void;
  className?: string;
  isLoading: boolean;
  fieldErrors?: Record<string, string>;
}

export function DeleteWebhookWorkerDialog({
  onSubmit,
  className,
  ...props
}: DeleteWebhookWorkerDialogProps) {
  return (
    <DialogContent className="w-fit max-w-[80%] min-w-[500px]">
      <DialogHeader>
        <DialogTitle>Delete Webhook Worker?</DialogTitle>
      </DialogHeader>
      <div className={cn('grid gap-6', className)}>
        <div className="grid gap-4">
          This is a permanent action. Are you sure you want to delete this
          webhook worker?
          <Button
            onClick={() => {
              onSubmit();
            }}
          >
            {props.isLoading && <Spinner />}
            Delete
          </Button>
        </div>
      </div>
    </DialogContent>
  );
}
