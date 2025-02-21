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
import { SecretCopier } from '@/components/v1/ui/secret-copier';
import { useEffect, useState } from 'react';
import { RecentWebhookRequests } from './recent-webhook-requests';

const schema = z.object({
  name: z.string().max(255).optional(),
  url: z.string().url().min(1).max(255),
  secret: z.string().min(1).max(255).optional(),
});

interface CreateWebhookWorkerDialogProps {
  className?: string;
  secret?: string;
  webhookId?: string;
  onSubmit: (opts: z.infer<typeof schema> & { name: string }) => void;
  isLoading: boolean;
  fieldErrors?: Record<string, string>;
  isOpen: boolean;
}

export function CreateWebhookWorkerDialog({
  className,
  secret,
  webhookId,
  isOpen,
  ...props
}: CreateWebhookWorkerDialogProps) {
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {},
  });

  const [canTestConnection, setCanTestConnection] = useState(false);
  const [waitingForConnection, setWaitingForConnection] = useState(false);
  const [isComplete, setIsComplete] = useState(false);

  const nameError = errors.name?.message?.toString() || props.fieldErrors?.name;
  const urlError = errors.url?.message?.toString() || props.fieldErrors?.url;

  useEffect(() => {
    if (!isOpen) {
      setCanTestConnection(false);
      setWaitingForConnection(false);
      setIsComplete(false);
      reset(); // Reset form fields
    }
  }, [isOpen, reset]);

  if (isComplete) {
    return (
      <DialogContent className="w-fit max-w-[700px]">
        <DialogHeader>
          <DialogTitle>Connected!</DialogTitle>
        </DialogHeader>
        <p className="text-sm">
          Your webhook worker is now connected and ready to receive runs.
        </p>
      </DialogContent>
    );
  }

  if (secret) {
    return (
      <DialogContent className="w-fit max-w-[700px]">
        <DialogHeader>
          <DialogTitle>Keep it secret, keep it safe</DialogTitle>
        </DialogHeader>
        <p className="text-sm">
          Set the following Hatchet configuration in your application
          environment:
        </p>
        <SecretCopier
          className="text-sm"
          maxWidth={'calc(700px - 4rem)'}
          secrets={{
            HATCHET_WEBHOOK_SECRET: secret,
          }}
          copy
          onClick={() => setCanTestConnection(true)}
        />

        <p className="text-sm text-gray-500">
          These values should be kept secret and not shared with anyone. They
          will only be displayed once.
        </p>

        <Button
          onClick={() => {
            setWaitingForConnection(true);
          }}
          disabled={!canTestConnection}
        >
          {waitingForConnection && <Spinner />}
          Test Connection
        </Button>
        {waitingForConnection && webhookId && (
          <RecentWebhookRequests
            webhookId={webhookId}
            filterBeforeNow={true}
            onConnected={() => setIsComplete(true)}
          />
        )}
      </DialogContent>
    );
  }

  return (
    <DialogContent className="w-fit max-w-[80%] min-w-[500px]">
      <DialogHeader>
        <DialogTitle>Create a New Webhook Worker</DialogTitle>
      </DialogHeader>
      <div className={cn('grid gap-6', className)}>
        <form
          onSubmit={handleSubmit((d) => {
            const name = d.name || d.url || '';
            props.onSubmit({ ...d, name });
          })}
        >
          <div className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="url">URL</Label>
              <p className="text-sm dark:text-gray-400 text-gray-800">
                The URL with full path where the webhook worker will be
                available.
              </p>
              <Input
                {...register('url')}
                id="webhook-worker-url"
                name="url"
                autoCapitalize="none"
                autoCorrect="off"
                disabled={props.isLoading}
              />
              {urlError && (
                <div className="text-sm text-red-500">{urlError}</div>
              )}
            </div>
            <div className="grid gap-2">
              <Label htmlFor="name">Friendly Name (optional)</Label>
              <p className="text-sm dark:text-gray-400 text-gray-800">
                An easy to remember name to identify worker.
              </p>
              <Input
                {...register('name')}
                id="webhook-worker-name"
                name="name"
                autoCapitalize="none"
                autoCorrect="off"
                disabled={props.isLoading}
              />
              {nameError && (
                <div className="text-sm text-red-500">{nameError}</div>
              )}
            </div>

            <Button disabled={props.isLoading}>
              {props.isLoading && <Spinner />}
              Continue
            </Button>
          </div>
        </form>
      </div>
    </DialogContent>
  );
}
