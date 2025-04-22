import { V1TaskRunMetrics, V1TaskStatus } from '@/lib/api';
import { RunsBadge } from '../runs-badge';
import { percent } from '@/next/lib/utils/percent';
import { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/next/components/ui/dialog';
import { useRuns } from '@/next/hooks/use-runs';
import { cn } from '@/next/lib/utils';

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

export const RunsMetricsView = () => {
  const {
    metrics,
    queueMetrics,
    filters: { filters, setFilter },
  } = useRuns();

  const [isQueueMetricsOpen, setIsQueueMetricsOpen] = useState(false);
  const total = metrics.data
    .map((m) => m.count)
    .reduce((acc, curr) => acc + curr, 0);

  const handleMetricClick = (status?: V1TaskStatus) => {
    if (status) {
      // Toggle the filter - if it's already set to this status, clear it
      const currentStatuses = filters.statuses || [];
      if (currentStatuses.includes(status)) {
        setFilter('statuses', undefined);
      } else {
        setFilter('statuses', [status]);
      }
    }
  };

  const isMetricActive = (status: V1TaskStatus) => {
    return !(
      filters.statuses &&
      filters.statuses.length > 0 &&
      !filters.statuses.includes(status)
    );
  };

  return (
    <div className="flex flex-row justify-between gap-6">
      <dl className="flex flex-row justify-start gap-6">
        <MetricBadge
          metrics={metrics.data}
          status={V1TaskStatus.COMPLETED}
          total={total}
          onClick={handleMetricClick}
          className={cn(
            'cursor-pointer text-sm px-2 py-1 w-fit',
            !isMetricActive(V1TaskStatus.COMPLETED) && 'opacity-50',
          )}
          isLoading={metrics.isLoading}
        />

        <MetricBadge
          metrics={metrics.data}
          status={V1TaskStatus.RUNNING}
          total={total}
          onClick={handleMetricClick}
          className={cn(
            'cursor-pointer text-sm px-2 py-1 w-fit',
            !isMetricActive(V1TaskStatus.RUNNING) && 'opacity-50',
          )}
          isLoading={metrics.isLoading}
        />

        <MetricBadge
          metrics={metrics.data}
          status={V1TaskStatus.FAILED}
          total={total}
          onClick={handleMetricClick}
          className={cn(
            'cursor-pointer text-sm px-2 py-1 w-fit',
            !isMetricActive(V1TaskStatus.FAILED) && 'opacity-50',
          )}
          isLoading={metrics.isLoading}
        />

        <MetricBadge
          metrics={metrics.data}
          status={V1TaskStatus.QUEUED}
          total={total}
          onClick={handleMetricClick}
          className={cn(
            'cursor-pointer rounded-sm font-normal text-sm px-2 py-1 w-fit',
            !isMetricActive(V1TaskStatus.QUEUED) && 'opacity-50',
          )}
          isLoading={metrics.isLoading}
        />
      </dl>

      <RunsBadge
        status="QueueMetrics"
        className="cursor-pointer rounded-sm font-normal text-sm px-2 py-1 w-fit"
        onClick={() => setIsQueueMetricsOpen(true)}
        isLoading={metrics.isLoading}
      >
        Queue metrics
      </RunsBadge>

      <Dialog open={isQueueMetricsOpen} onOpenChange={setIsQueueMetricsOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Current Queue Status</DialogTitle>
            <DialogDescription>
              Detailed information about queued tasks
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="space-y-2">
              <pre className="text-sm text-muted-foreground">
                {JSON.stringify(queueMetrics.data?.queues, null, 2)}
              </pre>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
};
