import { columns } from './components/webhook-columns';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import {
  useWebhooks,
  WebhookFormData,
  webhookFormSchema,
} from './hooks/use-webhooks';
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { useCallback, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  V1WebhookSourceName,
  V1WebhookAuthType,
  V1CreateWebhookRequest,
  V1WebhookHMACAlgorithm,
  V1WebhookHMACEncoding,
} from '@/lib/api';
import { Webhook, Copy, Check, AlertTriangle, Lightbulb } from 'lucide-react';
import { Spinner } from '@/components/v1/ui/loading';
import { SourceName } from './components/source-name';
import { AuthMethod } from './components/auth-method';
import { AuthSetup } from './components/auth-setup';
import { Link } from 'react-router-dom';
import { DocsButton } from '@/components/v1/docs/docs-button';
import { docsPages } from '@/lib/generated/docs';

export default function Webhooks() {
  const { data, isLoading, error } = useWebhooks();

  return (
    <div>
      <div className="flex flex-row justify-end w-full">
        <CreateWebhookModal />
      </div>
      <DataTable
        error={error}
        isLoading={isLoading}
        columns={columns()}
        data={data}
        emptyState={
          <div className="w-full h-full flex flex-col gap-y-4 text-foreground py-8 justify-center items-center">
            <p className="text-lg font-semibold">No webhooks found</p>
            <div className="w-fit">
              <DocsButton
                doc={docsPages.home['webhooks']}
                size="full"
                variant="outline"
                label="Learn about triggering runs from webhooks"
              />
            </div>
          </div>
        }
      />
    </div>
  );
}

const buildWebhookPayload = (data: WebhookFormData): V1CreateWebhookRequest => {
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
        <div className="flex flex-row items-center gap-x-2 ml-1">
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
        <Button variant="default">Create Webhook</Button>
      </DialogTrigger>
      <DialogContent className="max-w-[90%] md:max-w-[80%] lg:max-w-[60%] xl:max-w-[50%] max-h-[90dvh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex flex-col items-start gap-y-4">
            <div className="flex flex-row items-center gap-x-3">
              <div className="flex items-center justify-center w-8 h-8 bg-blue-100 rounded-full">
                <Webhook className="h-4 w-4 text-indigo-700" />
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
                <code className="max-w-full font-mono bg-muted px-2 py-1 rounded text-xs">
                  {createWebhookURL(webhookName)}
                </code>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={copyToClipboard}
                  className="h-6 w-6 p-0 flex-shrink-0"
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
                    <div className="h-10 flex flex-row items-center gap-x-2">
                      <SourceName sourceName={source} />
                      <span className="text-sm truncate max-w-full">
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
            <div className="text-xs text-muted-foreground pl-1">
              <p>
                CEL expression to extract the event key from the webhook
                payload. See{' '}
                <Link
                  to="https://cel.dev/"
                  className="text-blue-600"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  the docs
                </Link>{' '}
                for details.
              </p>
              <ul className="list-disc pl-4">
                <li>`input` refers to the payload</li>
                <li>`headers` refers to the headers</li>
              </ul>
            </div>
          </div>

          <div className="space-y-4">
            <div className="space-y-4 pl-4 border-l-2 border-gray-200">
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
