import { AuthMethod } from './components/auth-method';
import { AuthSetup } from './components/auth-setup';
import { SourceName } from './components/source-name';
import { WebhookActionsCell } from './components/webhook-columns';
import {
  useWebhooks,
  WebhookFormData,
  webhookFormSchema,
  WebhookUpdateFormData,
  webhookUpdateFormSchema,
} from './hooks/use-webhooks';
import { DocsButton } from '@/components/v1/docs/docs-button';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/v1/ui/dialog';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { Spinner } from '@/components/v1/ui/loading';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import {
  V1CreateWebhookRequest,
  V1Webhook,
  V1WebhookAuthType,
  V1WebhookHMACAlgorithm,
  V1WebhookHMACEncoding,
  V1WebhookSourceName,
} from '@/lib/api';
import { docsPages } from '@/lib/generated/docs';
import { zodResolver } from '@hookform/resolvers/zod';
import { AlertTriangle, Check, Copy, Lightbulb, Webhook } from 'lucide-react';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useForm } from 'react-hook-form';

export default function Webhooks() {
  const { data, isLoading } = useWebhooks();

  const webhookColumns = useMemo(
    () => [
      {
        columnLabel: 'Name',
        cellRenderer: (webhook: V1Webhook) => (
          <div className="w-full">{webhook.name}</div>
        ),
      },
      {
        columnLabel: 'Source',
        cellRenderer: (webhook: V1Webhook) => (
          <div className="w-full">
            <SourceName sourceName={webhook.sourceName} />
          </div>
        ),
      },
      {
        columnLabel: 'Event Key',
        cellRenderer: (webhook: V1Webhook) => {
          const text = webhook.eventKeyExpression || '';
          const truncated = text.length > 25 ? `${text.slice(0, 25)}...` : text;
          return (
            <code className="rounded bg-muted px-2 py-1 text-xs" title={text}>
              {truncated}
            </code>
          );
        },
      },
      {
        columnLabel: 'Scope',
        cellRenderer: (webhook: V1Webhook) => {
          if (!webhook.scopeExpression) {
            return <span className="text-xs text-muted-foreground">—</span>;
          }
          const text = webhook.scopeExpression;
          const truncated = text.length > 25 ? `${text.slice(0, 25)}...` : text;
          return (
            <code className="rounded bg-muted px-2 py-1 text-xs" title={text}>
              {truncated}
            </code>
          );
        },
      },
      {
        columnLabel: 'Auth Method',
        cellRenderer: (webhook: V1Webhook) => (
          <div className="w-full">
            <AuthMethod authMethod={webhook.authType} />
          </div>
        ),
      },
      {
        columnLabel: 'Actions',
        cellRenderer: (webhook: V1Webhook) => (
          <WebhookActionsCell webhook={webhook} />
        ),
      },
    ],
    [],
  );

  if (isLoading) {
    return <div>Loading...</div>;
  }

  return (
    <div>
      <div className="mb-4 flex w-full flex-row justify-end">
        <CreateWebhookModal />
      </div>
      {data && data.length > 0 ? (
        <SimpleTable columns={webhookColumns} data={data} />
      ) : (
        <div className="flex h-full w-full flex-col items-center justify-center gap-y-4 py-8 text-foreground">
          <p className="text-lg font-semibold">No webhooks found</p>
          <div className="w-fit">
            <DocsButton
              doc={docsPages.home.webhooks}
              label="Learn about triggering runs from webhooks"
            />
          </div>
        </div>
      )}
    </div>
  );
}

const parseStaticPayload = (
  staticPayload: string | undefined,
): object | undefined => {
  if (!staticPayload || staticPayload.trim() === '') {
    return undefined;
  }
  try {
    return JSON.parse(staticPayload);
  } catch {
    throw new Error('Static payload must be valid JSON');
  }
};

