import {
  WorkersProvider,
  useWorkers,
  Worker,
  WorkerPool,
} from '@/next/hooks/use-workers';
import {
  ManagedComputeProvider,
  useUnifiedWorkerPools,
} from '@/next/hooks/use-managed-compute';
import { Button } from '@/next/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/next/components/ui/select';
import {
  MoreHorizontal,
  ArrowUpRight,
  Cloud,
  Server,
  Zap,
  X,
  Pause,
  Play,
  Plus,
} from 'lucide-react';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/next/components/ui/table';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/next/components/ui/dropdown-menu';
import { Link } from 'react-router-dom';
import { Skeleton } from '@/next/components/ui/skeleton';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/next/components/ui/card';
import { useState } from 'react';
import { SlotsBadge } from './components/worker-slots-badge';
import { WorkerStatusBadge } from './components/worker-status-badge';
import BasicLayout from '@/next/components/layouts/basic.layout';
import { DocsButton } from '@/next/components/ui/docs-button';
import {
  Headline,
  PageTitle,
  HeadlineActions,
  HeadlineActionItem,
} from '@/next/components/ui/page-header';
import docs from '@/next/lib/docs';
import { Separator } from '@/next/components/ui/separator';
import { ROUTES } from '@/next/lib/routes';
import { WorkerType } from '@/lib/api';
import { useCurrentTenantId } from '@/next/hooks/use-tenant';

const WorkerPoolRow = ({ pool }: { pool: WorkerPool }) => {
  const { bulkUpdate } = useWorkers();
  const { tenantId } = useCurrentTenantId();

  const getLastActiveTime = () => {
    const mostRecentWorker = pool.workers
      .filter((worker) => worker.lastHeartbeatAt)
      .sort(
        (a: Worker, b: Worker) =>
          new Date(b.lastHeartbeatAt || '').getTime() -
          new Date(a.lastHeartbeatAt || '').getTime(),
      )[0];

    return mostRecentWorker?.lastHeartbeatAt
      ? new Date(mostRecentWorker.lastHeartbeatAt).toLocaleString()
      : 'Never';
  };

  const handlePauseAllActive = async () => {
    const activeWorkerIds = pool.workers
      .filter((worker) => worker.status === 'ACTIVE')
      .map((worker) => worker.metadata.id);

    if (activeWorkerIds.length > 0) {
      await bulkUpdate.mutateAsync({
        workerIds: activeWorkerIds,
        data: { isPaused: true },
      });
    }
  };

  const handleResumeAllPaused = async () => {
    const pausedWorkerIds = pool.workers
      .filter((worker) => worker.status === 'PAUSED')
      .map((worker) => worker.metadata.id);

    if (pausedWorkerIds.length > 0) {
      await bulkUpdate.mutateAsync({
        workerIds: pausedWorkerIds,
        data: { isPaused: false },
      });
    }
  };

  return (
    <TableRow key={pool.name}>
      <TableCell className="font-medium">
        <Link
          to={ROUTES.workers.poolDetail(
            tenantId,
            encodeURIComponent(pool.id || pool.name),
            pool.type,
          )}
        >
          {pool.name}
        </Link>
      </TableCell>
      <TableCell>
        <div className="flex gap-2">
          <WorkerStatusBadge
            status="ACTIVE"
            count={pool.activeCount}
            variant="outline"
          />
          {pool.pausedCount > 0 && (
            <WorkerStatusBadge
              status="PAUSED"
              count={pool.pausedCount}
              variant="outline"
            />
          )}
        </div>
      </TableCell>
      <TableCell>
        <SlotsBadge
          available={pool.totalAvailableRuns}
          max={pool.totalMaxRuns}
        />
      </TableCell>
      <TableCell>{getLastActiveTime()}</TableCell>
      <TableCell>{pool.type}</TableCell>
      <TableCell className="text-right">
        <div className="flex justify-end">
          <Link
            to={ROUTES.workers.poolDetail(
              tenantId,
              encodeURIComponent(pool.name),
              pool.type,
            )}
          >
            <Button variant="ghost" size="icon">
              <ArrowUpRight className="h-4 w-4" />
            </Button>
          </Link>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon">
                <MoreHorizontal className="h-4 w-4" />
                <span className="sr-only">Open menu</span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuLabel>Actions</DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem>
                <Link
                  to={ROUTES.workers.poolDetail(
                    tenantId,
                    encodeURIComponent(pool.name),
                    pool.type,
                  )}
                  className="w-full"
                >
                  View details
                </Link>
              </DropdownMenuItem>
              <DropdownMenuItem
                disabled={pool.activeCount === 0}
                onClick={handlePauseAllActive}
                className="flex items-center"
              >
                <Pause className="h-4 w-4 mr-2" />
                Pause all active workers
              </DropdownMenuItem>
              <DropdownMenuItem
                disabled={pool.pausedCount === 0}
                onClick={handleResumeAllPaused}
                className="flex items-center"
              >
                <Play className="h-4 w-4 mr-2" />
                Resume all paused workers
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </TableCell>
    </TableRow>
  );
};

