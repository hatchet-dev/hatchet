import {
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { z } from 'zod';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';
import { Spinner } from '@/components/ui/loading';
import { CodeHighlighter } from '@/components/ui/code-highlighter';

const schema = z.object({
  name: z.string().min(1).max(255),
  url: z.string().url().min(1).max(255),
  secret: z.string().min(1).max(255).optional(),
});

interface CreateWebhookWorkerDialogProps {
  className?: string;
  secret?: string;
  onSubmit: (opts: z.infer<typeof schema>) => void;
  isLoading: boolean;
  fieldErrors?: Record<string, string>;
}

export function CreateWebhookWorkerDialog({
  className,
  secret,
  ...props
}: CreateWebhookWorkerDialogProps) {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {},
  });

  const nameError = errors.name?.message?.toString() || props.fieldErrors?.name;
  const urlError = errors.url?.message?.toString() || props.fieldErrors?.url;

  if (secret) {
    return (
      <DialogContent className="w-fit max-w-[700px]">
        <DialogHeader>
          <DialogTitle>Keep it secret, keep it safe</DialogTitle>
        </DialogHeader>
        <p className="text-sm">
          Copy the webhook secret and add it in your application.
        </p>
        <CodeHighlighter
          language="typescript"
          className="text-sm"
          wrapLines={false}
          maxWidth={'calc(700px - 4rem)'}
          code={secret}
          copy
        />
      </DialogContent>
    );
  }

  return (
    <DialogContent className="w-fit max-w-[80%] min-w-[500px]">
      <DialogHeader>
        <DialogTitle>Create a new Webhook Endpoint</DialogTitle>
      </DialogHeader>
      <div className={cn('grid gap-6', className)}>
        <form
          onSubmit={handleSubmit((d) => {
            props.onSubmit(d);
          })}
        >
          <div className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="name">Name</Label>
              <Input
                {...register('name')}
                id="webhook-worker-name"
                name="name"
                placeholder="My Webhook Endpoint"
                autoCapitalize="none"
                autoCorrect="off"
                disabled={props.isLoading}
              />
              {nameError && (
                <div className="text-sm text-red-500">{nameError}</div>
              )}
            </div>
            <div className="grid gap-2">
              <Label htmlFor="url">URL</Label>
              <Input
                {...register('url')}
                id="webhook-worker-url"
                name="url"
                placeholder="The Webhook URL"
                autoCapitalize="none"
                autoCorrect="off"
                disabled={props.isLoading}
              />
              {urlError && (
                <div className="text-sm text-red-500">{urlError}</div>
              )}
            </div>

            <Button disabled={props.isLoading}>
              {props.isLoading && <Spinner />}
              Create
            </Button>
          </div>
        </form>
      </div>
    </DialogContent>
  );
}
