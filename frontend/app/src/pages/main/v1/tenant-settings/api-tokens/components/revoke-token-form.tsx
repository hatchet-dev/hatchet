import { Button } from '@/components/v1/ui/button';
import {
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Spinner } from '@/components/v1/ui/loading.tsx';
import { APIToken } from '@/lib/api';

interface RevokeTokenFormProps {
  className?: string;
  onSubmit: (apiToken: APIToken) => void;
  onCancel: () => void;
  apiToken: APIToken;
  isLoading: boolean;
}

export function RevokeTokenForm({ className, ...props }: RevokeTokenFormProps) {
  return (
    <DialogContent className="w-fit min-w-[500px] max-w-[80%]">
      <DialogHeader>
        <DialogTitle>Delete token</DialogTitle>
      </DialogHeader>
      <div>
        <div className="mb-4 text-sm text-foreground">
          Are you sure you want to revoke the API token {props.apiToken.name}?
          This action will immediately prevent any services running with this
          token from dispatching events or executing steps.
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
              props.onSubmit(props.apiToken);
            }}
          >
            {props.isLoading && <Spinner />}
            Revoke API token
          </Button>
        </div>
      </div>
    </DialogContent>
  );
}
