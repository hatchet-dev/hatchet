import React from 'react';

import {
  V1TaskRunMetric,
  V1TaskRunMetrics,
  V1TaskStatus,
  WorkflowRunStatus,
  WorkflowRunsMetrics,
} from '@/lib/api';
import { Badge, badgeVariants } from '@/components/v1/ui/badge';
import { VariantProps } from 'class-variance-authority';

interface WorkflowRunsMetricsProps {
  metrics: WorkflowRunsMetrics;
  onClick?: (status?: WorkflowRunStatus) => void;
  onViewQueueMetricsClick?: () => void;
  showQueueMetrics?: boolean;
}

interface V1TaskRunMetricsProps {
  metrics: V1TaskRunMetric[];
  onClick?: (status?: V1TaskStatus) => void;
  onViewQueueMetricsClick?: () => void;
  showQueueMetrics?: boolean;
}

const calculatePercentage = (value: number, total: number): number => {
  const res = Math.round((value / total) * 100);

  if (isNaN(res)) {
    return 0;
  }

  return res;
};

export const WorkflowRunsMetricsView: React.FC<WorkflowRunsMetricsProps> = ({
  metrics: { counts },
  showQueueMetrics = false,
  onClick = () => {},
  onViewQueueMetricsClick = () => {},
}) => {
  const total =
    (counts?.PENDING ?? 0) +
    (counts?.RUNNING ?? 0) +
    (counts?.SUCCEEDED ?? 0) +
    (counts?.QUEUED ?? 0) +
    (counts?.FAILED ?? 0);

  const succeededPercentage = calculatePercentage(
    counts?.SUCCEEDED ?? 0,
    total,
  );
  const runningPercentage = calculatePercentage(counts?.RUNNING ?? 0, total);
  const failedPercentage = calculatePercentage(counts?.FAILED ?? 0, total);
  const pendingPercentage = calculatePercentage(counts?.PENDING ?? 0, total);
  const queuedPercentage = calculatePercentage(counts?.QUEUED ?? 0, total);

  return (
    <dl className="flex flex-row justify-start gap-6">
      <Badge
        variant="successful"
        className="cursor-pointer text-sm px-2 py-1 w-fit"
        onClick={() => onClick(WorkflowRunStatus.SUCCEEDED)}
      >
        {counts?.SUCCEEDED?.toLocaleString('en-US')} Succeeded (
        {succeededPercentage}%)
      </Badge>
      <Badge
        variant="inProgress"
        className="cursor-pointer text-sm px-2 py-1 w-fit"
        onClick={() => onClick(WorkflowRunStatus.RUNNING)}
      >
        {counts?.RUNNING?.toLocaleString('en-US')} Running ({runningPercentage}
        %)
      </Badge>
      <Badge
        variant="failed"
        className="cursor-pointer text-sm px-2 py-1 w-fit"
        onClick={() => onClick(WorkflowRunStatus.FAILED)}
      >
        {counts?.FAILED?.toLocaleString('en-US')} Failed ({failedPercentage}%)
      </Badge>
      <Badge
        variant="outline"
        className="cursor-pointer rounded-sm font-normal text-sm px-2 py-1 w-fit"
        onClick={() => onClick(WorkflowRunStatus.PENDING)}
      >
        {counts?.PENDING?.toLocaleString('en-US')} Pending ({pendingPercentage}
        %)
      </Badge>
      <Badge
        variant="outline"
        className="cursor-pointer rounded-sm font-normal text-sm px-2 py-1 w-fit"
        onClick={() => onClick(WorkflowRunStatus.QUEUED)}
      >
        {counts?.QUEUED?.toLocaleString('en-US')} Queued ({queuedPercentage}%)
      </Badge>
      {showQueueMetrics && (
        <Badge
          variant="outline"
          className="cursor-pointer rounded-sm font-normal text-sm px-2 py-1 w-fit"
          onClick={() => onViewQueueMetricsClick()}
        >
          Queue metrics
        </Badge>
      )}
    </dl>
  );
};

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
      // eslint-disable-next-line no-case-declarations
      const exhaustivenessCheck: never = status;
      throw new Error(`Unknown status: ${exhaustivenessCheck}`);
  }
}

function MetricBadge({
  metrics,
  status,
  total,
  onClick,
  variant,
  className,
}: {
  metrics: V1TaskRunMetrics;
  status: V1TaskStatus;
  total: number;
  onClick: (status: V1TaskStatus) => void;
  variant: VariantProps<typeof badgeVariants>['variant'];
  className: string;
}) {
  const metric = metrics.find((m) => m.status === status);

  if (!metric) {
    return null;
  }

  const percentage = calculatePercentage(metric.count, total);

  return (
    <Badge
      variant={variant}
      className={className}
      onClick={() => onClick(status)}
    >
      {metric.count.toLocaleString('en-US')} {statusToFriendlyName(status)} (
      {percentage}%)
    </Badge>
  );
}

export const V1WorkflowRunsMetricsView = ({
  metrics,
  showQueueMetrics = false,
  onClick = () => {},
  onViewQueueMetricsClick = () => {},
}: V1TaskRunMetricsProps) => {
  const total = metrics
    .map((m) => m.count)
    .reduce((acc, curr) => acc + curr, 0);

  return (
    <dl className="flex flex-row justify-start gap-6">
      <MetricBadge
        metrics={metrics}
        status={V1TaskStatus.COMPLETED}
        total={total}
        onClick={onClick}
        variant="successful"
        className="cursor-pointer text-sm px-2 py-1 w-fit"
      />

      <MetricBadge
        metrics={metrics}
        status={V1TaskStatus.RUNNING}
        total={total}
        onClick={onClick}
        variant="inProgress"
        className="cursor-pointer text-sm px-2 py-1 w-fit"
      />

      <MetricBadge
        metrics={metrics}
        status={V1TaskStatus.FAILED}
        total={total}
        onClick={onClick}
        variant="failed"
        className="cursor-pointer text-sm px-2 py-1 w-fit"
      />

      <MetricBadge
        metrics={metrics}
        status={V1TaskStatus.QUEUED}
        total={total}
        onClick={onClick}
        variant="outline"
        className="cursor-pointer rounded-sm font-normal text-sm px-2 py-1 w-fit"
      />

      {showQueueMetrics && (
        <Badge
          variant="outline"
          className="cursor-pointer rounded-sm font-normal text-sm px-2 py-1 w-fit"
          onClick={() => onViewQueueMetricsClick()}
        >
          Queue metrics
        </Badge>
      )}
    </dl>
  );
};
