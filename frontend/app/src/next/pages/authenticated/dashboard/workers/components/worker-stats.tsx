import { Card, CardContent } from '@/next/components/ui/card';
import { Skeleton } from '@/next/components/ui/skeleton';
import { cn } from '@/next/lib/utils';
import { WorkerStatusBadge } from './worker-status-badge';
import { useEffect } from 'react';
import { useWorkers, WorkerPool } from '@/next/hooks/use-workers';

interface WorkerStatsProps {
  stats: WorkerPool;
  isLoading: boolean;
}

// Skeleton loader for stat cards
export const StatCardSkeleton = () => (
  <Card>
    <CardContent className="pt-6">
      <Skeleton className="h-8 w-16 mb-2" />
      <Skeleton className="h-4 w-32" />
    </CardContent>
  </Card>
);

interface StatCardProps {
  value: number | React.ReactNode;
  label: string;
  colorClass: string;
  className?: string;
  status?: 'ACTIVE' | 'PAUSED' | 'INACTIVE';
}

const StatCard = ({
  value,
  label,
  colorClass,
  className,
  status,
}: StatCardProps) => {
  const {
    filters: { setFilter },
  } = useWorkers();

  const handleClick = () => {
    if (status) {
      setFilter('status', status);
    }
  };

  return (
    <Card
      className={cn(
        'transition-all cursor-pointer hover:bg-muted/50',
        className,
      )}
      onClick={handleClick}
    >
      <CardContent className="pt-6">
        <div className={cn('text-2xl font-bold', colorClass)}>{value}</div>
        <p className="text-sm text-muted-foreground">{label}</p>
      </CardContent>
    </Card>
  );
};

export function WorkerStats({ stats, isLoading }: WorkerStatsProps) {
  const {
    filters: { setFilter },
  } = useWorkers();

  useEffect(() => {
    if (!isLoading) {
      if (stats.activeCount === 0 && stats.pausedCount > 0) {
        setFilter('status', 'paused');
      }
    }
  }, [stats.activeCount, stats.pausedCount, isLoading, setFilter]);

  if (isLoading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <StatCardSkeleton />
        <StatCardSkeleton />
        <StatCardSkeleton />
        <StatCardSkeleton />
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
      <StatCard
        value={stats.activeCount}
        label="Active Workers"
        colorClass="text-green-600"
        status="ACTIVE"
      />

      <StatCard
        value={stats.pausedCount}
        label="Paused Workers"
        colorClass="text-yellow-600"
        status="PAUSED"
      />

      <StatCard
        value={stats.inactiveCount}
        label="Inactive Workers"
        colorClass="text-red-600"
        status="INACTIVE"
      />

      <StatCard
        value={
          <div className="flex items-center gap-2">
            <span className="text-2xl font-bold">
              {stats.totalAvailableRuns}
            </span>
            <div className="text-sm">/ {stats.totalMaxRuns}</div>
            <WorkerStatusBadge
              status={
                stats.totalAvailableRuns === stats.totalMaxRuns
                  ? 'ACTIVE'
                  : 'INACTIVE'
              }
              variant="xs"
            />
          </div>
        }
        label="Total Slots"
        colorClass=""
      />
    </div>
  );
}
