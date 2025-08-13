import { V1WebhookSourceName } from '@/lib/api';
import { GitHubLogoIcon } from '@radix-ui/react-icons';
import { Webhook } from 'lucide-react';
import { FaDiscord, FaSlack, FaStripeS } from 'react-icons/fa';

export const SourceName = ({
  sourceName,
}: {
  sourceName: V1WebhookSourceName;
}) => {
  switch (sourceName) {
    case V1WebhookSourceName.GENERIC:
      return (
        <span className="flex flex-row gap-x-2 items-center">
          <Webhook className="size-4" />
          Generic
        </span>
      );
    case V1WebhookSourceName.GITHUB:
      return (
        <span className="flex flex-row gap-x-2 items-center">
          <GitHubLogoIcon className="size-4" />
          GitHub
        </span>
      );
    case V1WebhookSourceName.STRIPE:
      return (
        <span className="flex flex-row gap-x-2 items-center">
          <FaStripeS className="size-4" />
          Stripe
        </span>
      );
    case V1WebhookSourceName.SLACK:
      return (
        <span className="flex flex-row gap-x-2 items-center">
          <FaSlack className="size-4" />
          Slack
        </span>
      );
    case V1WebhookSourceName.DISCORD:
      return (
        <span className="flex flex-row gap-x-2 items-center">
          <FaDiscord className="size-4" />
          Discord
        </span>
      );

    default:
      // eslint-disable-next-line no-case-declarations
      const exhaustiveCheck: never = sourceName;
      throw new Error(`Unhandled source: ${exhaustiveCheck}`);
  }
};
