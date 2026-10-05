import { statusCount, totalTaskCount } from './dashboard-metrics';
import { PanelCard, PanelState } from './panel-card';
import {
  DataPoint,
  ZoomableChart,
} from '@/components/v1/molecules/charts/zoomable';
import { Button } from '@/components/v1/ui/button';
import { queries, V1TaskStatus } from '@/lib/api';
import { cn } from '@/lib/utils';
import { appRoutes } from '@/router';
import { useQuery } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import { ArrowUpRight } from 'lucide-react';
import { RiPlayLargeLine } from 'react-icons/ri';

const REFETCH_MS = 20000;

type LegendItem = {
  label: string;
  status: V1TaskStatus;
  dotClassName: string;
};

const LEGEND: LegendItem[] = [
  {
    label: 'Succeeded',
    status: V1TaskStatus.COMPLETED,
    dotClassName: 'bg-green-500',
  },
  {
    label: 'Running',
    status: V1TaskStatus.RUNNING,
    dotClassName: 'bg-yellow-500',
  },
  { label: 'Failed', status: V1TaskStatus.FAILED, dotClassName: 'bg-red-500' },
  {
    label: 'Cancelled',
    status: V1TaskStatus.CANCELLED,
    dotClassName: 'bg-orange-500',
  },
  {
    label: 'Queued',
    status: V1TaskStatus.QUEUED,
    dotClassName: 'bg-slate-400',
  },
];

export function TasksPanel({
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

  const pointQuery = useQuery({
    ...queries.v1TaskRuns.pointMetrics(tenantId, {
      createdAfter: since,
    }),
    refetchInterval: REFETCH_MS,
    placeholderData: (prev) => prev,
  });

  const metrics = statusQuery.data;
  const total = totalTaskCount(metrics);

  const chartData: DataPoint<'SUCCEEDED' | 'FAILED'>[] = (
    pointQuery.data?.results ?? []
  ).map((result) => ({
    date: result.time,
    SUCCEEDED: result.SUCCEEDED,
    FAILED: result.FAILED,
  }));

  return (
    <PanelCard
      icon={<RiPlayLargeLine className="size-4" />}
      title="Runs"
      subtitle="Last 24 hours"
      action={
        <Button variant="ghost" size="sm" className="h-auto p-1" asChild>
          <Link
            to={appRoutes.tenantRunsRoute.to}
            params={{ tenant: tenantId }}
            className="flex items-center gap-1 text-xs text-muted-foreground"
          >
            Open runs
            <ArrowUpRight className="size-3" />
          </Link>
        </Button>
      }
    >
      <PanelState
        loading={statusQuery.isLoading}
        error={statusQuery.isError}
        onRetry={() => statusQuery.refetch()}
        isEmpty={total === 0}
        emptyText="No task runs yet in the last 24 hours."
      >
        <div className="space-y-4">
          <div className="text-3xl font-semibold tabular-nums">
            {total.toLocaleString('en-US')}
          </div>

          <div className="h-24">
            {chartData.length > 0 && (
              <ZoomableChart
                kind="bar"
                data={chartData}
                colors={{
                  SUCCEEDED: 'rgb(34 197 94 / 0.5)',
                  FAILED: 'hsl(var(--destructive))',
                }}
                showYAxis={false}
                className="h-24 min-h-24"
              />
            )}
          </div>

          <div className="flex flex-wrap gap-x-4 gap-y-1">
            {LEGEND.map((item) => (
              <div
                key={item.status}
                className="flex items-center gap-1.5 text-xs text-muted-foreground"
              >
                <span
                  className={cn('size-2 rounded-full', item.dotClassName)}
                />
                <span>{item.label}</span>
                <span className="font-medium tabular-nums text-foreground">
                  {statusCount(metrics, item.status).toLocaleString('en-US')}
                </span>
              </div>
            ))}
          </div>
        </div>
      </PanelState>
    </PanelCard>
  );
}
