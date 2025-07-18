import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { useForm } from 'react-hook-form';
import { V1WebhookAuthType } from '@/lib/api';
import { WebhookFormData } from '../hooks/use-webhooks';

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
          onValueChange={(value: 'SHA1' | 'SHA256' | 'SHA512' | 'MD5') =>
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
          onValueChange={(value: 'HEX' | 'BASE64' | 'BASE64URL') =>
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
        id="signatureHeaderName"
        placeholder="X-Signature"
        {...register('signatureHeaderName')}
        className="h-10"
      />
    </div>
  </div>
);

export const AuthSetup = ({
  authMethod,
  register,
  watch,
  setValue,
}: HMACAuthProps & {
  authMethod: V1WebhookAuthType;
}) => {
  switch (authMethod) {
    case V1WebhookAuthType.BASIC:
      return <BasicAuth register={register} />;
    case V1WebhookAuthType.API_KEY:
      return <APIKeyAuth register={register} />;
    case V1WebhookAuthType.HMAC:
      return <HMACAuth register={register} watch={watch} setValue={setValue} />;
    default:
      return null;
  }
};
