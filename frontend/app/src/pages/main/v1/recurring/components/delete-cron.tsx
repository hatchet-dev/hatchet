import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Spinner } from '@/components/v1/ui/loading';
import api, { CronWorkflows } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { useMutation } from '@tanstack/react-query';

interface DeleteCronFormProps {
  className?: string;
  onCancel: () => void;
  isLoading: boolean;
  onSubmit: () => void;
}

export function DeleteCron({
  tenant,
  cron,
  setShowCronRevoke,
  onSuccess,
}: {
  tenant: string;
  cron?: CronWorkflows;
  setShowCronRevoke: (show?: CronWorkflows) => void;
  onSuccess: () => void;
}) {
  const { handleApiError } = useApiError({});

  const deleteMutation = useMutation({
    mutationKey: ['cron-job:delete', tenant, cron],
    mutationFn: async () => {
      if (!cron) {
        return;
      }
      await api.workflowCronDelete(tenant, cron.metadata.id);
    },
    onSuccess: onSuccess,
    onError: handleApiError,
  });

  return (
    <Dialog
      open={!!cron}
      onOpenChange={(open) => setShowCronRevoke(open ? cron : undefined)}
    >
      <DeleteCronForm
        isLoading={deleteMutation.isPending}
        onSubmit={() => deleteMutation.mutate()}
        onCancel={() => setShowCronRevoke(undefined)}
      />
    </Dialog>
  );
}

export function DeleteCronForm({ className, ...props }: DeleteCronFormProps) {
  return (
    <DialogContent className="w-fit max-w-[80%] min-w-[500px]">
      <DialogHeader>
        <DialogTitle>Delete cron job</DialogTitle>
      </DialogHeader>
      <div>
        <div className="text-sm text-foreground mb-4">
          Are you sure you want to delete the cron job? This action will prevent
          the run from running in the future and cannot be undone.
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
            Delete cron job
          </Button>
        </div>
      </div>
    </DialogContent>
  );
}
