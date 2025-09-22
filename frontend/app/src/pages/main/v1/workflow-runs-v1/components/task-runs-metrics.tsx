import { V1TaskStatus } from '@/lib/api';
import { Badge } from '@/components/v1/ui/badge';
import { useRunsContext } from '../hooks/runs-provider';
import { getStatusesFromFilters } from '../hooks/use-runs-table-state';
import { CheckCircleIcon, ClockIcon } from '@heroicons/react/24/outline';
import { PlayIcon, X, Ban, ChartColumn } from 'lucide-react';
import { cn } from '@/lib/utils';
import { useCallback } from 'react';

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
  status,
  className,
}: {
  status: V1TaskStatus;
  className?: string;
}) {
  const { filters, metrics } = useRunsContext();
  const currentStatuses = getStatusesFromFilters(filters.columnFilters);
  const isSelected = currentStatuses.includes(status);
  const { setStatuses } = filters;

  const handleStatusClick = useCallback(
    (status: V1TaskStatus) => {
      const currentStatuses = getStatusesFromFilters(filters.columnFilters);
      const isSelected = currentStatuses.includes(status);

      if (isSelected) {
        setStatuses(currentStatuses.filter((s) => s !== status));
      } else {
        setStatuses([...currentStatuses, status]);
      }
    },
    [filters.columnFilters, setStatuses],
  );

  const metric = metrics.find((m) => m.status === status);

  if (!metric) {
    return null;
  }

  const IconComponent = statusToIcon(status);
  const friendlyName = statusToFriendlyName(status);
  const formattedCount = metric.count.toLocaleString('en-US');

  return (
    <Badge
      data-is-selected={isSelected}
      variant={isSelected ? 'default' : 'outline'}
      className={cn('cursor-pointer text-sm px-3 py-1 w-fit h-8', className)}
      onClick={() => handleStatusClick(status)}
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
    display: { hideMetrics },
    actions: { updateUIState },
  } = useRunsContext();

  const onViewQueueMetricsClick = () => {
    updateUIState({ viewQueueMetrics: true });
  };

  // format of className strings is:
  // default, then unselected, then selected, then hover+selected, then hover+unselected
  return (
    <div className="flex flex-row justify-start gap-2">
      <MetricBadge
        status={V1TaskStatus.COMPLETED}
        className={`
          text-green-800 dark:text-green-300

          data-[is-selected=false]:border data-[is-selected=false]:border-green-500/20

          data-[is-selected=true]:bg-green-500/20

          hover:data-[is-selected=true]:border hover:data-[is-selected=true]:border-green-500/20 hover:data-[is-selected=true]:bg-inherit

          hover:data-[is-selected=false]:bg-green-500/20 hover:data-[is-selected=false]:border-transparent
          `}
      />

      <MetricBadge status={V1TaskStatus.RUNNING} />

      <MetricBadge status={V1TaskStatus.FAILED} />

      <MetricBadge status={V1TaskStatus.CANCELLED} />

      <MetricBadge status={V1TaskStatus.QUEUED} />

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
    </div>
  );
};
