import { t } from './copy';
import { Badge, BadgeProps } from '@/components/v1/ui/badge';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';
import { V1ServerlessEndpoint } from '@/lib/api';

type EndpointState = 'healthy' | 'unhealthy' | 'pending' | 'disabled';

export const endpointState = (
  endpoint: V1ServerlessEndpoint,
): EndpointState => {
  if (!endpoint.enabled) {
    return 'disabled';
  }

  if (endpoint.status.healthy === undefined) {
    return 'pending';
  }

  return endpoint.status.healthy ? 'healthy' : 'unhealthy';
};

const VARIANT: Record<EndpointState, BadgeProps['variant']> = {
  healthy: 'successful',
  unhealthy: 'failed',
  pending: 'queued',
  disabled: 'outline',
};

export const EndpointStatusBadge = ({
  endpoint,
}: {
  endpoint: V1ServerlessEndpoint;
}) => {
  const state = endpointState(endpoint);

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <span>
            <Badge variant={VARIANT[state]} className="whitespace-nowrap">
              {t(`status.${state}.label`)}
            </Badge>
          </span>
        </TooltipTrigger>
        <TooltipContent className="max-w-sm">
          <div>{t(`status.${state}.description`)}</div>
          {state === 'unhealthy' && endpoint.status.error && (
            <div className="mt-1 font-mono text-xs">
              {endpoint.status.error}
            </div>
          )}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};
