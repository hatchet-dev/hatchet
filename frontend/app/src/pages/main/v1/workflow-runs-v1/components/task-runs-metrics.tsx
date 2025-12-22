import { useRunsContext } from '../hooks/runs-provider';
import { Badge } from '@/components/v1/ui/badge';
import { V1TaskStatus } from '@/lib/api';
import { cn } from '@/lib/utils';
import { CheckCircleIcon, ClockIcon } from '@heroicons/react/24/outline';
import { PlayIcon, X, Ban, ChartColumn } from 'lucide-react';
import { useCallback, useMemo } from 'react';

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
  const { filters, runStatusCounts } = useRunsContext();
  const currentStatuses = useMemo(
    () => filters.apiFilters.statuses || [],
    [filters.apiFilters.statuses],
  );
  const isSelected = currentStatuses.includes(status);
  const { setStatuses } = filters;

  const handleStatusClick = useCallback(
    (status: V1TaskStatus) => {
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
    [currentStatuses, setStatuses],
  );

  const metric = runStatusCounts.find((m) => m.status === status);

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
        'h-8 w-fit cursor-pointer px-3 py-1 text-sm data-[is-selected=false]:font-light',
        className,
      )}
      onClick={() => handleStatusClick(status)}
    >
      <span className="flex items-center gap-1">
        <span>{formattedCount}</span>
        <span className="cq-xl:inline hidden">{friendlyName}</span>
        <IconComponent className="cq-xl:hidden size-4" />
      </span>
    </Badge>
  );
}

export const V1WorkflowRunsMetricsView = () => {
  const {
    display: { hideMetrics },
    actions: { setShowQueueMetrics },
  } = useRunsContext();

  // format of className strings is:
  // default, then unselected, then selected, then hover+selected, then hover+unselected
  return (
    <div className="flex flex-row justify-start gap-2">
      <MetricBadge
        status={V1TaskStatus.COMPLETED}
        className={`text-green-800 data-[is-selected=false]:border data-[is-selected=false]:border-green-500/20 data-[is-selected=true]:bg-green-500/20 hover:data-[is-selected=false]:border-transparent hover:data-[is-selected=false]:bg-green-500/20 hover:data-[is-selected=true]:bg-green-500/20 dark:text-green-300`}
      />

      <MetricBadge
        status={V1TaskStatus.RUNNING}
        className={`text-yellow-800 data-[is-selected=false]:border data-[is-selected=false]:border-yellow-500/20 data-[is-selected=true]:bg-yellow-500/20 hover:data-[is-selected=false]:border-transparent hover:data-[is-selected=false]:bg-yellow-500/20 hover:data-[is-selected=true]:bg-yellow-500/20 dark:text-yellow-300`}
      />

      <MetricBadge
        status={V1TaskStatus.FAILED}
        className={`text-red-800 data-[is-selected=false]:border data-[is-selected=false]:border-red-500/20 data-[is-selected=true]:bg-red-500/20 hover:data-[is-selected=false]:border-transparent hover:data-[is-selected=false]:bg-red-500/20 hover:data-[is-selected=true]:bg-red-500/20 dark:text-red-300`}
      />

      <MetricBadge
        status={V1TaskStatus.CANCELLED}
        className={`text-orange-800 data-[is-selected=false]:border data-[is-selected=false]:border-orange-500/20 data-[is-selected=true]:bg-orange-500/20 hover:data-[is-selected=false]:border-transparent hover:data-[is-selected=false]:bg-orange-500/20 hover:data-[is-selected=true]:bg-orange-500/20 dark:text-orange-300`}
      />

      <MetricBadge
        status={V1TaskStatus.QUEUED}
        className={`text-slate-800 data-[is-selected=false]:border data-[is-selected=false]:border-slate-500/20 data-[is-selected=true]:bg-slate-500/20 hover:data-[is-selected=false]:border-transparent hover:data-[is-selected=false]:bg-slate-500/20 hover:data-[is-selected=true]:bg-slate-500/20 dark:text-slate-300`}
      />

      {!hideMetrics && (
        <Badge
          variant="outline"
          className="h-8 w-fit cursor-pointer rounded-sm px-3 py-1 text-sm font-normal"
          onClick={() => setShowQueueMetrics(true)}
        >
          <span className="cq-xl:inline hidden">Queue metrics</span>
          <ChartColumn className="cq-xl:hidden size-4" />
        </Badge>
      )}
    </div>
  );
};
