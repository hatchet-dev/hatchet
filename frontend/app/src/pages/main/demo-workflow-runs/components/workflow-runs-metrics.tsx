import React from 'react';

import { WorkflowRunsMetrics } from '@/lib/api';
import { Badge } from '@/components/ui/badge';

interface WorkflowRunsMetricsProps {
  metrics: WorkflowRunsMetrics;
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
        className="cursor-default text-sm px-2 py-1 w-fit"
      >
        {counts?.SUCCEEDED} Succeeded ({succeededPercentage}%)
      </Badge>
      <Badge
        variant="inProgress"
        className="cursor-default text-sm px-2 py-1 w-fit"
      >
        {counts?.RUNNING} Running ({runningPercentage}%)
      </Badge>
      <Badge
        variant="failed"
        className="cursor-default text-sm px-2 py-1 w-fit"
      >
        {counts?.FAILED} Failed ({failedPercentage}%)
      </Badge>
      <Badge
        variant="outline"
        className="cursor-default rounded-sm font-normal text-sm px-2 py-1 w-fit"
      >
        {counts?.PENDING} Pending ({pendingPercentage}%)
      </Badge>
      <Badge
        variant="outline"
        className="cursor-default rounded-sm font-normal text-sm px-2 py-1 w-fit"
      >
        {counts?.QUEUED} Queued ({queuedPercentage}%)
      </Badge>
    </dl>
  );
};
