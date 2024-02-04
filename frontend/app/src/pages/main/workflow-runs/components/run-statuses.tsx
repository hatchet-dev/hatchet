import { Badge } from '@/components/ui/badge';
import { JobRunStatus, StepRunStatus, WorkflowRunStatus } from '@/lib/api';
import { capitalize, cn } from '@/lib/utils';

type RunStatusType = `${StepRunStatus | WorkflowRunStatus | JobRunStatus}`;

export function RunStatus({
  status,
  reason,
}: {
  status: RunStatusType;
  reason?: string;
}) {
  let variant: 'inProgress' | 'successful' | 'failed' = 'inProgress';
  let text = 'Running';

  switch (status) {
    case 'SUCCEEDED':
      variant = 'successful';
      text = 'Succeeded';
      break;
    case 'FAILED':
    case 'CANCELLED':
      variant = 'failed';
      text = 'Cancelled';

      switch (reason) {
        case 'TIMED_OUT':
          text = 'Timed out';
          break;
        case 'SCHEDULING_TIMED_OUT':
          text = 'No workers available';
          break;
        default:
          break;
      }

      break;
    default:
      break;
  }

  return <Badge variant={variant}>{capitalize(text)}</Badge>;
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
