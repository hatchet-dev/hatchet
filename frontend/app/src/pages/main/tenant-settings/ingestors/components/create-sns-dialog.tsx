import {
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Button } from '@/components/v1/ui/button';
import { z } from 'zod';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Label } from '@/components/v1/ui/label';
import { Input } from '@/components/v1/ui/input';
import { cn } from '@/lib/utils';
import { Spinner } from '@/components/v1/ui/loading';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';

const schema = z.object({
  topicArn: z.string().min(1).max(255),
});

interface CreateSNSDialogProps {
  className?: string;
  snsIngestionUrl?: string;
  onSubmit: (opts: z.infer<typeof schema>) => void;
  isLoading: boolean;
  fieldErrors?: Record<string, string>;
}

export function CreateSNSDialog({
  className,
  snsIngestionUrl,
  ...props
}: CreateSNSDialogProps) {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
  });

  if (snsIngestionUrl) {
    return (
      <DialogContent className="w-fit max-w-[700px]">
        <DialogHeader>
          <DialogTitle>Use this ingestion URL</DialogTitle>
        </DialogHeader>
        <p className="text-sm">
          Use this ingestion URL with an HTTPS subscriber for SNS.
        </p>
        <CodeHighlighter
          language="typescript"
          className="text-sm"
          wrapLines={false}
          maxWidth={'calc(700px - 4rem)'}
          code={snsIngestionUrl}
          copy
        />
      </DialogContent>
    );
  }

  const topicArnError =
    errors.topicArn?.message?.toString() || props.fieldErrors?.name;

  return (
    <DialogContent className="w-fit max-w-[80%] min-w-[500px]">
      <DialogHeader>
        <DialogTitle>Create a new SNS integration</DialogTitle>
      </DialogHeader>
      <div className={cn('grid gap-6', className)}>
        <form
          onSubmit={handleSubmit((d) => {
            props.onSubmit(d);
          })}
        >
          <div className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="email">Topic ARN</Label>
              <Input
                {...register('topicArn')}
                id="sns-topic-arn"
                placeholder="arn:aws:sns:us-west-1:123456789:topic"
                autoCapitalize="none"
                autoCorrect="off"
                disabled={props.isLoading}
              />
              {topicArnError && (
                <div className="text-sm text-red-500">{topicArnError}</div>
              )}
            </div>
            <Button disabled={props.isLoading}>
              {props.isLoading && <Spinner />}
              Create integration
            </Button>
          </div>
        </form>
      </div>
    </DialogContent>
  );
}
