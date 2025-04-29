import {
  WorkersProvider,
  useWorkers,
  Worker,
  WorkerService,
} from '@/next/hooks/use-workers';
import {
  ManagedComputeProvider,
  useUnifiedWorkerServices,
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

// Service row component to simplify the main component
const ServiceRow = ({ service }: { service: WorkerService }) => {
  const { bulkUpdate } = useWorkers();

  const getLastActiveTime = () => {
    const mostRecentWorker = service.workers
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

  // Handlers for pause and resume
  const handlePauseAllActive = async () => {
    // Get all active worker IDs for this service
    const activeWorkerIds = service.workers
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
    // Get all paused worker IDs for this service
    const pausedWorkerIds = service.workers
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
    <TableRow key={service.name}>
      <TableCell className="font-medium">
        <Link
          to={ROUTES.services.detail(
            encodeURIComponent(service.id || service.name),
            service.type,
          )}
        >
          {service.name}
        </Link>
      </TableCell>
      <TableCell>
        <div className="flex gap-2">
          <WorkerStatusBadge
            status="ACTIVE"
            count={service.activeCount}
            variant="outline"
          />
          {service.pausedCount > 0 && (
            <WorkerStatusBadge
              status="PAUSED"
              count={service.pausedCount}
              variant="outline"
            />
          )}
        </div>
      </TableCell>
      <TableCell>
        <SlotsBadge
          available={service.totalAvailableRuns}
          max={service.totalMaxRuns}
        />
      </TableCell>
      <TableCell>{getLastActiveTime()}</TableCell>
      <TableCell>{service.type}</TableCell>
      <TableCell className="text-right">
        <div className="flex justify-end">
          <Link
            to={ROUTES.services.detail(
              encodeURIComponent(service.name),
              service.type,
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
                  to={ROUTES.services.detail(
                    encodeURIComponent(service.name),
                    service.type,
                  )}
                  className="w-full"
                >
                  View details
                </Link>
              </DropdownMenuItem>
              <DropdownMenuItem
                disabled={service.activeCount === 0}
                onClick={handlePauseAllActive}
                className="flex items-center"
              >
                <Pause className="h-4 w-4 mr-2" />
                Pause all active workers
              </DropdownMenuItem>
              <DropdownMenuItem
                disabled={service.pausedCount === 0}
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

// Skeleton row component for loading state
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

// Hatchet Cloud advertisement component
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
            Fully managed worker services with auto-scaling
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

function WorkerServicesContent() {
  const { isLoading } = useWorkers();
  const unifiedServices = useUnifiedWorkerServices();
  const [showCloudCard, setShowCloudCard] = useState(true);
  const [statusFilter, setStatusFilter] = useState('all');

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

    if (!unifiedServices || unifiedServices.length === 0) {
      return (
        <TableRow>
          <TableCell colSpan={5} className="text-center">
            No worker services found
          </TableCell>
        </TableRow>
      );
    }

    return unifiedServices.map((service) => (
      <ServiceRow key={service.name} service={service} />
    ));
  };

  return (
    <BasicLayout>
      <Headline>
        <PageTitle description="Manage your worker services and view their status">
          Worker Services
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
                  New Service
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent>
                <DropdownMenuItem asChild>
                  <Link to={ROUTES.services.new(WorkerType.MANAGED)}>
                    Managed Service
                  </Link>
                </DropdownMenuItem>
                <DropdownMenuItem asChild>
                  <Link to={ROUTES.services.new(WorkerType.SELFHOSTED)}>
                    Self-hosted Service
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
              <TableHead>Service</TableHead>
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

export default function WorkerServicesPage() {
  return (
    <ManagedComputeProvider>
      <WorkersProvider>
        <WorkerServicesContent />
      </WorkersProvider>
    </ManagedComputeProvider>
  );
}
