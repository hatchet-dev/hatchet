import { Badge, BadgeProps } from '@/next/components/ui/badge';
import { V1TaskStatus, WorkflowRunStatus } from '@/next/lib/api';
import { cn } from '@/next/lib/utils';

interface RunsBadgeProps extends BadgeProps {
  status?: WorkflowRunStatus | V1TaskStatus;
  count?: number;
  percentage?: number;
  animated?: boolean;
  isLoading?: boolean;
}

type StatusConfig = {
  colors: string;
  label: string;
};

type StatusKey = WorkflowRunStatus | V1TaskStatus;

export const RunStatusConfigs: Record<StatusKey, StatusConfig> = {
  [WorkflowRunStatus.RUNNING]: {
    colors: 'bg-blue-500 text-white border-blue-600',
    label: 'Running',
  },
  [WorkflowRunStatus.SUCCEEDED]: {
    colors: 'bg-green-500 text-white border-green-600',
    label: 'Succeeded',
  },
  [V1TaskStatus.COMPLETED]: {
    colors: 'bg-green-500 text-white border-green-600',
    label: 'Succeeded',
  },
  [WorkflowRunStatus.FAILED]: {
    colors: 'bg-red-500 text-white border-red-600',
    label: 'Failed',
  },
  [WorkflowRunStatus.CANCELLED]: {
    colors: 'bg-gray-500 text-white border-gray-600',
    label: 'Cancelled',
  },
  [V1TaskStatus.QUEUED]: {
    colors: 'bg-yellow-500 text-white border-yellow-600',
    label: 'Queued',
  },
  [WorkflowRunStatus.BACKOFF]: {
    colors: 'bg-orange-500 text-white border-orange-600',
    label: 'Backoff',
  },
  [WorkflowRunStatus.PENDING]: {
    colors: 'bg-gray-500 text-white border-gray-600',
    label: 'Pending',
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
        isLoading
          ? 'animate-pulse bg-gray-200 text-transparent border-gray-300'
          : config.colors,
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