const buildWebhookPayload = (data: WebhookFormData): V1CreateWebhookRequest => {
  const basePayload = {
    scopeExpression: data.scopeExpression || undefined,
    staticPayload: parseStaticPayload(data.staticPayload),
  };

  switch (data.sourceName) {
    case V1WebhookSourceName.GENERIC:
      switch (data.authType) {
        case V1WebhookAuthType.BASIC:
          if (!data.username || !data.password) {
            throw new Error(
              'Username and password are required for basic auth',
            );
          }

          return {
            ...basePayload,
            sourceName: data.sourceName,
            name: data.name,
            eventKeyExpression: data.eventKeyExpression,
            authType: data.authType,
            auth: {
              username: data.username,
              password: data.password,
            },
          };
        case V1WebhookAuthType.API_KEY:
          if (!data.headerName || !data.apiKey) {
            throw new Error(
              'Header name and API key are required for API key auth',
            );
          }

          return {
            ...basePayload,
            sourceName: data.sourceName,
            name: data.name,
            eventKeyExpression: data.eventKeyExpression,
            authType: data.authType,
            auth: {
              headerName: data.headerName,
              apiKey: data.apiKey,
            },
          };
        case V1WebhookAuthType.HMAC:
          if (
            !data.algorithm ||
            !data.encoding ||
            !data.signatureHeaderName ||
            !data.signingSecret
          ) {
            throw new Error(
              'Algorithm, encoding, signature header name, and signing secret are required for HMAC auth',
            );
          }

          return {
            ...basePayload,
            sourceName: data.sourceName,
            name: data.name,
            eventKeyExpression: data.eventKeyExpression,
            authType: data.authType,
            auth: {
              algorithm: data.algorithm,
              encoding: data.encoding,
              signatureHeaderName: data.signatureHeaderName,
              signingSecret: data.signingSecret,
            },
          };
        default:
          const exhaustiveCheck: never = data.authType;
          throw new Error(`Unhandled auth type: ${exhaustiveCheck}`);
      }
    case V1WebhookSourceName.GITHUB:
      if (!data.signingSecret) {
        throw new Error('Signing secret is required for GitHub webhooks');
      }

      return {
        ...basePayload,
        sourceName: data.sourceName,
        name: data.name,
        eventKeyExpression: data.eventKeyExpression,
        authType: V1WebhookAuthType.HMAC,
        auth: {
          // Header name is 'X-Hub-Signature-256'
          // Encoding algorithm is SHA256
          // Encoding type is HEX
          // See GitHub docs: https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries#validating-webhook-deliveries
          algorithm: V1WebhookHMACAlgorithm.SHA256,
          encoding: V1WebhookHMACEncoding.HEX,
          signatureHeaderName: 'X-Hub-Signature-256',
          signingSecret: data.signingSecret,
        },
      };
    case V1WebhookSourceName.LINEAR:
      if (!data.signingSecret) {
        throw new Error('Signing secret is required for Linear webhooks');
      }

      return {
        ...basePayload,
        sourceName: data.sourceName,
        name: data.name,
        eventKeyExpression: data.eventKeyExpression,
        authType: V1WebhookAuthType.HMAC,
        auth: {
          // Header name is 'linear-signature'
          // Encoding algorithm is SHA256
          // Encoding type is HEX
          // See Linear docs: https://linear.app/developers/webhooks#creating-a-simple-webhook-consumer
          algorithm: V1WebhookHMACAlgorithm.SHA256,
          encoding: V1WebhookHMACEncoding.HEX,
          signatureHeaderName: 'linear-signature',
          signingSecret: data.signingSecret,
        },
      };
    case V1WebhookSourceName.STRIPE:
      if (!data.signingSecret) {
        throw new Error('Signing secret is required for Stripe webhooks');
      }

      return {
        ...basePayload,
        sourceName: data.sourceName,
        name: data.name,
        eventKeyExpression: data.eventKeyExpression,
        authType: V1WebhookAuthType.HMAC,
        auth: {
          // Header name is 'Stripe-Signature'
          // Encoding algorithm is SHA256
          // Encoding type is HEX
          // See Stripe docs: https://docs.stripe.com/webhooks?verify=verify-manually#verify-manually
          algorithm: V1WebhookHMACAlgorithm.SHA256,
          encoding: V1WebhookHMACEncoding.HEX,
          signatureHeaderName: 'Stripe-Signature',
          signingSecret: data.signingSecret,
        },
      };
    case V1WebhookSourceName.SLACK:
      if (!data.signingSecret) {
        throw new Error('signing secret is required for Slack webhooks');
      }

      return {
        ...basePayload,
        sourceName: data.sourceName,
        name: data.name,
        eventKeyExpression: data.eventKeyExpression,
        authType: V1WebhookAuthType.HMAC,
        auth: {
          // Slack sends the expected signature and timestamp as headers
          // https://api.slack.com/apis/events-api#receiving-events
          algorithm: V1WebhookHMACAlgorithm.SHA256,
          encoding: V1WebhookHMACEncoding.HEX,
          signatureHeaderName: 'X-Slack-Signature',
          signingSecret: data.signingSecret,
        },
      };
    case V1WebhookSourceName.SVIX:
      if (!data.signingSecret) {
        throw new Error('signing secret is required for Svix webhooks');
      }

      return {
        ...basePayload,
        sourceName: data.sourceName,
        name: data.name,
        eventKeyExpression: data.eventKeyExpression,
        authType: V1WebhookAuthType.HMAC,
        auth: {
          // Svix uses its own SDK for verification; these HMAC fields are
          // stored but the server-side validation implements Svix's signature
          // verification protocol.
          // See: https://docs.svix.com/receiving/verifying-payloads/how-to-verify-a-payload
          algorithm: V1WebhookHMACAlgorithm.SHA256,
          encoding: V1WebhookHMACEncoding.BASE64,
          signatureHeaderName: 'svix-signature',
          signingSecret: data.signingSecret,
        },
      };
    default:
      const exhaustiveCheck: never = data.sourceName;
      throw new Error(`Unhandled source name: ${exhaustiveCheck}`);
  }
};

