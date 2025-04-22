import { useState, useEffect, useMemo } from 'react';
import { Button } from '@/next/components/ui/button';
import { Skeleton } from '@/next/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/next/components/ui/table';

import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/next/components/ui/tooltip';
import { Play, Pause } from 'lucide-react';
import { WorkerStatusBadge } from './worker-status-badge';
import { SlotsBadge } from './worker-slots-badge';
import { WorkerId } from './worker-id';
import { Time } from '@/next/components/ui/time';
import { useWorkers } from '@/next/hooks/use-workers';
import {
  FilterGroup,
  FilterSelect,
} from '@/next/components/ui/filters/filters';
import { useNavigate } from 'react-router-dom';
import { ROUTES } from '@/next/lib/routes';

interface WorkerTableProps {
  serviceName: string;
}

interface WorkerFilters {
  status?: 'all' | 'active' | 'paused' | 'inactive';
}

// Skeleton for table row
export const TableRowSkeleton = () => (
  <TableRow>
    <TableCell>
      <Skeleton className="h-4 w-4 mx-auto" />
    </TableCell>
    <TableCell>
      <Skeleton className="h-4 w-24" />
    </TableCell>
    <TableCell>
      <Skeleton className="h-6 w-16" />
    </TableCell>
    <TableCell>
      <Skeleton className="h-6 w-20" />
    </TableCell>
    <TableCell>
      <Skeleton className="h-4 w-32" />
    </TableCell>
    <TableCell>
      <div className="flex justify-end gap-2">
        <Skeleton className="h-8 w-8" />
        <Skeleton className="h-8 w-8" />
      </div>
    </TableCell>
  </TableRow>
);

