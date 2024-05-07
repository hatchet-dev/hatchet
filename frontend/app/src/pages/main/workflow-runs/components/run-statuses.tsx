import { Badge } from '@/components/ui/badge';
import { JobRunStatus, StepRunStatus, WorkflowRunStatus } from '@/lib/api';
import { capitalize, cn } from '@/lib/utils';

type RunStatusType = `${StepRunStatus | WorkflowRunStatus | JobRunStatus}`;

const INDICATORS: Record<
  RunStatusType,
  {
    variant: 'inProgress' | 'successful' | 'failed' | 'outline';
    text: string;
  }
> = {
  PENDING: {
    variant: 'outline',
    text: 'Pending',
  },
  PENDING_ASSIGNMENT: {
    variant: 'outline',
    text: 'Pending assignment',
  },
  ASSIGNED: {
    variant: 'inProgress',
    text: 'Assigned',
  },
  RUNNING: {
    variant: 'inProgress',
    text: 'Running',
  },
  SUCCEEDED: {
    variant: 'successful',
    text: 'Succeeded',
  },
  FAILED: {
    variant: 'failed',
    text: 'Failed',
  },
  CANCELLED: {
    variant: 'failed',
    text: 'Cancelled',
  },
  QUEUED: {
    variant: 'inProgress',
    text: 'Queued',
  },
};

export function RunStatus({
  status,
  reason,
}: {
  status: RunStatusType;
  reason?: string;
}) {
  const indicator = INDICATORS[status];

  return (
    <Badge variant={indicator.variant}>{capitalize(indicator.text)}</Badge>
  );
}

const indicatorVariants = {
  successful: 'border-transparent rounded-full bg-green-500',
  failed: 'border-transparent rounded-full bg-red-500',
  inProgress: 'border-transparent rounded-full bg-[#4EB4D7]',
};

export function RunIndicator({
  status,
  reason,
}: {
  status: RunStatusType;
  reason?: string;
}) {
  let variant: 'inProgress' | 'successful' | 'failed' = 'inProgress';

  switch (status) {
    case 'SUCCEEDED':
      variant = 'successful';
      break;
    case 'FAILED':
    case 'CANCELLED':
      variant = 'failed';

      switch (reason) {
        case 'TIMED_OUT':
          break;
        case 'SCHEDULING_TIMED_OUT':
          break;
        default:
          break;
      }

      break;
    default:
      break;
  }

  return (
    <div
      className={cn(indicatorVariants[variant], 'rounded-full h-[6px] w-[6px]')}
    />
  );
}
