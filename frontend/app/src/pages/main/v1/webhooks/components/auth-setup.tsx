import { WebhookFormData } from '../hooks/use-webhooks';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import {
  V1WebhookAuthType,
  V1WebhookHMACAlgorithm,
  V1WebhookHMACEncoding,
  V1WebhookSourceName,
} from '@/lib/api';
import { useForm } from 'react-hook-form';

type BaseAuthMethodProps = {
  register: ReturnType<typeof useForm<WebhookFormData>>['register'];
};

type HMACAuthProps = BaseAuthMethodProps & {
  watch: ReturnType<typeof useForm<WebhookFormData>>['watch'];
  setValue: ReturnType<typeof useForm<WebhookFormData>>['setValue'];
};

const BasicAuth = ({ register }: BaseAuthMethodProps) => (
  <div className="space-y-4">
    <div className="space-y-2">
      <Label htmlFor="username" className="text-sm font-medium">
        Username <span className="text-red-500">*</span>
      </Label>
      <Input
        data-1p-ignore
        id="username"
        placeholder="username"
        {...register('username')}
        className="h-10"
      />
    </div>

    <div className="space-y-2">
      <Label htmlFor="password" className="text-sm font-medium">
        Password <span className="text-red-500">*</span>
      </Label>
      <div className="relative">
        <Input
          data-1p-ignore
          id="password"
          type={'text'}
          placeholder="password"
          {...register('password')}
          className="h-10 pr-10"
        />
      </div>
    </div>
  </div>
);

const APIKeyAuth = ({ register }: BaseAuthMethodProps) => (
  <div className="space-y-4">
    <div className="space-y-2">
      <Label htmlFor="headerName" className="text-sm font-medium">
        Header Name <span className="text-red-500">*</span>
      </Label>
      <Input
        data-1p-ignore
        id="headerName"
        placeholder="X-API-Key"
        {...register('headerName')}
        className="h-10"
      />
    </div>

    <div className="space-y-2">
      <Label htmlFor="apiKey" className="text-sm font-medium">
        API Key <span className="text-red-500">*</span>
      </Label>
      <div className="relative">
        <Input
          data-1p-ignore
          id="apiKey"
          type={'text'}
          placeholder="API key..."
          {...register('apiKey')}
          className="h-10 pr-10"
        />
      </div>
    </div>
  </div>
);

const HMACAuth = ({ register, watch, setValue }: HMACAuthProps) => (
  <div className="space-y-4">
    <div className="space-y-2">
      <Label htmlFor="signingSecret" className="text-sm font-medium">
        Webhook Signing Secret <span className="text-red-500">*</span>
      </Label>
      <div className="relative">
        <Input
          data-1p-ignore
          id="signingSecret"
          type={'text'}
          placeholder="Secret key..."
          {...register('signingSecret')}
          className="h-10 pr-10"
        />
      </div>
    </div>

    <div className="grid grid-cols-2 gap-4">
      <div className="space-y-2">
        <Label htmlFor="algorithm" className="text-sm font-medium">
          Algorithm
        </Label>
        <Select
          value={watch('algorithm')}
          onValueChange={(value: V1WebhookHMACAlgorithm) =>
            setValue('algorithm', value)
          }
        >
          <SelectTrigger className="h-10">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="SHA256">SHA256</SelectItem>
            <SelectItem value="SHA1">SHA1</SelectItem>
            <SelectItem value="SHA512">SHA512</SelectItem>
            <SelectItem value="MD5">MD5</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="space-y-2">
        <Label htmlFor="encoding" className="text-sm font-medium">
          Encoding
        </Label>
        <Select
          value={watch('encoding')}
          onValueChange={(value: V1WebhookHMACEncoding) =>
            setValue('encoding', value)
          }
        >
          <SelectTrigger className="h-10">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="HEX">HEX</SelectItem>
            <SelectItem value="BASE64">BASE64</SelectItem>
            <SelectItem value="BASE64URL">BASE64URL</SelectItem>
          </SelectContent>
        </Select>
      </div>
    </div>

    <div className="space-y-2">
      <Label htmlFor="signatureHeaderName" className="text-sm font-medium">
        Signature Header Name
      </Label>
      <Input
        data-1p-ignore
        id="signatureHeaderName"
        placeholder="X-Signature"
        {...register('signatureHeaderName')}
        className="h-10"
      />
    </div>
  </div>
);

const PreconfiguredHMACAuth = ({
  register,
  secretLabel = 'Signing Secret',
  secretPlaceholder = 'super-secret',
  helpText,
  helpLink,
}: BaseAuthMethodProps & {
  secretLabel?: string;
  secretPlaceholder?: string;
  helpText?: string;
  helpLink?: string;
}) => (
  // Intended to be used for Stripe, Slack, Github, Linear, etc.
  <div className="space-y-4">
    <div className="space-y-2">
      <Label htmlFor="signingSecret" className="text-sm font-medium">
        {secretLabel} <span className="text-red-500">*</span>
      </Label>
      <div className="relative">
        <Input
          data-1p-ignore
          id="signingSecret"
          type={'text'}
          placeholder={secretPlaceholder}
          {...register('signingSecret')}
          className="h-10 pr-10"
        />
      </div>
      {helpText && (
        <div className="pl-1 text-xs text-muted-foreground">
          <p>
            {helpText}
            {helpLink && (
              <>
                {' '}
                <a
                  href={helpLink}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-600 hover:underline"
                >
                  Learn more
                </a>
                .
              </>
            )}
          </p>
        </div>
      )}
    </div>
  </div>
);

export const AuthSetup = ({
  authMethod,
  sourceName,
  register,
  watch,
  setValue,
}: HMACAuthProps & {
  authMethod: V1WebhookAuthType;
  sourceName: V1WebhookSourceName;
}) => {
  switch (sourceName) {
    case V1WebhookSourceName.GENERIC:
      switch (authMethod) {
        case V1WebhookAuthType.BASIC:
          return <BasicAuth register={register} />;
        case V1WebhookAuthType.API_KEY:
          return <APIKeyAuth register={register} />;
        case V1WebhookAuthType.HMAC:
          return (
            <HMACAuth register={register} watch={watch} setValue={setValue} />
          );
        default:
          const exhaustiveCheck: never = authMethod;
          throw new Error(`Unhandled auth method: ${exhaustiveCheck}`);
      }
    case V1WebhookSourceName.GITHUB:
      return <PreconfiguredHMACAuth register={register} />;
    case V1WebhookSourceName.LINEAR:
      return (
        <PreconfiguredHMACAuth
          register={register}
          secretPlaceholder="lin_wh_..."
        />
      );
    case V1WebhookSourceName.STRIPE:
      return (
        <PreconfiguredHMACAuth
          register={register}
          secretPlaceholder="whsec_..."
        />
      );
    case V1WebhookSourceName.SLACK:
      return (
        <PreconfiguredHMACAuth
          register={register}
          helpText="You can find your signing secret in the Basic Information panel of your Slack app settings."
          helpLink="https://docs.slack.dev/authentication/verifying-requests-from-slack/#validating-a-request"
        />
      );
    case V1WebhookSourceName.SVIX:
      return (
        <PreconfiguredHMACAuth
          register={register}
          secretLabel="Svix Endpoint Secret"
          secretPlaceholder="whsec_..."
          helpText="You can find your endpoint secret in the Svix dashboard under the endpoint's settings."
        />
      );
    default:
      const exhaustiveCheck: never = sourceName;
      throw new Error(`Unhandled source name: ${exhaustiveCheck}`);
  }
};
