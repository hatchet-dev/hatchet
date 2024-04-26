import React from 'react';

import { WorkflowRunsMetrics } from '@/lib/api';

interface WorkflowRunsMetricsProps {
  metrics: WorkflowRunsMetrics;
}

const calculatePercentage = (value: number, total: number): number => {
  return Math.round((value / total) * 100);
};

export const WorkflowRunsMetricsView: React.FC<WorkflowRunsMetricsProps> = ({
  metrics: { counts },
}) => {
  const total =
    (counts?.PENDING ?? 0) +
    (counts?.RUNNING ?? 0) +
    (counts?.SUCCEEDED ?? 0) +
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
    <div className="flex flex-row space-x-4 dark:text-white">
      <div className="flex items-center">
        <div className="w-12 h-12 bg-green-500 rounded-full flex items-center justify-center">
          <span className="text-white text-sm font-bold">
            {succeededPercentage}%
          </span>
        </div>
        <p className="ml-2">Succeeded: {counts?.SUCCEEDED}</p>
      </div>
      <div className="flex items-center">
        <div className="w-12 h-12 bg-green-300 rounded-full flex items-center justify-center">
          <span className="text-white text-sm font-bold">
            {runningPercentage}%
          </span>
        </div>
        <p className="ml-2">Running: {counts?.RUNNING}</p>
      </div>
      <div className="flex items-center">
        <div className="w-12 h-12 bg-red-500 rounded-full flex items-center justify-center">
          <span className="text-white text-sm font-bold">
            {failedPercentage}%
          </span>
        </div>
        <p className="ml-2">Failed: {counts?.FAILED}</p>
      </div>
      <div className="flex items-center">
        <div className="w-12 h-12 bg-gray-400 rounded-full flex items-center justify-center">
          <span className="text-white text-sm font-bold">
            {pendingPercentage}%
          </span>
        </div>
        <p className="ml-2">Pending: {counts?.PENDING}</p>
      </div>
      <div className="flex items-center">
        <div className="w-12 h-12 bg-gray-400 rounded-full flex items-center justify-center">
          <span className="text-white text-sm font-bold">
            {queuedPercentage}%
          </span>
        </div>
        <p className="ml-2">Queued: {counts?.QUEUED}</p>
      </div>
    </div>
  );
};
