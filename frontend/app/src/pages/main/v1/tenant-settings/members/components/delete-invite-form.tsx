import { Button } from '@/components/v1/ui/button';
import {
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Spinner } from '@/components/v1/ui/loading.tsx';
import { TenantInvite } from '@/lib/api';

interface DeleteInviteFormProps {
  className?: string;
  onSubmit: (invite: TenantInvite) => void;
  onCancel: () => void;
  invite: TenantInvite;
  isLoading: boolean;
}

export function DeleteInviteForm({
  className,
  ...props
}: DeleteInviteFormProps) {
  return (
    <DialogContent className="w-fit min-w-[500px] max-w-[80%]">
      <DialogHeader>
        <DialogTitle>Delete invite</DialogTitle>
      </DialogHeader>
      <div>
        <div className="mb-4 text-sm text-foreground">
          Are you sure you want to delete this invite?
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
              props.onSubmit(props.invite);
            }}
          >
            {props.isLoading && <Spinner />}
            Delete invite
          </Button>
        </div>
      </div>
    </DialogContent>
  );
}
