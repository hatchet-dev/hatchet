import { Button } from '@/components/ui/button';
import { Spinner } from '@/components/ui/loading.tsx';
import {
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Workflow } from '@/lib/api';

interface DeleteWorkflowFormProps {
  className?: string;
  onSubmit: (workflow: Workflow) => void;
  onCancel: () => void;
  workflow: Workflow;
  isLoading: boolean;
}

export function DeleteWorkflowForm({
  className,
  ...props
}: DeleteWorkflowFormProps) {
  return (
    <DialogContent className="w-fit max-w-[80%] min-w-[500px]">
      <DialogHeader>
        <DialogTitle>Delete workflow</DialogTitle>
      </DialogHeader>
      <div>
        <div className="text-sm text-foreground mb-4">
          Are you sure you want to delete the workflow {props.workflow.name}?
          This action cannot be undone, and will immediately prevent any
          services running with this workflow from executing steps.
        </div>
        <div className="flex flex-row gap-4">
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
              props.onSubmit(props.workflow);
            }}
          >
            {props.isLoading && <Spinner />}
            Delete workflow
          </Button>
        </div>
      </div>
    </DialogContent>
  );
}
