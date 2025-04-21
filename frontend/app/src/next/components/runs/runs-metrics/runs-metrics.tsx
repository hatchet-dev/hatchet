import { V1TaskRunMetric, V1TaskRunMetrics, V1TaskStatus } from '@/lib/api';
import { RunsBadge } from '../runs-badge';
import { percent } from '@/next/lib/utils/percent';

interface V1TaskRunMetricsProps {
  metrics: {
    data: V1TaskRunMetric[];
    isLoading: boolean;
  };
  onClick?: (status?: V1TaskStatus) => void;
  onViewQueueMetricsClick?: () => void;
  showQueueMetrics?: boolean;
}

function MetricBadge({
  metrics,
  status,
  total,
  onClick,
  className,
  isLoading,
}: {
  metrics: V1TaskRunMetrics;
  status: V1TaskStatus;
  total: number;
  onClick: (status: V1TaskStatus) => void;
  className: string;
  isLoading: boolean;
}) {
  const metric = metrics.find((m) => m.status === status);

  const percentage = percent(metric?.count || 0, total);

  return (
    <RunsBadge
      status={status}
      count={metric?.count}
      percentage={percentage}
      className={className}
      variant="detail"
      onClick={() => onClick(status)}
      isLoading={isLoading}
      animated={false}
    />
  );
}

export const RunsMetricsView = ({
  metrics,
  showQueueMetrics = false,
  onClick = () => {},
  onViewQueueMetricsClick = () => {},
}: V1TaskRunMetricsProps) => {
  const total = metrics.data
    .map((m) => m.count)
    .reduce((acc, curr) => acc + curr, 0);

  return (
    <dl className="flex flex-row justify-start gap-6">
      <MetricBadge
        metrics={metrics.data}
        status={V1TaskStatus.COMPLETED}
        total={total}
        onClick={onClick}
        className="cursor-pointer text-sm px-2 py-1 w-fit"
        isLoading={metrics.isLoading}
      />

      <MetricBadge
        metrics={metrics.data}
        status={V1TaskStatus.RUNNING}
        total={total}
        onClick={onClick}
        className="cursor-pointer text-sm px-2 py-1 w-fit"
        isLoading={metrics.isLoading}
      />

      <MetricBadge
        metrics={metrics.data}
        status={V1TaskStatus.FAILED}
        total={total}
        onClick={onClick}
        className="cursor-pointer text-sm px-2 py-1 w-fit"
        isLoading={metrics.isLoading}
      />

      <MetricBadge
        metrics={metrics.data}
        status={V1TaskStatus.QUEUED}
        total={total}
        onClick={onClick}
        className="cursor-pointer rounded-sm font-normal text-sm px-2 py-1 w-fit"
        isLoading={metrics.isLoading}
      />

      {showQueueMetrics && (
        <RunsBadge
          className="cursor-pointer rounded-sm font-normal text-sm px-2 py-1 w-fit"
          onClick={() => onViewQueueMetricsClick()}
          isLoading={metrics.isLoading}
        >
          Queue metrics
        </RunsBadge>
      )}
    </dl>
  );
};