const createSourceInlineDescription = (sourceName: V1WebhookSourceName) => {
  switch (sourceName) {
    case V1WebhookSourceName.GENERIC:
      return '(receive incoming webhook requests from any service)';
    case V1WebhookSourceName.GITHUB:
    case V1WebhookSourceName.LINEAR:
    case V1WebhookSourceName.STRIPE:
    case V1WebhookSourceName.SLACK:
    case V1WebhookSourceName.SVIX:
      return '';
    default:
      const exhaustiveCheck: never = sourceName;
      throw new Error(`Unhandled source name: ${exhaustiveCheck}`);
  }
};

const SourceCaption = ({ sourceName }: { sourceName: V1WebhookSourceName }) => {
  switch (sourceName) {
    case V1WebhookSourceName.GITHUB:
      return (
        <div className="ml-1 flex flex-row items-center gap-x-2">
          <AlertTriangle className="size-4 text-yellow-500" />
          <p className="text-xs text-muted-foreground">
            Select <span className="font-semibold">application/json</span> as
            the content type in your GitHub webhook settings.
          </p>
        </div>
      );
    case V1WebhookSourceName.GENERIC:
    case V1WebhookSourceName.LINEAR:
    case V1WebhookSourceName.STRIPE:
    case V1WebhookSourceName.SLACK:
    case V1WebhookSourceName.SVIX:
      return '';
    default:
      const exhaustiveCheck: never = sourceName;
      throw new Error(`Unhandled source name: ${exhaustiveCheck}`);
  }
};

