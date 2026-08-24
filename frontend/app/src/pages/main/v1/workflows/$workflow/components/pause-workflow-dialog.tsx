import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Input } from '@/components/v1/ui/input';
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

const DEFAULT_QUEUE_TTL = '24h';

// matches the regex we have in `convert_duration_to_interval` in pg
const QUEUE_TTL_REGEX = /^(([0-9]+(\.[0-9]*)?|\.[0-9]+)(ms|s|m|h|d))+$/;

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
    queueTtl: string;
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
  const [queueTtl, setQueueTtl] = useState(DEFAULT_QUEUE_TTL);

  const isTtlValid = QUEUE_TTL_REGEX.test(queueTtl);

  return (
    <Dialog open={isOpen}>
      <DialogContent className="w-full max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Pause workflow</DialogTitle>
        </DialogHeader>
        <div>
          <div className="mb-4 text-sm text-foreground">
            Are you sure you want to pause this workflow? New runs will not
            start until it is unpaused. Choose how cron and scheduled runs
            triggered while paused should be handled.
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
          <div className="mb-4 grid gap-2">
            <Label htmlFor="queueTtl">Queued run TTL</Label>
            <Input
              id="queueTtl"
              type="text"
              placeholder="e.g. 1d7h30m"
              value={queueTtl}
              onChange={(e) => setQueueTtl(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Queued runs older than this will be dropped instead of running
              once the workflow is unpaused. Duration string composed of
              ms/s/m/h/d units, e.g. "1d7h30m".
            </p>
          </div>
          <div className="flex flex-row justify-end gap-4">
            <Button variant="ghost" disabled={isLoading} onClick={onCancel}>
              Cancel
            </Button>
            <Button
              variant="default"
              disabled={isLoading || !isTtlValid}
              onClick={() =>
                onSubmit({
                  cronRunQueueBehavior,
                  scheduledRunQueueBehavior,
                  queueTtl,
                })
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
