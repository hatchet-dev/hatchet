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

interface V1TaskRunMetricsProps {
  onClick?: (status?: V1TaskStatus) => void;
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
  onClick = () => {},
}: V1TaskRunMetricsProps) => {
  const { metrics, queueMetrics } = useRuns();

  const [isQueueMetricsOpen, setIsQueueMetricsOpen] = useState(false);
  const total = metrics.data
    .map((m) => m.count)
    .reduce((acc, curr) => acc + curr, 0);

  return (
    <>
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

        <RunsBadge
          status="QueueMetrics"
          className="cursor-pointer rounded-sm font-normal text-sm px-2 py-1 w-fit"
          onClick={() => setIsQueueMetricsOpen(true)}
          isLoading={metrics.isLoading}
        >
          Queue metrics
        </RunsBadge>
      </dl>

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
    </>
  );
};
