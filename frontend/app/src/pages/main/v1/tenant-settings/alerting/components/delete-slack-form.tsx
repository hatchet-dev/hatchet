import { Button } from '@/components/v1/ui/button';
import {
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Spinner } from '@/components/v1/ui/loading.tsx';
import { SlackWebhook } from '@/lib/api';

interface DeleteSlackFormProps {
  className?: string;
  onSubmit: (slackWebhook: SlackWebhook) => void;
  onCancel: () => void;
  slackWebhook: SlackWebhook;
  isLoading: boolean;
}

export function DeleteSlackForm({ className, ...props }: DeleteSlackFormProps) {
  return (
    <DialogContent className="w-fit min-w-[500px] max-w-[80%]">
      <DialogHeader>
        <DialogTitle>Delete webhook</DialogTitle>
      </DialogHeader>
      <div>
        <div className="mb-4 text-sm text-foreground">
          Are you sure you want to delete the Slack webhook for channel{' '}
          {props.slackWebhook.channelName} in team {props.slackWebhook.teamName}
          ?
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
              props.onSubmit(props.slackWebhook);
            }}
          >
            {props.isLoading && <Spinner />}
            Delete webhook
          </Button>
        </div>
      </div>
    </DialogContent>
  );
}
