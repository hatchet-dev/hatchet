import { Badge, BadgeProps } from '@/next/components/ui/badge';
import { V1TaskStatus, WorkflowRunStatus } from '@/lib/api';
import { cn } from '@/next/lib/utils';

interface RunsBadgeProps extends BadgeProps {
  status?: StatusKey;
  count?: number;
  percentage?: number;
  animated?: boolean;
  isLoading?: boolean;
}

type StatusConfig = {
  colors: string;
  primary: string;
  primaryOKLCH: string;
  label: string;
};

type StatusKey = WorkflowRunStatus | V1TaskStatus | 'QueueMetrics';

export const RunStatusConfigs: Record<StatusKey, StatusConfig> = {
  [WorkflowRunStatus.RUNNING]: {
    colors:
      'text-indigo-800 dark:text-indigo-300 bg-indigo-500/20 ring-indigo-500/30',
    primary: 'text-indigo-500 bg-indigo-500',
    primaryOKLCH: 'oklch(0.585 0.233 277.117)',
    label: 'Running',
  },
  [WorkflowRunStatus.SUCCEEDED]: {
    colors:
      'text-green-800 dark:text-green-300 bg-green-500/20 ring-green-500/30',
    primary: 'text-green-500 bg-green-500',
    primaryOKLCH: 'oklch(0.723 0.219 149.579)',
    label: 'Succeeded',
  },
  [V1TaskStatus.COMPLETED]: {
    colors:
      'text-green-800 dark:text-green-300 bg-green-500/20 ring-green-500/30',
    primary: 'text-green-500 bg-green-500',
    primaryOKLCH: 'oklch(0.723 0.219 149.579)',
    label: 'Succeeded',
  },
  [WorkflowRunStatus.FAILED]: {
    colors: 'text-red-800 dark:text-red-300 bg-red-500/20 ring-red-500',
    primary: 'text-red-500 bg-red-500',
    primaryOKLCH: 'oklch(0.637 0.237 25.331)',
    label: 'Failed',
  },
  [WorkflowRunStatus.CANCELLED]: {
    colors: 'text-gray-800 dark:text-gray-300 bg-gray-500/20 ring-gray-500/30',
    primary: 'text-gray-500 bg-gray-500',
    primaryOKLCH: 'oklch(0.551 0.027 264.364)',
    label: 'Cancelled',
  },
  [V1TaskStatus.QUEUED]: {
    colors:
      'text-yellow-800 dark:text-yellow-300 bg-yellow-500/20 ring-yellow-500/30',
    primary: 'text-yellow-500 bg-yellow-500',
    primaryOKLCH: 'oklch(0.795 0.184 86.047)',
    label: 'Queued',
  },
  [WorkflowRunStatus.BACKOFF]: {
    colors:
      'text-orange-800 dark:text-orange-300 bg-orange-500/20 ring-orange-500/30',
    primary: 'text-orange-500 bg-orange-500',
    primaryOKLCH: 'oklch(0.705 0.213 47.604)',
    label: 'Backoff',
  },
  [WorkflowRunStatus.PENDING]: {
    colors: 'text-gray-800 dark:text-gray-300 bg-gray-500/20 ring-gray-500/30',
    primary: 'text-gray-500 bg-gray-500',
    primaryOKLCH: 'oklch(0.551 0.027 264.364)',
    label: 'Pending',
  },
  QueueMetrics: {
    colors: 'text-gray-800 dark:text-gray-300 bg-gray-500/20 ring-gray-500/30',
    primary: 'text-gray-500 bg-gray-500',
    primaryOKLCH: 'oklch(0.551 0.027 264.364)',
    label: 'Queue Metrics',
  },
};

export function RunsBadge({
  status,
  variant,
  count,
  percentage,
  animated,
  isLoading,
  className,
  ...props
}: RunsBadgeProps) {
  const config = !status
    ? RunStatusConfigs.PENDING
    : RunStatusConfigs[status] || {
        colors: 'bg-gray-50 text-gray-700 border-gray-200',
        primary: 'text-gray-500',
        label: 'Pending',
      };

  const content =
    variant === 'detail' ? (
      <>
        {count?.toLocaleString('en-US')} {config.label} ({percentage}%)
      </>
    ) : variant !== 'xs' ? (
      config.label
    ) : null;

  return (
    <Badge
      className={cn(
        variant === 'xs' ? 'p-0 w-2 h-2' : 'px-3 py-1',
        isLoading
          ? 'animate-pulse bg-gray-200/20 text-transparent'
          : variant === 'xs'
            ? config.primary
            : config.colors,
        'text-xs font-medium rounded-md border-transparent',
        className,
      )}
      tooltipContent={status}
      animated={
        isLoading
          ? false
          : animated !== undefined
            ? animated
            : status === V1TaskStatus.RUNNING
      }
      variant={variant}
      {...props}
    >
      {content}
    </Badge>
  );
}
