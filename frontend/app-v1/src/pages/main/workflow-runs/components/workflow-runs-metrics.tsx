import React from 'react';

import { WorkflowRunStatus, WorkflowRunsMetrics } from '@/lib/api';
import { Badge } from '@/components/ui/badge';

interface WorkflowRunsMetricsProps {
  metrics: WorkflowRunsMetrics;
  onClick?: (status?: WorkflowRunStatus) => void;
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
    (counts?.FAILED ?? 0) +
    (counts?.CANCELLED ?? 0);

  const succeededPercentage = calculatePercentage(
    counts?.SUCCEEDED ?? 0,
    total,
  );
  const runningPercentage = calculatePercentage(counts?.RUNNING ?? 0, total);
  const failedPercentage = calculatePercentage(counts?.FAILED ?? 0, total);
  const pendingPercentage = calculatePercentage(counts?.PENDING ?? 0, total);
  const queuedPercentage = calculatePercentage(counts?.QUEUED ?? 0, total);
  const cancelledPercentage = calculatePercentage(
    counts?.CANCELLED ?? 0,
    total,
  );
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
        variant="outlineDestructive"
        className="cursor-pointer text-sm px-2 py-1 w-fit"
        onClick={() => onClick(WorkflowRunStatus.CANCELLED)}
      >
        {counts?.CANCELLED?.toLocaleString('en-US')} Cancelled (
        {cancelledPercentage}%)
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
