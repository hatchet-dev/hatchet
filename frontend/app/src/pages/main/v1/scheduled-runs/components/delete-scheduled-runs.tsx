import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Spinner } from '@/components/v1/ui/loading';
import api, { ScheduledWorkflows } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { useMutation } from '@tanstack/react-query';

interface DeleteScheduledRunFormProps {
  isFutureRun: boolean;
  className?: string;
  onCancel: () => void;
  isLoading: boolean;
  onSubmit: () => void;
}

export function DeleteScheduledRun({
  tenant,
  scheduledRun,
  setShowScheduledRunRevoke,
  onSuccess,
}: {
  tenant: string;
  scheduledRun?: ScheduledWorkflows;
  setShowScheduledRunRevoke: (show?: ScheduledWorkflows) => void;
  onSuccess: () => void;
}) {
  const { handleApiError } = useApiError({});

  const deleteMutation = useMutation({
    mutationKey: ['scheduled-run:delete', tenant, scheduledRun],
    mutationFn: async () => {
      if (!scheduledRun) {
        return;
      }
      await api.workflowScheduledDelete(tenant, scheduledRun.metadata.id);
    },
    onSuccess: onSuccess,
    onError: handleApiError,
  });

  return (
    <Dialog
      open={!!scheduledRun}
      onOpenChange={(open) =>
        setShowScheduledRunRevoke(open ? scheduledRun : undefined)
      }
    >
      <DeleteScheduledRunForm
        isLoading={deleteMutation.isPending}
        onSubmit={() => deleteMutation.mutate()}
        onCancel={() => setShowScheduledRunRevoke(undefined)}
        isFutureRun={
          scheduledRun?.triggerAt
            ? new Date(scheduledRun.triggerAt) > new Date()
            : false
        }
      />
    </Dialog>
  );
}

export function DeleteScheduledRunForm({
  className,
  isFutureRun,
  ...props
}: DeleteScheduledRunFormProps) {
  return (
    <DialogContent className="w-fit max-w-[80%] min-w-[500px]">
      <DialogHeader>
        <DialogTitle>Delete scheduled run</DialogTitle>
      </DialogHeader>
      <div>
        <div className="text-sm text-foreground mb-4">
          Are you sure you want to delete the scheduled run?
          {isFutureRun ? (
            <>
              This action will prevent the run from running in the future and
              cannot be undone.
            </>
          ) : (
            <>
              This action will delete the scheduled run trigger, but will not
              affect the run itself and cannot be undone.
            </>
          )}
        </div>
        <div className="flex flex-row gap-4 justify-end">
          <Button
            variant="ghost"
            onClick={() => {
              props.onCancel();
            }}
          >
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={() => {
              props.onSubmit();
            }}
          >
            {props.isLoading && <Spinner />}
            Delete scheduled run
          </Button>
        </div>
      </div>
    </DialogContent>
  );
}