export function WorkerTable({ serviceName }: WorkerTableProps) {
  const [selectedWorkers, setSelectedWorkers] = useState<string[]>([]);
  const navigate = useNavigate();

  const {
    services,
    isLoading,
    update,
    filters: { filters, setFilter },
  } = useWorkers();

  const filterStatus = useMemo(() => filters.status || 'ALL', [filters]);

  // Filter workers for this service
  const service = services.find((worker) => worker.name === serviceName);

  // Set filter to paused if there are no active workers but there are paused workers
  useEffect(() => {
    if (!isLoading) {
      if (
        (service?.activeCount === 0 && filterStatus === 'ACTIVE') ||
        (service?.pausedCount === 0 && filterStatus === 'PAUSED')
      ) {
        setFilter('status', 'all');
      }
    }
  }, [
    service?.activeCount,
    service?.pausedCount,
    isLoading,
    filterStatus,
    setFilter,
  ]);

  // Filter workers based on selected status
  const filteredWorkers = service?.workers.filter((worker) => {
    if (filterStatus === 'ALL') {
      return true;
    }
    if (filterStatus === 'ACTIVE' && worker.status === 'ACTIVE') {
      return true;
    }
    if (filterStatus === 'PAUSED' && worker.status === 'PAUSED') {
      return true;
    }
    if (filterStatus === 'INACTIVE' && worker.status === 'INACTIVE') {
      return true;
    }
    return false;
  });

  const toggleSelectWorker = (workerId: string) => {
    setSelectedWorkers((prev) =>
      prev.includes(workerId)
        ? prev.filter((id) => id !== workerId)
        : [...prev, workerId],
    );
  };

  const selectAllWorkers = () => {
    const allWorkerIds = filteredWorkers?.map((worker) => worker.metadata.id);
    setSelectedWorkers(allWorkerIds || []);
  };

  const clearSelection = () => {
    setSelectedWorkers([]);
  };

  const handleWorkerAction = async (
    workerId: string,
    action: 'pause' | 'resume' | 'stop',
  ) => {
    try {
      await update.mutateAsync({
        workerId,
        data: { isPaused: action !== 'resume' },
      });
    } catch (error) {
      console.error(`Failed to ${action} worker:`, error);
    }
  };

  const handleResumeWorker = (workerId: string) =>
    handleWorkerAction(workerId, 'resume');
  const handlePauseWorker = (workerId: string) =>
    handleWorkerAction(workerId, 'pause');

  const handleBulkAction = async (action: 'pause' | 'resume' | 'stop') => {
    const isPaused = action === 'resume' ? false : true;

    for (const workerId of selectedWorkers) {
      try {
        await update.mutateAsync({
          workerId,
          data: { isPaused },
        });
      } catch (error) {
        console.error(`Failed to ${action} worker ${workerId}:`, error);
      }
    }

    // Clear selection after bulk action
    setSelectedWorkers([]);
  };

  const handleWorkerClick = (workerId: string) => {
    navigate(
      ROUTES.services.selfhostedWorkerDetail(encodeURIComponent(serviceName), workerId),
    );
  };

  const statusOptions = [
    { label: 'All Workers', value: 'all', count: service?.workers.length },
    { label: 'Active', value: 'active', count: service?.activeCount },
    { label: 'Paused', value: 'paused', count: service?.pausedCount },
    { label: 'Inactive', value: 'inactive', count: service?.inactiveCount },
  ];

  return (
    <div className="space-y-4">
      <div className="flex flex-col space-y-4 md:flex-row md:items-center md:justify-between md:space-y-0">
        {/* Worker Filters */}
        <FilterGroup>
          <FilterSelect<WorkerFilters, string>
            name="status"
            options={statusOptions.map(({ label, value, count }) => ({
              label: `${label} (${count})`,
              value,
            }))}
            placeholder="Filter by status"
          />
        </FilterGroup>

        {/* Bulk Actions */}
        {selectedWorkers.length > 0 ? (
          <div className="flex items-center gap-2">
            <Button
              size="sm"
              variant="outline"
              onClick={() => handleBulkAction('resume')}
            >
              <Play className="h-4 w-4 mr-1" />
              Resume Selected
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={() => handleBulkAction('pause')}
            >
              <Pause className="h-4 w-4 mr-1" />
              Pause Selected
            </Button>
            <Button size="sm" variant="ghost" onClick={clearSelection}>
              Clear Selection ({selectedWorkers.length})
            </Button>
          </div>
        ) : (
          <Button size="sm" variant="outline" onClick={selectAllWorkers}>
            Select All
          </Button>
        )}
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-12">
                <div className="flex items-center justify-center">
                  <input
                    type="checkbox"
                    checked={
                      selectedWorkers.length > 0 &&
                      selectedWorkers.length === filteredWorkers?.length
                    }
                    onChange={
                      selectedWorkers.length === filteredWorkers?.length
                        ? clearSelection
                        : selectAllWorkers
                    }
                    className="h-4 w-4"
                  />
                </div>
              </TableHead>
              <TableHead>ID</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Slots</TableHead>
              <TableHead>Type</TableHead>
              <TableHead>Last Heartbeat</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              // Skeleton loading state
              Array(5)
                .fill(0)
                .map((_, index) => <TableRowSkeleton key={index} />)
            ) : filteredWorkers?.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="h-24 text-center">
                  {filterStatus === 'ALL'
                    ? 'No workers in this service.'
                    : `No ${filterStatus} workers in this service.`}
                </TableCell>
              </TableRow>
            ) : (
              filteredWorkers?.map((worker) => (
                <TableRow key={worker.metadata.id}>
                  <TableCell>
                    <div className="flex items-center justify-center">
                      <input
                        type="checkbox"
                        checked={selectedWorkers.includes(worker.metadata.id)}
                        onChange={() => toggleSelectWorker(worker.metadata.id)}
                        className="h-4 w-4"
                      />
                    </div>
                  </TableCell>
                  <TableCell className="font-medium">
                    <button
                      onClick={() => handleWorkerClick(worker.metadata.id)}
                      className="hover:underline text-left"
                    >
                      <WorkerId worker={worker} serviceName={serviceName} />
                    </button>
                  </TableCell>
                  <TableCell>
                    <WorkerStatusBadge
                      status={worker.status}
                      variant="outline"
                    />
                  </TableCell>
                  <TableCell>
                    <SlotsBadge
                      available={
                        worker.status === 'ACTIVE'
                          ? worker.availableRuns || 0
                          : 0
                      }
                      max={worker.maxRuns || 0}
                    />
                  </TableCell>
                  <TableCell>{worker.type}</TableCell>
                  <TableCell>
                    <TooltipProvider>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span>
                            <Time
                              date={worker.lastHeartbeatAt}
                              variant="timeSince"
                            />
                          </span>
                        </TooltipTrigger>
                        <TooltipContent>
                          <Time
                            date={worker.lastHeartbeatAt}
                            variant="timestamp"
                          />
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-2">
                      {worker.status !== 'ACTIVE' &&
                        worker.status !== 'INACTIVE' && (
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() =>
                              handleResumeWorker(worker.metadata.id)
                            }
                            title="Resume Worker"
                          >
                            <Play className="h-4 w-4" />
                          </Button>
                        )}
                      {worker.status !== 'PAUSED' &&
                        worker.status !== 'INACTIVE' && (
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() =>
                              handlePauseWorker(worker.metadata.id)
                            }
                            title="Pause Worker"
                          >
                            <Pause className="h-4 w-4" />
                          </Button>
                        )}
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
