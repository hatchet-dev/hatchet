import { ReviewedButtonTemp } from '@/components/v1/ui/button';
import { Spinner } from '@/components/v1/ui/loading.tsx';
import {
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
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
    <DialogContent className="w-fit max-w-[80%] min-w-[500px]">
      <DialogHeader>
        <DialogTitle>Delete token</DialogTitle>
      </DialogHeader>
      <div>
        <div className="text-sm text-foreground mb-4">
          Are you sure you want to revoke the API token {props.apiToken.name}?
          This action will immediately prevent any services running with this
          token from dispatching events or executing steps.
        </div>
        <div className="flex flex-row gap-4">
          <ReviewedButtonTemp
            variant="ghost"
            onClick={() => {
              props.onCancel();
            }}
          >
            Cancel
          </ReviewedButtonTemp>
          <ReviewedButtonTemp
            variant="destructive"
            onClick={() => {
              props.onSubmit(props.apiToken);
            }}
          >
            {props.isLoading && <Spinner />}
            Revoke API token
          </ReviewedButtonTemp>
        </div>
      </div>
    </DialogContent>
  );
}
