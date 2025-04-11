import useWorkers from '@/next/hooks/use-workers';
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
import { useEffect, useState } from 'react';
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
import docs from '@/next/docs-meta-data';
import { Separator } from '@/next/components/ui/separator';
import useDefinitions from '@/next/hooks/use-definitions';

// Service row component to simplify the main component
const ServiceRow = ({ service }: { service: any }) => {
  const { bulkUpdate } = useWorkers();

  const getLastActiveTime = () => {
    const mostRecentWorker = service.workers
      .filter((worker: any) => worker.lastHeartbeatAt)
      .sort(
        (a: any, b: any) =>
          new Date(b.lastHeartbeatAt).getTime() -
          new Date(a.lastHeartbeatAt).getTime(),
      )[0];

    return mostRecentWorker?.lastHeartbeatAt
      ? new Date(mostRecentWorker.lastHeartbeatAt).toLocaleString()
      : 'Never';
  };

  // Calculate slots totals
  const totalMaxRuns = service.workers
    .filter((worker: any) => worker.status === 'ACTIVE')
    .reduce((sum: number, worker: any) => sum + (worker.maxRuns || 0), 0);
  const totalAvailableRuns = service.workers
    .filter((worker: any) => worker.status === 'ACTIVE')
    .reduce((sum: number, worker: any) => sum + (worker.availableRuns || 0), 0);

  // Handlers for pause and resume
  const handlePauseAllActive = async () => {
    // Get all active worker IDs for this service
    const activeWorkerIds = service.workers
      .filter((worker: any) => worker.status === 'ACTIVE')
      .map((worker: any) => worker.metadata.id);

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
      .filter((worker: any) => worker.status === 'PAUSED')
      .map((worker: any) => worker.metadata.id);

    if (pausedWorkerIds.length > 0) {
      await bulkUpdate.mutateAsync({
        workerIds: pausedWorkerIds,
        data: { isPaused: false },
      });
    }
  };

  return (
    <TableRow key={service.id}>
      <TableCell className="font-medium">
        <Link to={`/services/${encodeURIComponent(service.name)}`}>
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
        <SlotsBadge available={totalAvailableRuns} max={totalMaxRuns} />
      </TableCell>
      <TableCell>{getLastActiveTime()}</TableCell>
      <TableCell className="text-right">
        <div className="flex justify-end">
          <Link to={`/services/${encodeURIComponent(service.name)}`}>
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
                  to={`/services/${encodeURIComponent(service.name)}`}
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
      <a
        href="https://docs.hatchet.run/home/managed-compute"
        target="_blank"
        rel="noopener noreferrer"
      >
        <Button variant="outline">Learn more about Hatchet Cloud</Button>
      </a>
    </CardFooter>
  </Card>
);

export default function WorkerServicesPage() {
  const [showCloudCard, setShowCloudCard] = useState(true);
  const { data: definitions, slots } = useDefinitions();

  // Check local storage on component mount
  useEffect(() => {
    const isHidden = localStorage.getItem('hideHatchetCloudCard') === 'true';
    setShowCloudCard(!isHidden);
  }, []);

  // Handle dismissal
  const handleDismissCard = () => {
    localStorage.setItem('hideHatchetCloudCard', 'true');
    setShowCloudCard(false);
  };

  const {
    data: workers = [],
    isLoading,
    filters,
    setFilters,
  } = useWorkers({
    refetchInterval: 5000,
  });

  // Process the workers to group by name (service)
  const workerServices = workers.reduce(
    (services: Record<string, any>, worker) => {
      const serviceName = worker.name;
      if (!services[serviceName]) {
        services[serviceName] = {
          id: serviceName,
          name: serviceName,
          workersCount: 0,
          activeCount: 0,
          inactiveCount: 0,
          pausedCount: 0,
          workers: [],
        };
      }

      services[serviceName].workersCount++;
      if (worker.status === 'ACTIVE') {
        services[serviceName].activeCount++;
      } else if (worker.status === 'INACTIVE') {
        services[serviceName].inactiveCount++;
      } else if (worker.status === 'PAUSED') {
        services[serviceName].pausedCount++;
      }

      services[serviceName].workers.push(worker);
      return services;
    },
    {},
  );

  // Convert to array for the table
  const servicesData = Object.values(workerServices);

  // Handle status filter change
  const handleStatusChange = (status: string) => {
    setFilters({
      ...filters,
      status:
        status === 'ALL'
          ? undefined
          : (status as 'ACTIVE' | 'INACTIVE' | 'PAUSED'),
    });
  };

  const renderTableContent = () => {
    if (isLoading) {
      return Array(5)
        .fill(0)
        .map((_, index) => <SkeletonRow key={`skeleton-${index}`} />);
    }

    if (servicesData.length === 0) {
      return (
        <TableRow>
          <TableCell colSpan={5} className="h-24">
            <div className="flex flex-col items-center justify-center gap-4 p-6">
              <p className="text-muted-foreground">No worker services found.</p>
              <DocsButton doc={docs.home.workers} size="lg" />
            </div>
          </TableCell>
        </TableRow>
      );
    }

    return servicesData.map((service: any) => (
      <ServiceRow key={service.id} service={service} />
    ));
  };

  return (
    <BasicLayout>
      <Headline>
        <PageTitle description="Manage your worker services">
          Worker Services
        </PageTitle>
        <HeadlineActions>
          <HeadlineActionItem>
            <DocsButton doc={docs.home.workers} size="icon" />
          </HeadlineActionItem>
        </HeadlineActions>
      </Headline>
      <Separator className="my-4" />
      <pre>{JSON.stringify(definitions, null, 2)}</pre>
      <pre>{JSON.stringify(slots, null, 2)}</pre>
      <div className="mb-6">
        <div className="flex gap-4 mb-6">
          <Select
            value={filters.status || 'ALL'}
            onValueChange={handleStatusChange}
          >
            <SelectTrigger className="w-[180px]">
              <SelectValue placeholder="Filter by status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="ALL">All Statuses</SelectItem>
              <SelectItem value="ACTIVE">Active</SelectItem>
              <SelectItem value="INACTIVE">Inactive</SelectItem>
              <SelectItem value="PAUSED">Paused</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Service Name</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Total Slots</TableHead>
                <TableHead>Last Active</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>{renderTableContent()}</TableBody>
          </Table>
        </div>
        {showCloudCard && <HatchetCloudCard onDismiss={handleDismissCard} />}
      </div>
    </BasicLayout>
  );
}