const CreateWebhookModal = () => {
  const { mutations, createWebhookURL } = useWebhooks();
  const { createWebhook, isCreatePending } = mutations;
  const [open, setOpen] = useState(false);
  const [copied, setCopied] = useState(false);

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    reset,
    formState: { errors },
  } = useForm<WebhookFormData>({
    resolver: zodResolver(webhookFormSchema),
    defaultValues: {
      sourceName: V1WebhookSourceName.GENERIC,
      authType: V1WebhookAuthType.BASIC,
      name: '',
      eventKeyExpression: 'input.id',
      username: '',
      password: '',
    },
  });

  const sourceName = watch('sourceName');
  const authType = watch('authType');
  const webhookName = watch('name');
  const eventKeyExpression = watch('eventKeyExpression');

  /* Update default event key expression when source changes */
  useEffect(() => {
    if (sourceName === V1WebhookSourceName.SLACK && !eventKeyExpression) {
      setValue('eventKeyExpression', 'input.type');
    } else if (
      sourceName === V1WebhookSourceName.GENERIC &&
      !eventKeyExpression
    ) {
      setValue('eventKeyExpression', 'input.id');
    }
  }, [sourceName, eventKeyExpression, setValue]);

  const copyToClipboard = useCallback(async () => {
    if (webhookName) {
      try {
        await navigator.clipboard.writeText(createWebhookURL(webhookName));
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      } catch (err) {
        console.error('Failed to copy URL:', err);
      }
    }
  }, [webhookName, createWebhookURL]);

  const onSubmit = useCallback(
    (data: WebhookFormData) => {
      const payload = buildWebhookPayload(data);

      createWebhook(payload, {
        onSuccess: () => {
          setOpen(false);
          reset();
        },
      });
    },
    [createWebhook, reset],
  );

  return (
    <Dialog
      open={open}
      onOpenChange={(isOpen) => {
        setOpen(isOpen);

        if (!isOpen) {
          reset();
          setCopied(false);
        }
      }}
    >
      <DialogTrigger asChild>
        <Button variant="cta">Create Webhook</Button>
      </DialogTrigger>
      <DialogContent className="max-h-[90dvh] max-w-[90%] overflow-y-auto md:max-w-[80%] lg:max-w-[60%] xl:max-w-[50%]">
        <DialogHeader>
          <DialogTitle className="flex flex-col items-start gap-y-4">
            <div className="flex flex-row items-center gap-x-3">
              <div className="flex h-8 w-8 items-center justify-center rounded-full bg-blue-100">
                <Webhook className="size-4 text-indigo-700" />
              </div>
              Create a webhook
            </div>
            <span className="text-sm text-muted-foreground">
              Webhooks are a beta feature
            </span>
          </DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          <div className="space-y-2">
            <Label htmlFor="name" className="text-sm font-medium">
              Webhook Name <span className="text-red-500">*</span>
            </Label>
            <Input
              data-1p-ignore
              id="name"
              placeholder="test-webhook"
              {...register('name')}
              className="h-10"
            />
            {errors.name && (
              <p className="text-xs text-red-500">{errors.name.message}</p>
            )}
            <div className="flex flex-col items-start gap-2 text-xs text-muted-foreground">
              <span className="">Send incoming webhook requests to:</span>
              <div className="flex flex-row items-center gap-2">
                <code className="max-w-full rounded bg-muted px-2 py-1 font-mono text-xs">
                  {createWebhookURL(webhookName)}
                </code>
                <Button
                  type="button"
                  variant="icon"
                  size="xs"
                  onClick={copyToClipboard}
                  className="flex-shrink-0"
                  disabled={!webhookName}
                >
                  {copied ? (
                    <Check className="size-4 text-green-600" />
                  ) : (
                    <Copy className="size-4" />
                  )}
                </Button>
              </div>
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="sourceName" className="text-sm font-medium">
              Source <span className="text-red-500">*</span>
            </Label>
            <Select
              value={sourceName}
              onValueChange={(value: V1WebhookSourceName) =>
                setValue('sourceName', value)
              }
            >
              <SelectTrigger>
                <SelectValue>
                  <SourceName sourceName={sourceName} />
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                {Object.values(V1WebhookSourceName).map((source) => (
                  <SelectItem key={source} value={source} className="h-10">
                    <div className="flex h-10 flex-row items-center gap-x-2">
                      <SourceName sourceName={source} />
                      <span className="max-w-full truncate text-sm">
                        {createSourceInlineDescription(source)}
                      </span>
                    </div>
                  </SelectItem>
                ))}
                <SelectItem
                  disabled
                  key="empty"
                  value="reach-out"
                  className="text-sm data-[disabled]:text-white data-[disabled]:opacity-100"
                >
                  <div className="flex flex-row items-center gap-x-2">
                    <Lightbulb className="size-4 text-yellow-500" />
                    <span>Want a new source added? Reach out to support</span>
                  </div>
                </SelectItem>
              </SelectContent>
            </Select>
            <SourceCaption sourceName={sourceName} />
          </div>

          <div className="space-y-2">
            <Label htmlFor="eventKeyExpression" className="text-sm font-medium">
              Event Key Expression <span className="text-red-500">*</span>
            </Label>
            <Input
              id="eventKeyExpression"
              placeholder="input.id"
              {...register('eventKeyExpression')}
              className="h-10"
            />
            {errors.eventKeyExpression && (
              <p className="text-xs text-red-500">
                {errors.eventKeyExpression.message}
              </p>
            )}
            <div className="pl-1 text-xs text-muted-foreground">
              <p>
                CEL expression to extract the event key from the webhook
                payload. See{' '}
                <a
                  href="https://cel.dev/"
                  className="text-blue-600"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  the docs
                </a>{' '}
                for details.
              </p>
              <ul className="list-disc pl-4">
                <li>`input` refers to the payload</li>
                <li>`headers` refers to the headers</li>
              </ul>
              {sourceName === V1WebhookSourceName.SLACK && (
                <div className="mt-2 rounded-md border border-border bg-muted p-3">
                  <p className="text-xs text-muted-foreground">
                    For Slack webhooks, the event key expression{' '}
                    <code className="rounded bg-background px-1.5 py-0.5 text-foreground">
                      input.type
                    </code>{' '}
                    works well since Slack interactive payloads don't have a
                    top-level `id` field.
                  </p>
                </div>
              )}
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="scopeExpression" className="text-sm font-medium">
              Scope Expression{' '}
              <span className="text-muted-foreground">(optional)</span>
            </Label>
            <Input
              id="scopeExpression"
              placeholder="input.organization_id"
              {...register('scopeExpression')}
              className="h-10"
            />
            {errors.scopeExpression && (
              <p className="text-xs text-red-500">
                {errors.scopeExpression.message}
              </p>
            )}
            <div className="pl-1 text-xs text-muted-foreground">
              <p>
                CEL expression to extract the scope from the webhook payload.
                Used to filter which workflows to trigger based on event
                filters.
              </p>
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="staticPayload" className="text-sm font-medium">
              Static Payload{' '}
              <span className="text-muted-foreground">(optional)</span>
            </Label>
            <textarea
              id="staticPayload"
              placeholder='{"source": "webhook", "version": "v1"}'
              {...register('staticPayload')}
              className="h-24 w-full rounded-md border border-input bg-background px-3 py-2 font-mono text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
            />
            {errors.staticPayload && (
              <p className="text-xs text-red-500">
                {errors.staticPayload.message}
              </p>
            )}
            <div className="pl-1 text-xs text-muted-foreground">
              <p>
                JSON object to merge into the webhook payload. Static payload
                fields take precedence over incoming payload fields.
              </p>
            </div>
          </div>

          <div className="space-y-4">
            <div className="space-y-4 border-l-2 border-gray-200 pl-4">
              {sourceName === V1WebhookSourceName.GENERIC && (
                <div className="space-y-2">
                  <Label htmlFor="authType" className="text-sm font-medium">
                    Authentication Type
                  </Label>
                  <Select
                    value={authType}
                    onValueChange={(value: V1WebhookAuthType) =>
                      setValue('authType', value)
                    }
                  >
                    <SelectTrigger className="h-10">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value={V1WebhookAuthType.BASIC}>
                        <AuthMethod authMethod={V1WebhookAuthType.BASIC} />
                      </SelectItem>
                      <SelectItem value={V1WebhookAuthType.API_KEY}>
                        <AuthMethod authMethod={V1WebhookAuthType.API_KEY} />
                      </SelectItem>
                      <SelectItem value={V1WebhookAuthType.HMAC}>
                        <AuthMethod authMethod={V1WebhookAuthType.HMAC} />
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              )}

              <AuthSetup
                authMethod={authType}
                sourceName={sourceName}
                register={register}
                watch={watch}
                setValue={setValue}
              />
            </div>
          </div>

          <div className="flex justify-end gap-3 pt-4">
            <DialogClose asChild>
              <Button type="button" variant="outline">
                Cancel
              </Button>
            </DialogClose>
            <Button type="submit" disabled={isCreatePending}>
              {isCreatePending && <Spinner />}
              Create Webhook
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
};

