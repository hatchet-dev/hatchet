import { Button } from '@/next/components/ui/button';
import { Input } from '@/next/components/ui/input';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/next/components/ui/dialog';
import { ScheduledWorkflows, WorkflowRunStatus } from '@/next/lib/api';

interface EditScheduledRunDialogProps {
  editingRun: ScheduledWorkflows | null;
  onClose: () => void;
  onSave: (run: ScheduledWorkflows) => void;
}

export function EditScheduledRunDialog({
  editingRun,
  onClose,
  onSave,
}: EditScheduledRunDialogProps) {
  if (!editingRun) {
    return null;
  }

  return (
    <Dialog open={!!editingRun} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit Scheduled Run</DialogTitle>
          <DialogDescription>
            Modify your scheduled workflow execution
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          <div className="grid grid-cols-4 items-center gap-4">
            <label className="text-right text-sm" htmlFor="edit-trigger">
              Trigger At
            </label>
            <Input
              id="edit-trigger"
              type="datetime-local"
              value={editingRun.triggerAt}
              onChange={(e) =>
                onSave({
                  ...editingRun,
                  triggerAt: e.target.value,
                })
              }
              className="col-span-3"
            />
          </div>
          <div className="grid grid-cols-4 items-center gap-4">
            <label className="text-right text-sm" htmlFor="edit-status">
              Status
            </label>
            <div className="col-span-3">
              <select
                id="edit-status"
                value={editingRun.workflowRunStatus || 'PENDING'}
                onChange={(e) =>
                  onSave({
                    ...editingRun,
                    workflowRunStatus: e.target.value as WorkflowRunStatus,
                  })
                }
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              >
                <option value="PENDING">Pending</option>
                <option value="RUNNING">Running</option>
                <option value="SUCCEEDED">Succeeded</option>
                <option value="FAILED">Failed</option>
                <option value="CANCELLED">Cancelled</option>
                <option value="QUEUED">Queued</option>
                <option value="BACKOFF">Backoff</option>
              </select>
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={() => onSave(editingRun)}>Save Changes</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
