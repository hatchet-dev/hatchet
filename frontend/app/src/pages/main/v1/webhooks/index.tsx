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
import { Webhook, Copy, Check } from 'lucide-react';
import { Spinner } from '@/components/v1/ui/loading';
import { SourceName } from './components/source-name';
import { AuthMethod } from './components/auth-method';
import { AuthSetup } from './components/auth-setup';

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
        filters={[]}
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
          // eslint-disable-next-line no-case-declarations
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
          algorithm: V1WebhookHMACAlgorithm.SHA256,
          encoding: V1WebhookHMACEncoding.HEX,
          signatureHeaderName: 'X-Hub-Signature-256',
          signingSecret: data.signingSecret,
        },
      };
    case V1WebhookSourceName.STRIPE:
      if (!data.signingSecret) {
        throw new Error('Signing secret is required for GitHub webhooks');
      }

      return {
        sourceName: data.sourceName,
        name: data.name,
        eventKeyExpression: data.eventKeyExpression,
        authType: V1WebhookAuthType.HMAC,
        auth: {
          algorithm: V1WebhookHMACAlgorithm.SHA256,
          encoding: V1WebhookHMACEncoding.HEX,
          signatureHeaderName: 'Stripe-Signature',
          signingSecret: data.signingSecret,
        },
      };
    default:
      // eslint-disable-next-line no-case-declarations
      const exhaustiveCheck: never = data.sourceName;
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
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <div className="flex items-center justify-center w-8 h-8 bg-blue-100 rounded-full">
              <Webhook className="h-4 w-4 text-blue-600" />
            </div>
            Create a webhook
          </DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          <div className="space-y-2">
            <Label htmlFor="name" className="text-sm font-medium">
              Webhook ID <span className="text-red-500">*</span>
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
              Source Type <span className="text-red-500">*</span>
            </Label>
            <Select
              value={sourceName}
              onValueChange={(value: V1WebhookSourceName) =>
                setValue('sourceName', value)
              }
            >
              <SelectTrigger className="h-10">
                <SelectValue>
                  <SourceName sourceName={sourceName} />
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                {Object.values(V1WebhookSourceName).map((source) => (
                  <SelectItem key={source} value={source}>
                    <SourceName sourceName={source} />
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">
              Represents the producer of your HTTP requests.
            </p>
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
            <p className="text-xs text-muted-foreground">
              CEL expression to extract the event key from the webhook payload.
              Use `input` to refer to the payload.
            </p>
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
