import {
  computeWorkerUtilization,
  statusCount,
  sumQueueMetrics,
} from './dashboard-metrics';
import { PanelCard } from './panel-card';
import { Skeleton } from '@/components/v1/ui/skeleton';
import { queries, V1TaskStatus, WorkerStatus } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { type ReactNode } from 'react';
import { RiBarChart2Line } from 'react-icons/ri';

const REFETCH_MS = 20000;

function StatCell({
  label,
  hint,
  loading,
  error,
  children,
}: {
  label: string;
  hint?: string;
  loading: boolean;
  // A failed request must not render as an authoritative-looking zero.
  error: boolean;
  children: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1 p-4">
      <span className="font-mono text-xs uppercase tracking-wider text-muted-foreground">
        {label}
      </span>
      {loading ? (
        <Skeleton className="h-8 w-16" />
      ) : (
        <span className="text-2xl font-semibold tabular-nums">
          {error ? 'n/a' : children}
        </span>
      )}
      {hint && <span className="text-xs text-muted-foreground/70">{hint}</span>}
    </div>
  );
}

export function StatsPanel({
  tenantId,
  since,
}: {
  tenantId: string;
  since: string;
}) {
  const statusQuery = useQuery({
    ...queries.v1TaskRuns.metrics(tenantId, {
      since,
      workflow_ids: [],
    }),
    refetchInterval: REFETCH_MS,
    placeholderData: (prev) => prev,
  });

  const workersQuery = useQuery({
    ...queries.workers.list(tenantId),
    refetchInterval: REFETCH_MS,
    placeholderData: (prev) => prev,
  });

  const queueQuery = useQuery({
    ...queries.metrics.getStepRunQueueMetrics(tenantId),
    refetchInterval: REFETCH_MS,
    placeholderData: (prev) => prev,
  });

  const metrics = statusQuery.data;
  const workers = workersQuery.data?.rows ?? [];

  const utilization = computeWorkerUtilization(workers);
  const queued = sumQueueMetrics(queueQuery.data?.queues);
  const succeeded = statusCount(metrics, V1TaskStatus.COMPLETED);
  const running = statusCount(metrics, V1TaskStatus.RUNNING);
  const failed = statusCount(metrics, V1TaskStatus.FAILED);
  const workersConnected = workers.filter(
    (worker) => worker.status === WorkerStatus.ACTIVE,
  ).length;

  const fmt = (n: number) => n.toLocaleString('en-US');

  return (
    <PanelCard
      icon={<RiBarChart2Line className="size-4" />}
      title="Stats"
      bodyClassName="p-0"
    >
      <div className="grid grid-cols-2 divide-x divide-y divide-border/50 md:grid-cols-3 lg:grid-cols-6 lg:divide-y-0">
        <StatCell
          label="Worker utilization"
          hint="Occupied slots across active workers."
          loading={workersQuery.isLoading}
          error={workersQuery.isError}
        >
          {`${Math.round(utilization * 100)}%`}
        </StatCell>
        {/* Queue depth is a live total with no time window; the three status
            counts beside it come from the 24h run metrics, so each label says
            which one it is. */}
        <StatCell
          label="Queued tasks (total)"
          loading={queueQuery.isLoading}
          error={queueQuery.isError}
        >
          {fmt(queued)}
        </StatCell>
        <StatCell
          label="Succeeded (24h)"
          loading={statusQuery.isLoading}
          error={statusQuery.isError}
        >
          {fmt(succeeded)}
        </StatCell>
        <StatCell
          label="Running (24h)"
          loading={statusQuery.isLoading}
          error={statusQuery.isError}
        >
          {fmt(running)}
        </StatCell>
        <StatCell
          label="Failed (24h)"
          loading={statusQuery.isLoading}
          error={statusQuery.isError}
        >
          {fmt(failed)}
        </StatCell>
        <StatCell
          label="Workers connected"
          loading={workersQuery.isLoading}
          error={workersQuery.isError}
        >
          {fmt(workersConnected)}
        </StatCell>
      </div>
    </PanelCard>
  );
}
