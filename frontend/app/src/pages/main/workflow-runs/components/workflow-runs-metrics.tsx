import React from 'react';

import { WorkflowRunStatus, WorkflowRunsMetrics } from '@/lib/api';
import { Badge } from '@/components/ui/badge';

interface WorkflowRunsMetricsProps {
  metrics: WorkflowRunsMetrics;
  onClick?: (status?: WorkflowRunStatus) => void;
  onViewQueueMetricsClick?: () => void;
  showQueueMetrics?: boolean;
}

export const WorkflowRunsMetricsView: React.FC<WorkflowRunsMetricsProps> = ({
  metrics: { counts },
  showQueueMetrics = false,
  onClick = () => {},
  onViewQueueMetricsClick = () => {},
}) => {
  return (
    <dl className="flex flex-row justify-start gap-6">
      <Badge
        variant="successful"
        className="cursor-pointer text-sm px-2 py-1 w-fit"
        onClick={() => onClick(WorkflowRunStatus.SUCCEEDED)}
      >
        {counts?.SUCCEEDED?.toLocaleString('en-US')} Succeeded
      </Badge>
      <Badge
        variant="inProgress"
        className="cursor-pointer text-sm px-2 py-1 w-fit"
        onClick={() => onClick(WorkflowRunStatus.RUNNING)}
      >
        {counts?.RUNNING?.toLocaleString('en-US')} Running
      </Badge>
      <Badge
        variant="failed"
        className="cursor-pointer text-sm px-2 py-1 w-fit"
        onClick={() => onClick(WorkflowRunStatus.FAILED)}
      >
        {counts?.FAILED?.toLocaleString('en-US')} Failed
      </Badge>
      <Badge
        variant="outlineDestructive"
        className="cursor-pointer text-sm px-2 py-1 w-fit"
        onClick={() => onClick(WorkflowRunStatus.CANCELLED)}
      >
        {counts?.CANCELLED?.toLocaleString('en-US')} Cancelled
      </Badge>
      <Badge
        variant="outline"
        className="cursor-pointer rounded-sm font-normal text-sm px-2 py-1 w-fit"
        onClick={() => onClick(WorkflowRunStatus.PENDING)}
      >
        {counts?.PENDING?.toLocaleString('en-US')} Pending
      </Badge>
      <Badge
        variant="outline"
        className="cursor-pointer rounded-sm font-normal text-sm px-2 py-1 w-fit"
        onClick={() => onClick(WorkflowRunStatus.QUEUED)}
      >
        {counts?.QUEUED?.toLocaleString('en-US')} Queued
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
