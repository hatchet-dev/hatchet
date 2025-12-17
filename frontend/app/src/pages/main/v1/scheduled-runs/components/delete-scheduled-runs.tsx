import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Spinner } from '@/components/v1/ui/loading';
import { useCurrentTenantId } from '@/hooks/use-tenant';
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
  scheduledRun,
  setShowScheduledRunRevoke,
  onSuccess,
}: {
  scheduledRun?: ScheduledWorkflows;
  setShowScheduledRunRevoke: (show?: ScheduledWorkflows) => void;
  onSuccess: () => void;
}) {
  const { tenantId } = useCurrentTenantId();
  const { handleApiError } = useApiError({});

  const deleteMutation = useMutation({
    mutationKey: ['scheduled-run:delete', tenantId, scheduledRun],
    mutationFn: async () => {
      if (!scheduledRun) {
        return;
      }
      await api.workflowScheduledDelete(tenantId, scheduledRun.metadata.id);
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

function DeleteScheduledRunForm({
  className,
  isFutureRun,
  ...props
}: DeleteScheduledRunFormProps) {
  return (
    <DialogContent className="w-fit min-w-[500px] max-w-[80%]">
      <DialogHeader>
        <DialogTitle>Delete scheduled run</DialogTitle>
      </DialogHeader>
      <div>
        <div className="mb-4 text-sm text-foreground">
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
        <div className="flex flex-row justify-end gap-4">
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
