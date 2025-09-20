import { V1TaskRunMetrics, V1TaskStatus } from '@/lib/api';
import { Badge, badgeVariants } from '@/components/v1/ui/badge';
import { VariantProps } from 'class-variance-authority';
import { useRunsContext } from '../hooks/runs-provider';
import { getStatusesFromFilters } from '../hooks/use-runs-table-state';
import {
  CheckCircleIcon,
  XCircleIcon,
  ClockIcon,
} from '@heroicons/react/24/outline';
import { PlayIcon, PauseIcon, ChartColumn } from 'lucide-react';
import { useSidePanel } from '@/hooks/use-side-panel';

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
      return XCircleIcon;
    case V1TaskStatus.CANCELLED:
      return PauseIcon;
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
  className: string;
}) {
  const metric = metrics.find((m) => m.status === status);

  if (!metric) {
    return null;
  }

  const IconComponent = statusToIcon(status);
  const friendlyName = statusToFriendlyName(status);
  const formattedCount = metric.count.toLocaleString('en-US');
  const { isOpen } = useSidePanel();

  return (
    <Badge
      variant={variant}
      className={className}
      onClick={() => onClick?.(status)}
    >
      <span className="flex items-center gap-1">
        <span>{formattedCount}</span>
        <span className="cq-lg:inline hidden">{friendlyName}</span>
        <IconComponent className="size-4 cq-lg:hidden" />
      </span>
    </Badge>
  );
}

export const V1WorkflowRunsMetricsView = () => {
  const {
    metrics,
    state,
    display: { hideMetrics },
    filters: { setStatuses },
    actions: { updateUIState },
  } = useRunsContext();

  const onViewQueueMetricsClick = () => {
    updateUIState({ viewQueueMetrics: true });
  };

  const handleStatusClick = (status: V1TaskStatus) => {
    const currentStatuses = getStatusesFromFilters(state.columnFilters);
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
        className="cursor-pointer text-sm px-2 py-1 w-fit"
      />

      <MetricBadge
        metrics={metrics}
        status={V1TaskStatus.RUNNING}
        onClick={handleStatusClick}
        variant="inProgress"
        className="cursor-pointer text-sm px-2 py-1 w-fit"
      />

      <MetricBadge
        metrics={metrics}
        status={V1TaskStatus.FAILED}
        onClick={handleStatusClick}
        variant="failed"
        className="cursor-pointer text-sm px-2 py-1 w-fit"
      />

      <MetricBadge
        metrics={metrics}
        status={V1TaskStatus.CANCELLED}
        onClick={handleStatusClick}
        variant="outlineDestructive"
        className="cursor-pointer text-sm px-2 py-1 w-fit"
      />

      <MetricBadge
        metrics={metrics}
        status={V1TaskStatus.QUEUED}
        onClick={handleStatusClick}
        variant="outline"
        className="cursor-pointer rounded-sm font-normal text-sm px-2 py-1 w-fit"
      />

      {!hideMetrics && (
        <Badge
          variant="outline"
          className="cursor-pointer rounded-sm font-normal text-sm px-2 py-1 w-fit"
          onClick={() => onViewQueueMetricsClick()}
        >
          <span className="cq-lg:inline hidden">Queue metrics</span>
          <ChartColumn className="size-4 cq-lg:hidden" />
        </Badge>
      )}
    </dl>
  );
};
