import { Button } from '@/components/v1/ui/button';
import { Spinner } from '@/components/v1/ui/loading.tsx';
import {
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { SNSIntegration } from '@/lib/api';

interface DeleteSNSFormProps {
  className?: string;
  onSubmit: (snsIntegration: SNSIntegration) => void;
  onCancel: () => void;
  snsIntegration: SNSIntegration;
  isLoading: boolean;
}

export function DeleteSNSForm({ className, ...props }: DeleteSNSFormProps) {
  return (
    <DialogContent className="w-fit max-w-[80%] min-w-[500px]">
      <DialogHeader>
        <DialogTitle>Delete integration</DialogTitle>
      </DialogHeader>
      <div>
        <div className="text-sm text-foreground mb-4">
          Are you sure you want to revoke the SNS integration on Topic ARN{' '}
          {props.snsIntegration.topicArn}? This action will immediately prevent
          any SNS events from being sent to the Hatchet subscriber.
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
              props.onSubmit(props.snsIntegration);
            }}
          >
            {props.isLoading && <Spinner />}
            Delete integration
          </Button>
        </div>
      </div>
    </DialogContent>
  );
}