const SkeletonRow = () => (
  <TableRow>
    <TableCell>
      <Skeleton className="h-5 w-[180px]" />
    </TableCell>
    <TableCell>
      <div className="flex gap-2">
        <Skeleton className="h-6 w-[90px]" />
        <Skeleton className="h-6 w-[100px]" />
      </div>
    </TableCell>
    <TableCell>
      <Skeleton className="h-6 w-[80px]" />
    </TableCell>
    <TableCell>
      <Skeleton className="h-5 w-[120px]" />
    </TableCell>
    <TableCell className="text-right">
      <div className="flex justify-end gap-2">
        <Skeleton className="h-8 w-8 rounded-md" />
        <Skeleton className="h-8 w-8 rounded-md" />
      </div>
    </TableCell>
  </TableRow>
);

const HatchetCloudCard = ({ onDismiss }: { onDismiss: () => void }) => (
  <Card className="mt-8 relative">
    <Button
      variant="ghost"
      size="icon"
      className="absolute right-2 top-2 h-6 w-6"
      onClick={onDismiss}
      aria-label="Dismiss"
    >
      <X className="h-4 w-4" />
    </Button>
    <CardHeader>
      <div className="flex items-center gap-2">
        <Cloud className="h-5 w-5 text-primary" />
        <CardTitle>Hatchet Cloud</CardTitle>
      </div>
      <CardDescription>
        Managed compute infrastructure for your Hatchet workflows
      </CardDescription>
    </CardHeader>
    <CardContent>
      <div className="flex flex-col space-y-4 md:flex-row md:space-x-6 md:space-y-0">
        <div className="flex items-start space-x-2">
          <Server className="mt-1 h-4 w-4 text-muted-foreground flex-shrink-0" />
          <p className="text-sm text-muted-foreground">
            Fully managed workers with auto-scaling
          </p>
        </div>
        <div className="flex items-start space-x-2">
          <Zap className="mt-1 h-4 w-4 text-muted-foreground flex-shrink-0" />
          <p className="text-sm text-muted-foreground">
            Zero maintenance, high availability, and instant scaling
          </p>
        </div>
      </div>
    </CardContent>
    <CardFooter>
      <DocsButton doc={docs.home.compute} size="lg" />
    </CardFooter>
  </Card>
);

function WorkerContext() {
  const { pools, isLoading } = useUnifiedWorkerPools();

  console.log(isLoading, pools);

  const [showCloudCard, setShowCloudCard] = useState(true);
  const [statusFilter, setStatusFilter] = useState('all');
  const { tenantId } = useCurrentTenantId();

  const handleDismissCard = () => {
    setShowCloudCard(false);
  };

  const handleStatusChange = (status: string) => {
    setStatusFilter(status);
  };

  const renderTableContent = () => {
    if (isLoading) {
      return Array(5)
        .fill(null)
        .map((_, i) => <SkeletonRow key={i} />);
    }

    if (!pools || pools.length === 0) {
      return (
        <TableRow>
          <TableCell colSpan={5} className="text-center">
            No worker pools found
          </TableCell>
        </TableRow>
      );
    }

    return pools.map((pool) => <WorkerPoolRow key={pool.name} pool={pool} />);
  };

  return (
    <BasicLayout>
      <Headline>
        <PageTitle description="Manage your workers and view their statuses and task runs">
          Worker Pools
        </PageTitle>
        <HeadlineActions>
          <HeadlineActionItem>
            <DocsButton doc={docs.home.workers} size="icon" />
          </HeadlineActionItem>
          <HeadlineActionItem>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button>
                  <Plus className="h-4 w-4" />
                  New Worker
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent>
                <DropdownMenuItem asChild>
                  <Link to={ROUTES.workers.new(tenantId, WorkerType.MANAGED)}>
                    Managed Worker
                  </Link>
                </DropdownMenuItem>
                <DropdownMenuItem asChild>
                  <Link
                    to={ROUTES.workers.new(tenantId, WorkerType.SELFHOSTED)}
                  >
                    Self-hosted Worker
                  </Link>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </HeadlineActionItem>
        </HeadlineActions>
      </Headline>

      <Separator className="my-4" />

      <div className="flex flex-col gap-4">
        <div className="flex justify-between items-center">
          <Select value={statusFilter} onValueChange={handleStatusChange}>
            <SelectTrigger className="w-[180px]">
              <SelectValue placeholder="Filter by status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Statuses</SelectItem>
              <SelectItem value="active">Active</SelectItem>
              <SelectItem value="paused">Paused</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Pool</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Slots</TableHead>
              <TableHead>Last Active</TableHead>
              <TableHead>Type</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>{renderTableContent()}</TableBody>
        </Table>

        {showCloudCard && <HatchetCloudCard onDismiss={handleDismissCard} />}
      </div>
    </BasicLayout>
  );
}

export default function WorkerPoolsPage() {
  return (
    <ManagedComputeProvider>
      <WorkersProvider>
        <WorkerContext />
      </WorkersProvider>
    </ManagedComputeProvider>
  );
}
