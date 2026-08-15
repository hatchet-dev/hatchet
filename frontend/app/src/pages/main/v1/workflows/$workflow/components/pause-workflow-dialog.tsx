import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Label } from '@/components/v1/ui/label';
import { Spinner } from '@/components/v1/ui/loading.tsx';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { WorkflowPauseScheduledCronRunQueueBehavior } from '@/lib/api';
import { useState } from 'react';

const QUEUE_BEHAVIOR_LABELS: Record<
  WorkflowPauseScheduledCronRunQueueBehavior,
  string
> = {
  [WorkflowPauseScheduledCronRunQueueBehavior.QUEUE]:
    'Queue runs to execute once unpaused',
  [WorkflowPauseScheduledCronRunQueueBehavior.DROP]:
    'Drop runs triggered while paused',
};

interface PauseWorkflowDialogProps {
  isOpen: boolean;
  isLoading: boolean;
  onCancel: () => void;
  onSubmit: (behavior: {
    cronRunQueueBehavior: WorkflowPauseScheduledCronRunQueueBehavior;
    scheduledRunQueueBehavior: WorkflowPauseScheduledCronRunQueueBehavior;
  }) => void;
}

function QueueBehaviorSelect({
  id,
  value,
  onValueChange,
}: {
  id: string;
  value: WorkflowPauseScheduledCronRunQueueBehavior;
  onValueChange: (value: WorkflowPauseScheduledCronRunQueueBehavior) => void;
}) {
  return (
    <Select
      value={value}
      onValueChange={(value) =>
        onValueChange(value as WorkflowPauseScheduledCronRunQueueBehavior)
      }
    >
      <SelectTrigger id={id}>
        <SelectValue placeholder="Select a behavior" />
      </SelectTrigger>
      <SelectContent>
        {Object.entries(QUEUE_BEHAVIOR_LABELS).map(([value, label]) => (
          <SelectItem key={value} value={value}>
            {label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

export function PauseWorkflowDialog({
  isOpen,
  isLoading,
  onCancel,
  onSubmit,
}: PauseWorkflowDialogProps) {
  const [cronRunQueueBehavior, setCronRunQueueBehavior] =
    useState<WorkflowPauseScheduledCronRunQueueBehavior>(
      WorkflowPauseScheduledCronRunQueueBehavior.QUEUE,
    );
  const [scheduledRunQueueBehavior, setScheduledRunQueueBehavior] =
    useState<WorkflowPauseScheduledCronRunQueueBehavior>(
      WorkflowPauseScheduledCronRunQueueBehavior.QUEUE,
    );

  return (
    <Dialog open={isOpen}>
      <DialogContent className="w-fit min-w-[500px] max-w-[80%]">
        <DialogHeader>
          <DialogTitle>Pause workflow</DialogTitle>
        </DialogHeader>
        <div>
          <div className="mb-4 text-sm text-foreground">
            Pausing this workflow will prevent new runs from starting. Choose
            how cron and scheduled runs triggered while the workflow is paused
            should be handled.
          </div>
          <div className="mb-4 grid gap-2">
            <Label htmlFor="cronRunQueueBehavior">Cron run behavior</Label>
            <QueueBehaviorSelect
              id="cronRunQueueBehavior"
              value={cronRunQueueBehavior}
              onValueChange={setCronRunQueueBehavior}
            />
          </div>
          <div className="mb-4 grid gap-2">
            <Label htmlFor="scheduledRunQueueBehavior">
              Scheduled run behavior
            </Label>
            <QueueBehaviorSelect
              id="scheduledRunQueueBehavior"
              value={scheduledRunQueueBehavior}
              onValueChange={setScheduledRunQueueBehavior}
            />
          </div>
          <div className="flex flex-row justify-end gap-4">
            <Button variant="ghost" disabled={isLoading} onClick={onCancel}>
              Cancel
            </Button>
            <Button
              variant="default"
              disabled={isLoading}
              onClick={() =>
                onSubmit({ cronRunQueueBehavior, scheduledRunQueueBehavior })
              }
            >
              {isLoading && <Spinner />}
              Pause
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
