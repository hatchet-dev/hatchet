import { Badge } from '@/components/ui/badge';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import { JobRunStatus, StepRunStatus, WorkflowRunStatus } from '@/lib/api';
import { capitalize, cn } from '@/lib/utils';

type RunStatusType =
  `${StepRunStatus | WorkflowRunStatus | JobRunStatus | 'SCHEDULED'}`;

type RunStatusVariant = {
  text: string;
  variant:
    | 'inProgress'
    | 'successful'
    | 'failed'
    | 'outline'
    | 'outlineDestructive';
};

const RUN_STATUS_VARIANTS: Record<RunStatusType, RunStatusVariant> = {
  SUCCEEDED: {
    text: 'Succeeded',
    variant: 'successful',
  },
  FAILED: {
    text: 'Failed',
    variant: 'failed',
  },
  CANCELLED: {
    text: 'Cancelled',
    variant: 'outlineDestructive',
  },
  CANCELLING: {
    text: 'Cancelling',
    variant: 'inProgress',
  },
  RUNNING: {
    text: 'Running',
    variant: 'inProgress',
  },
  QUEUED: {
    text: 'Queued',
    variant: 'outline',
  },
  PENDING: {
    text: 'Pending',
    variant: 'outline',
  },
  PENDING_ASSIGNMENT: {
    text: 'Pending',
    variant: 'outline',
  },
  ASSIGNED: {
    text: 'Assigned',
    variant: 'inProgress',
  },
  SCHEDULED: {
    text: 'Scheduled',
    variant: 'outline',
  },
};

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
  const { text, variant } = RUN_STATUS_VARIANTS[status];
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

const indicatorVariants = {
  successful: 'border-transparent rounded-full bg-green-500',
  failed: 'border-transparent rounded-full bg-red-500',
  inProgress: 'border-transparent rounded-full bg-yellow-500',
  outline: 'border-transparent rounded-full bg-muted',
  outlineDestructive: 'border-transparent rounded-full bg-red-500',
};

export function RunIndicator({
  status,
}: {
  status: RunStatusType;
  reason?: string;
}) {
  const variant = RUN_STATUS_VARIANTS[status].variant;

  return (
    <div
      className={cn(indicatorVariants[variant], 'rounded-full h-[6px] w-[6px]')}
    />
  );
}
