import { t } from './copy';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';
import { V1ServerlessEndpointKind } from '@/lib/api';
import { cn } from '@/lib/utils';
import { Globe } from 'lucide-react';
import { SiCloudflare, SiVercel } from 'react-icons/si';

export const PlatformIcon = ({
  kind,
  className,
}: {
  kind: V1ServerlessEndpointKind;
  className?: string;
}) => {
  switch (kind) {
    case V1ServerlessEndpointKind.CLOUDFLARE_WORKERS:
      return (
        <SiCloudflare className={cn('size-4 text-[#f38020]', className)} />
      );
    case V1ServerlessEndpointKind.VERCEL:
      return <SiVercel className={cn('size-4', className)} />;
    case V1ServerlessEndpointKind.GENERIC_HTTP:
      return <Globe className={cn('size-4', className)} />;
    default:
      const exhaustiveCheck: never = kind;
      throw new Error(`Unhandled endpoint kind: ${exhaustiveCheck}`);
  }
};

export const PlatformName = ({ kind }: { kind: V1ServerlessEndpointKind }) => (
  <span className="flex flex-row items-center gap-x-2">
    <PlatformIcon kind={kind} />
    {t(`platform.${kind}.label`)}
  </span>
);

/** The platform's icon, naming the platform on hover. */
export const PlatformBadge = ({
  kind,
  className,
}: {
  kind: V1ServerlessEndpointKind;
  className?: string;
}) => (
  <TooltipProvider>
    <Tooltip>
      <TooltipTrigger asChild>
        <span
          className="inline-flex shrink-0 items-center"
          aria-label={t(`platform.${kind}.label`)}
        >
          <PlatformIcon kind={kind} className={className} />
        </span>
      </TooltipTrigger>
      <TooltipContent>{t(`platform.${kind}.label`)}</TooltipContent>
    </Tooltip>
  </TooltipProvider>
);
