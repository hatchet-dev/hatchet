import { V1WebhookSourceName } from '@/lib/api';
import { GitHubLogoIcon } from '@radix-ui/react-icons';
import { Webhook } from 'lucide-react';
import { CgLinear } from 'react-icons/cg';
import { FaSlack, FaStripeS } from 'react-icons/fa';

export const SourceName = ({
  sourceName,
}: {
  sourceName: V1WebhookSourceName;
}) => {
  switch (sourceName) {
    case V1WebhookSourceName.GENERIC:
      return (
        <span className="flex flex-row items-center gap-x-2">
          <Webhook className="size-4" />
          Generic
        </span>
      );
    case V1WebhookSourceName.GITHUB:
      return (
        <span className="flex flex-row items-center gap-x-2">
          <GitHubLogoIcon className="size-4" />
          GitHub
        </span>
      );
    case V1WebhookSourceName.STRIPE:
      return (
        <span className="flex flex-row items-center gap-x-2">
          <FaStripeS className="size-4" />
          Stripe
        </span>
      );
    case V1WebhookSourceName.SLACK:
      return (
        <span className="flex flex-row items-center gap-x-2">
          <FaSlack className="size-4" />
          Slack
        </span>
      );
    case V1WebhookSourceName.LINEAR:
      return (
        <span className="flex flex-row items-center gap-x-2">
          <CgLinear className="size-4" />
          Linear
        </span>
      );

    default:
      const exhaustiveCheck: never = sourceName;
      throw new Error(`Unhandled source: ${exhaustiveCheck}`);
  }
};
