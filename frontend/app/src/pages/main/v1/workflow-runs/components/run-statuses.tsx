import { Badge } from '@/components/v1/ui/badge';
import { HoverCard, HoverCardTrigger } from '@/components/v1/ui/hover-card';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';
import {
  JobRunStatus,
  StepRunStatus,
  V2TaskStatus,
  WorkflowRunStatus,
} from '@/lib/api';
import { capitalize, cn } from '@/lib/utils';
import { HoverCardContent } from '@radix-ui/react-hover-card';

type RunStatusType =
  `${StepRunStatus | WorkflowRunStatus | JobRunStatus | 'SCHEDULED'}`;

type RunStatusVariant = {
  text: string;
  variant: 'inProgress' | 'successful' | 'failed' | 'outline';
};

export function createRunStatusVariant(
  status: RunStatusType,
): RunStatusVariant {
  switch (status) {
    case 'SUCCEEDED':
      return { text: 'Succeeded', variant: 'successful' };
    case 'FAILED':
      return { text: 'Failed', variant: 'failed' };
    case 'CANCELLED':
      return { text: 'Cancelled', variant: 'failed' };
    case 'CANCELLING':
      return { text: 'Cancelling', variant: 'inProgress' };
    case 'RUNNING':
      return { text: 'Running', variant: 'inProgress' };
    case 'QUEUED':
      return { text: 'Queued', variant: 'outline' };
    case 'PENDING':
      return { text: 'Pending', variant: 'outline' };
    case 'PENDING_ASSIGNMENT':
      return { text: 'Pending', variant: 'outline' };
    case 'ASSIGNED':
      return { text: 'Assigned', variant: 'inProgress' };
    case 'SCHEDULED':
      return { text: 'Scheduled', variant: 'outline' };
    default:
      return { text: 'Unknown', variant: 'outline' };
  }
}

export function createV2RunStatusVariant(
  status: V2TaskStatus,
): RunStatusVariant {
  switch (status) {
    case V2TaskStatus.COMPLETED:
      return { text: 'Succeeded', variant: 'successful' };
    case V2TaskStatus.FAILED:
      return { text: 'Failed', variant: 'failed' };
    case V2TaskStatus.CANCELLED:
      return { text: 'Cancelled', variant: 'failed' };
    case V2TaskStatus.RUNNING:
      return { text: 'Running', variant: 'inProgress' };
    case V2TaskStatus.QUEUED:
      return { text: 'Queued', variant: 'outline' };
    default:
      return { text: 'Unknown', variant: 'outline' };
  }
}

const RUN_STATUS_REASONS: Record<string, string> = {
  TIMED_OUT: 'Runtime Timed Out',
  SCHEDULING_TIMED_OUT: 'Scheduling Timed Out',
};

const RUN_STATUS_VARIANTS_REASON_OVERRIDES: Record<
  keyof typeof RUN_STATUS_REASONS,
  RunStatusVariant
> = {
  TIMED_OUT: {
    text: 'Timed Out',
    variant: 'failed',
  },
  SCHEDULING_TIMED_OUT: {
    text: 'Timed Out',
    variant: 'failed',
  },
};

// TIMED_OUT
// SCHEDULING_TIMED_OUT

export function RunStatus({
  status,
  reason,
  className,
}: {
  status: RunStatusType;
  reason?: string;
  className?: string;
}) {
  const { text, variant } = createRunStatusVariant(status);
  const { text: overrideText, variant: overrideVariant } =
    (reason && RUN_STATUS_VARIANTS_REASON_OVERRIDES[reason]) || {};

  const StatusBadge = () => (
    <Badge variant={overrideVariant || variant} className={className}>
      {capitalize(overrideText || text)}
    </Badge>
  );

  if (!reason) {
    return <StatusBadge />;
  }

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger>
          <StatusBadge />
        </TooltipTrigger>
        <TooltipContent>{RUN_STATUS_REASONS[reason] || reason}</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

export function V2RunStatus({
  status,
  errorMessage,
  className,
}: {
  status: V2TaskStatus;
  errorMessage?: string;
  className?: string;
}) {
  const { text, variant } = createV2RunStatusVariant(status);

  const StatusBadge = () => (
    <Badge variant={variant} className={className}>
      {capitalize(text)}
    </Badge>
  );

  if (!errorMessage) {
    return <StatusBadge />;
  }

  return (
    <HoverCard>
      <HoverCardTrigger>
        <StatusBadge />
      </HoverCardTrigger>
      <HoverCardContent className="bg-card max-w-96 overflow-auto z-10 shadow-xl p-4 rounded-md border border-gray-600 border-opacity-50">
        <p className="text-xs">{errorMessage}</p>
      </HoverCardContent>
    </HoverCard>
  );
}

const indicatorVariants = {
  successful: 'border-transparent rounded-full bg-green-500',
  failed: 'border-transparent rounded-full bg-red-500',
  inProgress: 'border-transparent rounded-full bg-yellow-500',
  outline: 'border-transparent rounded-full bg-muted',
};

export function createV2IndicatorVariant(eventType: V2TaskStatus | undefined) {
  switch (eventType) {
    case V2TaskStatus.CANCELLED:
    case V2TaskStatus.FAILED:
      return 'border-transparent rounded-full bg-red-500';
    case V2TaskStatus.RUNNING:
    case V2TaskStatus.QUEUED:
      return 'border-transparent rounded-full bg-yellow-500';
    case V2TaskStatus.COMPLETED:
      return 'border-transparent rounded-full bg-green-500';
    default:
      return 'border-transparent rounded-full bg-muted';
  }
}

export function RunIndicator({
  status,
}: {
  status: RunStatusType;
  reason?: string;
}) {
  const variant = createRunStatusVariant(status).variant;

  return (
    <div
      className={cn(indicatorVariants[variant], 'rounded-full h-[6px] w-[6px]')}
    />
  );
}

export function V2RunIndicator({ status }: { status: V2TaskStatus }) {
  const indicator = createV2IndicatorVariant(status);

  return <div className={cn(indicator, 'rounded-full h-[6px] w-[6px]')} />;
}
