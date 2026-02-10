import { V1WebhookSourceName } from '@/lib/api';
import { GitHubLogoIcon } from '@radix-ui/react-icons';
import { Webhook } from 'lucide-react';
import { CgLinear } from 'react-icons/cg';
import { FaSlack, FaStripeS } from 'react-icons/fa';

const SvixLogo = ({ className }: { className?: string }) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    viewBox="0 0 230 230"
    fill="currentColor"
    className={className}
  >
    <g transform="translate(10.7,10.6)">
      <path
        d="M208.8 104.4c0 57.6-46.8 104.4-104.4 104.4c-57.6 0-104.4-46.8-104.4-104.4c0-57.6 46.8-104.4 104.4-104.4c57.6 0 104.4 46.8 104.4 104.4Zm-125 42.4c-7.2-16.1-23.3-26.8-41-27.2c-11.7-0.3-22.8 3-31.8 9.1c-2.1-8.3-3.2-17-3.1-26c1-53.2 44.9-95.7 98.2-94.8c35 .7 65.3 19.8 81.7 48c-4.9 5.4-12.8 8.6-21.2 8.3c-8.1-0.2-15.5-5.1-18.8-12.5c-7.3-16.1-23.4-26.8-41-27.2c-8.2-0.2-16.2 1.9-23.3 5.9c-24.5 13.7-32 47-14.3 69.3c8 10.1 20.2 16.2 34.2 17.2c7.3 .5 12.8 3.1 16.5 7.7c6.6 8.4 5.6 21.6-2.1 28.9c-4.1 3.9-9.5 6-15.1 5.8c-8.1-0.2-15.5-5.1-18.9-12.5Zm82.2-57.7c-17.3-0.8-33.5-10.4-41-27.1c-3.4-7.4-10.8-12.3-18.9-12.5c-17.1-0.5-28 21.1-17.2 34.7c3.7 4.6 9.2 7.2 16.5 7.7c14 1 26.1 7.1 34.1 17.2c14.7 18.4 12.6 46.3-4.4 62.5c-20.6 19.5-54.8 15.6-70.4-7.9c-1.4-2.1-2.6-4.3-3.7-6.6c-3.3-7.4-10.7-12.3-18.8-12.5c-8.4-0.2-16.2 3-21.2 8.3c16.4 28.2 46.7 47.4 81.7 48c53.3 .9 97.2-41.5 98.2-94.8c.1-9-1-17.7-3.1-26c.4 1.8-13.4 6.6-15 7c-5.5 1.6-11.2 2.3-16.8 2Z"
        fillRule="evenodd"
      />
    </g>
    <ellipse
      cx="115"
      cy="115"
      rx="104.5"
      ry="104.5"
      fill="none"
      stroke="currentColor"
      strokeWidth="20"
    />
  </svg>
);

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
    case V1WebhookSourceName.SVIX:
      return (
        <span className="flex flex-row items-center gap-x-2">
          <SvixLogo className="size-4" />
          Svix
        </span>
      );
    default:
      const exhaustiveCheck: never = sourceName;
      throw new Error(`Unhandled source: ${exhaustiveCheck}`);
  }
};
