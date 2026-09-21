import { Label } from '@/components/v1/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';

type WorkflowStatusValue = 'active' | 'paused';

export function WorkflowStatusSettings({
  isPaused,
  disabled,
  isPending,
  onRequestPause,
  onRequestUnpause,
}: {
  isPaused: boolean;
  disabled: boolean;
  isPending: boolean;
  onRequestPause: () => void;
  onRequestUnpause: () => void;
}) {
  const value: WorkflowStatusValue = isPaused ? 'paused' : 'active';

  return (
    <div className="mb-8 space-y-3">
      <h3 className="border-b border-gray-200 pb-2 text-base font-semibold text-gray-900 dark:border-gray-700 dark:text-gray-100">
        Status
      </h3>
      <div className="pl-1">
        <div className="max-w-xl space-y-3">
          <div className="grid gap-2">
            <Label htmlFor="workflow-status">Workflow status</Label>
            <Select
              value={value}
              onValueChange={(next) => {
                if (next === value) {
                  return;
                }

                if (next === 'paused') {
                  onRequestPause();
                  return;
                }

                onRequestUnpause();
              }}
              disabled={disabled || isPending}
            >
              <SelectTrigger id="workflow-status" className="w-[180px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="active">Active</SelectItem>
                <SelectItem value="paused">Paused</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Pausing prevents new runs from starting. You will confirm how cron
            and scheduled runs should be handled before the workflow is paused.
          </p>
        </div>
      </div>
    </div>
  );
}
