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
import { V1WebhookSourceName, V1WebhookAuthType } from '@/lib/api';
import { Webhook } from 'lucide-react';
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

const CreateWebhookModal = () => {
  const { mutations } = useWebhooks();
  const { createWebhook, isCreatePending } = mutations;
  const [open, setOpen] = useState(false);

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
      authType: V1WebhookAuthType.HMAC,
      eventKeyExpression: 'body.id',
      algorithm: 'SHA256',
      encoding: 'HEX',
      signatureHeaderName: 'X-Signature',
    },
  });

  const sourceName = watch('sourceName');
  const authType = watch('authType');

  const onSubmit = useCallback(
    (data: WebhookFormData) => {
      const webhookData: any = {
        name: data.name,
        sourceName: data.sourceName,
        eventKeyExpression: data.eventKeyExpression,
        authType: data.authType,
      };

      webhookData.authType = data.authType;

      if (data.authType === V1WebhookAuthType.BASIC) {
        webhookData.auth = {
          username: data.username,
          password: data.password,
        };
      } else if (data.authType === V1WebhookAuthType.API_KEY) {
        webhookData.auth = {
          headerName: data.headerName,
          apiKey: data.apiKey,
        };
      } else if (data.authType === V1WebhookAuthType.HMAC) {
        webhookData.auth = {
          algorithm: data.algorithm,
          encoding: data.encoding,
          signatureHeaderName: data.signatureHeaderName,
          signingSecret: data.signingSecret,
        };
      }

      createWebhook(webhookData, {
        onSuccess: () => {
          setOpen(false);
          reset();
        },
      });
    },
    [createWebhook, reset],
  );

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="default">Create Webhook</Button>
      </DialogTrigger>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <div className="flex items-center justify-center w-8 h-8 bg-blue-100 rounded-full">
              <Webhook className="h-4 w-4 text-blue-600" />
            </div>
            Create a Hatchet webhook
          </DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
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
              placeholder="body.id"
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

          <div className="space-y-4">
            <div className="space-y-4 pl-4 border-l-2 border-gray-200">
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

              <AuthSetup
                authMethod={authType}
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
