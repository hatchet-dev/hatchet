import { Badge, BadgeProps } from '@/components/v1/ui/badge';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';
import { SandboxProvider } from '@/lib/api/generated/cloud/data-contracts';

type ProviderState = 'healthy' | 'unhealthy' | 'pending' | 'disabled';

const providerState = (provider: SandboxProvider): ProviderState => {
  if (!provider.enabled) {
    return 'disabled';
  }

  if (provider.status.healthy === undefined) {
    return 'pending';
  }

  return provider.status.healthy ? 'healthy' : 'unhealthy';
};

const STATES: Record<
  ProviderState,
  { label: string; description: string; variant: BadgeProps['variant'] }
> = {
  healthy: {
    label: 'Healthy',
    description: 'The vendor accepted the API key.',
    variant: 'successful',
  },
  unhealthy: {
    label: 'Unhealthy',
    description: 'The vendor rejected the API key.',
    variant: 'failed',
  },
  pending: {
    label: 'Pending',
    description: 'Not checked yet.',
    variant: 'queued',
  },
  disabled: {
    label: 'Disabled',
    description: 'Serves no sandboxes until enabled.',
    variant: 'outline',
  },
};

export const ProviderStatusBadge = ({
  provider,
}: {
  provider: SandboxProvider;
}) => {
  const state = STATES[providerState(provider)];

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <span>
            <Badge variant={state.variant} className="whitespace-nowrap">
              {state.label}
            </Badge>
          </span>
        </TooltipTrigger>
        <TooltipContent className="max-w-sm">
          <div>{state.description}</div>
          {!provider.status.healthy && provider.status.error && (
            <div className="mt-1 font-mono text-xs">
              {provider.status.error}
            </div>
          )}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};
