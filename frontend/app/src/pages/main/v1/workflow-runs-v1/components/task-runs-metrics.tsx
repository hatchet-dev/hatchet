import { V1TaskRunMetrics, V1TaskStatus } from '@/lib/api';
import { Badge, badgeVariants } from '@/components/v1/ui/badge';
import { VariantProps } from 'class-variance-authority';
import { useRunsContext } from '../hooks/runs-provider';
import { getStatusesFromFilters } from '../hooks/use-runs-table-state';
import { CheckCircleIcon, ClockIcon } from '@heroicons/react/24/outline';
import { PlayIcon, X, Ban, ChartColumn } from 'lucide-react';
import { cn } from '@/lib/utils';

function statusToFriendlyName(status: V1TaskStatus) {
  switch (status) {
    case V1TaskStatus.CANCELLED:
      return 'Cancelled';
    case V1TaskStatus.COMPLETED:
      return 'Succeeded';
    case V1TaskStatus.FAILED:
      return 'Failed';
    case V1TaskStatus.QUEUED:
      return 'Queued';
    case V1TaskStatus.RUNNING:
      return 'Running';
    default:
      const exhaustivenessCheck: never = status;
      throw new Error(`Unknown status: ${exhaustivenessCheck}`);
  }
}

function statusToIcon(status: V1TaskStatus) {
  switch (status) {
    case V1TaskStatus.COMPLETED:
      return CheckCircleIcon;
    case V1TaskStatus.FAILED:
      return X;
    case V1TaskStatus.CANCELLED:
      return Ban;
    case V1TaskStatus.RUNNING:
      return PlayIcon;
    case V1TaskStatus.QUEUED:
      return ClockIcon;
    default:
      const exhaustivenessCheck: never = status;
      throw new Error(`Unknown status: ${exhaustivenessCheck}`);
  }
}

function MetricBadge({
  metrics,
  status,
  onClick,
  variant,
  className,
}: {
  metrics: V1TaskRunMetrics;
  status: V1TaskStatus;
  onClick?: (status: V1TaskStatus) => void;
  variant: VariantProps<typeof badgeVariants>['variant'];
  className?: string;
}) {
  const metric = metrics.find((m) => m.status === status);

  if (!metric) {
    return null;
  }

  const IconComponent = statusToIcon(status);
  const friendlyName = statusToFriendlyName(status);
  const formattedCount = metric.count.toLocaleString('en-US');

  return (
    <Badge
      variant={variant}
      className={cn('cursor-pointer text-sm px-3 py-1 w-fit h-8', className)}
      onClick={() => onClick?.(status)}
    >
      <span className="flex items-center gap-1">
        <span>{formattedCount}</span>
        <span className="cq-xl:inline hidden">{friendlyName}</span>
        <IconComponent className="size-4 cq-xl:hidden" />
      </span>
    </Badge>
  );
}

export const V1WorkflowRunsMetricsView = () => {
  const {
    metrics,
    filters,
    display: { hideMetrics },
    actions: { updateUIState },
  } = useRunsContext();

  const { setStatuses } = filters;

  const onViewQueueMetricsClick = () => {
    updateUIState({ viewQueueMetrics: true });
  };

  const handleStatusClick = (status: V1TaskStatus) => {
    const currentStatuses = getStatusesFromFilters(filters.columnFilters);
    const isSelected = currentStatuses.includes(status);

    if (isSelected) {
      setStatuses(currentStatuses.filter((s) => s !== status));
    } else {
      setStatuses([...currentStatuses, status]);
    }
  };

  return (
    <dl className="flex flex-row justify-start gap-2">
      <MetricBadge
        metrics={metrics}
        status={V1TaskStatus.COMPLETED}
        onClick={handleStatusClick}
        variant="successful"
      />

      <MetricBadge
        metrics={metrics}
        status={V1TaskStatus.RUNNING}
        onClick={handleStatusClick}
        variant="inProgress"
      />

      <MetricBadge
        metrics={metrics}
        status={V1TaskStatus.FAILED}
        onClick={handleStatusClick}
        variant="failed"
      />

      <MetricBadge
        metrics={metrics}
        status={V1TaskStatus.CANCELLED}
        onClick={handleStatusClick}
        variant="outlineDestructive"
      />

      <MetricBadge
        metrics={metrics}
        status={V1TaskStatus.QUEUED}
        onClick={handleStatusClick}
        variant="outline"
        className="rounded-sm font-normal"
      />

      {!hideMetrics && (
        <Badge
          variant="outline"
          className="rounded-sm font-normal cursor-pointer text-sm px-3 py-1 w-fit h-8"
          onClick={() => onViewQueueMetricsClick()}
        >
          <span className="cq-xl:inline hidden">Queue metrics</span>
          <ChartColumn className="size-4 cq-xl:hidden" />
        </Badge>
      )}
    </dl>
  );
};
