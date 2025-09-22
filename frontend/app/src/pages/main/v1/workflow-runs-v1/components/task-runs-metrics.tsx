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

      const allStatuses = Object.values(V1TaskStatus);

      const isAllSelected =
        currentStatuses.length === allStatuses.length &&
        allStatuses.every((s) => currentStatuses.includes(s));

      if (isSelected) {
        if (isAllSelected) {
          setStatuses([status]);
        } else {
          const newStatuses = currentStatuses.filter((s) => s !== status);

          if (newStatuses.length === 0) {
            setStatuses(allStatuses);
          } else {
            setStatuses(newStatuses);
          }
        }
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
      className={cn(
        'cursor-pointer text-sm px-3 py-1 w-fit h-8 data-[is-selected=false]:font-light',
        className,
      )}
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

          hover:data-[is-selected=true]:bg-green-500/20

          hover:data-[is-selected=false]:bg-green-500/20 hover:data-[is-selected=false]:border-transparent
          `}
      />

      <MetricBadge
        status={V1TaskStatus.RUNNING}
        className={`
          text-yellow-800 dark:text-yellow-300

          data-[is-selected=false]:border data-[is-selected=false]:border-yellow-500/20

          data-[is-selected=true]:bg-yellow-500/20

          hover:data-[is-selected=true]:bg-yellow-500/20

          hover:data-[is-selected=false]:bg-yellow-500/20 hover:data-[is-selected=false]:border-transparent
          `}
      />

      <MetricBadge
        status={V1TaskStatus.FAILED}
        className={`
          text-red-800 dark:text-red-300

          data-[is-selected=false]:border data-[is-selected=false]:border-red-500/20

          data-[is-selected=true]:bg-red-500/20

          hover:data-[is-selected=true]:bg-red-500/20

          hover:data-[is-selected=false]:bg-red-500/20 hover:data-[is-selected=false]:border-transparent
          `}
      />

      <MetricBadge
        status={V1TaskStatus.CANCELLED}
        className={`
          text-orange-800 dark:text-orange-300

          data-[is-selected=false]:border data-[is-selected=false]:border-orange-500/20

          data-[is-selected=true]:bg-orange-500/20

          hover:data-[is-selected=true]:bg-orange-500/20

          hover:data-[is-selected=false]:bg-orange-500/20 hover:data-[is-selected=false]:border-transparent
          `}
      />

      <MetricBadge
        status={V1TaskStatus.QUEUED}
        className={`
          text-fuchsia-800 dark:text-fuchsia-300

          data-[is-selected=false]:border data-[is-selected=false]:border-fuchsia-500/20

          data-[is-selected=true]:bg-fuchsia-500/20

          hover:data-[is-selected=true]:bg-fuchsia-500/20

          hover:data-[is-selected=false]:bg-fuchsia-500/20 hover:data-[is-selected=false]:border-transparent
          `}
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
    </div>
  );
};