export const EditWebhookModal = ({
  webhook,
  open,
  onOpenChange,
}: {
  webhook: V1Webhook;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) => {
  const { mutations } = useWebhooks();

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<WebhookUpdateFormData>({
    resolver: zodResolver(webhookUpdateFormSchema),
    defaultValues: {
      eventKeyExpression: webhook.eventKeyExpression || '',
      scopeExpression: webhook.scopeExpression || '',
      staticPayload: webhook.staticPayload
        ? JSON.stringify(webhook.staticPayload, null, 2)
        : '',
    },
  });

  useEffect(() => {
    if (open) {
      reset({
        eventKeyExpression: webhook.eventKeyExpression || '',
        scopeExpression: webhook.scopeExpression || '',
        staticPayload: webhook.staticPayload
          ? JSON.stringify(webhook.staticPayload, null, 2)
          : '',
      });
    }
  }, [webhook, open, reset]);

  const onSubmit = useCallback(
    (data: WebhookUpdateFormData) => {
      const staticPayload = parseStaticPayload(data.staticPayload);

      mutations.updateWebhook(
        {
          webhookName: webhook.name,
          webhookData: {
            eventKeyExpression: data.eventKeyExpression,
            scopeExpression: data.scopeExpression ?? '',
            staticPayload: staticPayload ?? {},
          },
        },
        {
          onSuccess: () => {
            onOpenChange(false);
          },
        },
      );
    },
    [mutations, webhook.name, onOpenChange],
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Edit Webhook: {webhook.name}</DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <Label
              htmlFor="edit-eventKeyExpression"
              className="text-sm font-medium"
            >
              Event Key Expression <span className="text-red-500">*</span>
            </Label>
            <Input
              id="edit-eventKeyExpression"
              placeholder="input.id"
              {...register('eventKeyExpression')}
              className="h-10"
            />
            {errors.eventKeyExpression && (
              <p className="text-xs text-red-500">
                {errors.eventKeyExpression.message}
              </p>
            )}
            <p className="text-xs text-muted-foreground">
              CEL expression to extract the event key from the webhook payload.
            </p>
          </div>

          <div className="space-y-2">
            <Label
              htmlFor="edit-scopeExpression"
              className="text-sm font-medium"
            >
              Scope Expression{' '}
              <span className="text-muted-foreground">(optional)</span>
            </Label>
            <Input
              id="edit-scopeExpression"
              placeholder="input.organization_id"
              {...register('scopeExpression')}
              className="h-10"
            />
            {errors.scopeExpression && (
              <p className="text-xs text-red-500">
                {errors.scopeExpression.message}
              </p>
            )}
            <p className="text-xs text-muted-foreground">
              CEL expression to extract the scope for event filtering.
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="edit-staticPayload" className="text-sm font-medium">
              Static Payload{' '}
              <span className="text-muted-foreground">(optional)</span>
            </Label>
            <textarea
              id="edit-staticPayload"
              placeholder='{"source": "webhook", "version": "v1"}'
              {...register('staticPayload')}
              className="h-32 w-full rounded-md border border-input bg-background px-3 py-2 font-mono text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
            />
            {errors.staticPayload && (
              <p className="text-xs text-red-500">
                {errors.staticPayload.message}
              </p>
            )}
            <p className="text-xs text-muted-foreground">
              JSON object to merge into the webhook payload.
            </p>
          </div>

          <div className="flex justify-end gap-3 pt-4">
            <DialogClose asChild>
              <Button type="button" variant="outline">
                Cancel
              </Button>
            </DialogClose>
            <Button type="submit" disabled={mutations.isUpdatePending}>
              {mutations.isUpdatePending && <Spinner />}
              Save Changes
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
};
