import { Button } from '@/components/v1/ui/button';
import {
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Spinner } from '@/components/v1/ui/loading.tsx';
import { TenantAlertEmailGroup } from '@/lib/api';

interface DeleteEmailGroupFormProps {
  className?: string;
  onSubmit: (emailGroup: TenantAlertEmailGroup) => void;
  onCancel: () => void;
  emailGroup: TenantAlertEmailGroup;
  isLoading: boolean;
}

export function DeleteEmailGroupForm({
  className,
  ...props
}: DeleteEmailGroupFormProps) {
  return (
    <DialogContent className="w-fit min-w-[500px] max-w-[80%]">
      <DialogHeader>
        <DialogTitle>Delete email group</DialogTitle>
      </DialogHeader>
      <div>
        <div className="mb-4 text-sm text-foreground">
          Are you sure you want to delete this email group?
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
              props.onSubmit(props.emailGroup);
            }}
          >
            {props.isLoading && <Spinner />}
            Delete group
          </Button>
        </div>
      </div>
    </DialogContent>
  );
}
