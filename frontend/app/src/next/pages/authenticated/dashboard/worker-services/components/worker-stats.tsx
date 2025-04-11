import { Card, CardContent } from '@/next/components/ui/card';
import { Skeleton } from '@/next/components/ui/skeleton';
import { cn } from '@/next/lib/utils';
import { WorkerStatusBadge } from './worker-status-badge';
import { useFilters } from '@/next/hooks/use-filters';
import { useEffect } from 'react';

interface WorkerStatsProps {
  stats: {
    active: number;
    paused: number;
    inactive: number;
    total: number;
    slots: number;
    maxSlots: number;
  };
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
  slots?: number;
  maxSlots?: number;
  className?: string;
  status?: 'ACTIVE' | 'PAUSED' | 'INACTIVE';
}

const StatCard = ({
  value,
  label,
  colorClass,
  slots,
  maxSlots,
  className,
  status,
}: StatCardProps) => {
  const { setFilter } = useFilters<{ status?: string }>();

  const handleClick = () => {
    if (status) {
      setFilter('status', status.toLowerCase());
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
  const { setFilter } = useFilters<{ status?: string }>();

  useEffect(() => {
    if (!isLoading) {
      if (stats.active === 0 && stats.paused > 0) {
        setFilter('status', 'paused');
      }
    }
  }, [stats.active, stats.paused, isLoading, setFilter]);

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
        value={stats.active}
        label="Active Workers"
        colorClass="text-green-600"
        status="ACTIVE"
      />

      <StatCard
        value={stats.paused}
        label="Paused Workers"
        colorClass="text-yellow-600"
        status="PAUSED"
      />

      <StatCard
        value={stats.inactive}
        label="Inactive Workers"
        colorClass="text-red-600"
        status="INACTIVE"
      />

      <StatCard
        value={
          <div className="flex items-center gap-2">
            <span className="text-2xl font-bold">{stats.slots}</span>
            <div className="text-sm">/ {stats.maxSlots}</div>
            <WorkerStatusBadge
              status={stats.slots === stats.maxSlots ? 'ACTIVE' : 'INACTIVE'}
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
