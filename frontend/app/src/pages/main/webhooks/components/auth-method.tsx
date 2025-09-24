import { V1WebhookAuthType } from '@/lib/api';
import { Key, ShieldCheck, UserCheck } from 'lucide-react';

export const AuthMethod = ({
  authMethod,
}: {
  authMethod: V1WebhookAuthType;
}) => {
  switch (authMethod) {
    case V1WebhookAuthType.BASIC:
      return (
        <span className="flex flex-row gap-x-2 items-center">
          <UserCheck className="size-4" />
          Basic
        </span>
      );
    case V1WebhookAuthType.API_KEY:
      return (
        <span className="flex flex-row gap-x-2 items-center">
          <Key className="size-4" />
          API Key
        </span>
      );
    case V1WebhookAuthType.HMAC:
      return (
        <span className="flex flex-row gap-x-2 items-center">
          <ShieldCheck className="size-4" />
          HMAC
        </span>
      );

    default:
      const exhaustiveCheck: never = authMethod;
      throw new Error(`Unhandled auth method: ${exhaustiveCheck}`);
  }
};
