import api, { Worker, UpdateWorkerRequest, WorkerList } from '@/lib/api';
import {
  useMutation,
  UseMutationResult,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import useTenant from './use-tenant';
import {
  createContext,
  useContext,
  PropsWithChildren,
  createElement,
} from 'react';
import { FilterProvider, useFilters } from './utils/use-filters';
import { PaginationProvider, usePagination } from './utils/use-pagination';
import { useToast } from './utils/use-toast';

export type { Worker };

export interface WorkerService {
  id?: string;
  name: string;
  type: Worker['type'];
  workers: Worker[];
  activeCount: number;
  pausedCount: number;
  inactiveCount: number;
  totalMaxRuns: number;
  totalAvailableRuns: number;
}

// Types for filters and pagination
interface WorkersFilters {
  search?: string;
  sortBy?: string;
  sortDirection?: 'asc' | 'desc';
  fromDate?: string;
  toDate?: string;
  status?: 'ACTIVE' | 'INACTIVE' | 'PAUSED';
}

// Update worker params
interface UpdateWorkerParams {
  workerId: string;
  data: UpdateWorkerRequest;
}

// Bulk update workers params
interface BulkUpdateWorkersParams {
  workerIds: string[];
  data: UpdateWorkerRequest;
  serviceName?: string; // Optional parameter to update all workers in a service
}

// Main hook return type
interface WorkersState {
  data?: WorkerList['rows'];
  paginationData?: WorkerList['pagination'];
  isLoading: boolean;
  update: UseMutationResult<Worker, Error, UpdateWorkerParams, unknown>;
  bulkUpdate: UseMutationResult<void, Error, BulkUpdateWorkersParams, unknown>;
  filters: ReturnType<typeof useFilters<WorkersFilters>>;
  pagination: ReturnType<typeof usePagination>;
  services: WorkerService[];
}

interface WorkersProviderProps extends PropsWithChildren {
  refetchInterval?: number;
}

const WorkersContext = createContext<WorkersState | null>(null);

export function useWorkers() {
  const context = useContext(WorkersContext);
  if (!context) {
    throw new Error('useWorkers must be used within a WorkersProvider');
  }
  return context;
}

function WorkersProviderContent({
  children,
  refetchInterval,
}: WorkersProviderProps) {
  const { tenant } = useTenant();
  const queryClient = useQueryClient();
  const filters = useFilters<WorkersFilters>();
  const pagination = usePagination();
  const { toast } = useToast();

  const listWorkersQuery = useQuery({
    queryKey: ['worker:list', tenant, filters.filters, pagination],
    queryFn: async () => {
      if (!tenant) {
        return {
          rows: [],
          pagination: { current_page: 0, num_pages: 0 },
          services: [],
        };
      }

      try {
        const res = await api.workerList(tenant?.metadata.id || '');

        const sorted = (res?.data?.rows || []).sort((a, b) => {
          const aCreatedAt = new Date(a.metadata.createdAt);
          const bCreatedAt = new Date(b.metadata.createdAt);
          return bCreatedAt.getTime() - aCreatedAt.getTime();
        });

        // Client-side filtering for search if API doesn't support it
        let filteredRows = sorted || [];
        if (filters.filters.search) {
          const searchLower = filters.filters.search.toLowerCase();
          filteredRows = filteredRows.filter((worker) =>
            worker.name.toLowerCase().includes(searchLower),
          );
        }

        // Client-side date filtering
        if (filters.filters.fromDate) {
          const fromDate = new Date(filters.filters.fromDate);
          filteredRows = filteredRows.filter((worker) => {
            if (!worker.lastHeartbeatAt) {
              return true;
            }
            const lastHeartbeatAt = new Date(worker.lastHeartbeatAt);
            return lastHeartbeatAt >= fromDate;
          });
        }

        if (filters.filters.toDate) {
          const toDate = new Date(filters.filters.toDate);
          filteredRows = filteredRows.filter((worker) => {
            if (!worker.lastHeartbeatAt) {
              return true;
            }
            const lastHeartbeatAt = new Date(worker.lastHeartbeatAt);
            return lastHeartbeatAt <= toDate;
          });
        }

        // Filter by status
        if (filters.filters.status) {
          filteredRows = filteredRows.filter(
            (worker) => worker.status === filters.filters.status,
          );
        }

        // Client-side sorting if API doesn't support it
        if (filters.filters.sortBy) {
          filteredRows.sort((a, b) => {
            let valueA: any;
            let valueB: any;

            switch (filters.filters.sortBy) {
              case 'name':
                valueA = a.name;
                valueB = b.name;
                break;
              case 'lastHeartbeatAt':
                valueA = a.lastHeartbeatAt
                  ? new Date(a.lastHeartbeatAt).getTime()
                  : 0;
                valueB = b.lastHeartbeatAt
                  ? new Date(b.lastHeartbeatAt).getTime()
                  : 0;
                break;
              case 'status':
                valueA = a.status || '';
                valueB = b.status || '';
                break;
              case 'type':
                valueA = a.type;
                valueB = b.type;
                break;
              default:
                return 0;
            }

            const direction = filters.filters.sortDirection === 'desc' ? -1 : 1;
            if (valueA < valueB) {
              return -1 * direction;
            }
            if (valueA > valueB) {
              return 1 * direction;
            }
            return 0;
          });
        }

        const groupedByName = filteredRows.reduce(
          (acc, worker) => {
            const name = worker.name;
            if (!acc[name]) {
              acc[name] = [];
            }
            acc[name].push(worker);
            return acc;
          },
          {} as Record<string, Worker[]>,
        );

        const services = Object.entries(groupedByName).map(
          ([name, workers]) => {
            const activeWorkers = workers.filter((w) => w.status === 'ACTIVE');
            return {
              name,
              type: workers[0].type,
              workers,
              activeCount: activeWorkers.length,
              inactiveCount: workers.filter((w) => w.status === 'INACTIVE')
                .length,
              pausedCount: workers.filter((w) => w.status === 'PAUSED').length,
              totalMaxRuns: activeWorkers.reduce(
                (sum, worker) => sum + (worker.maxRuns || 0),
                0,
              ),
              totalAvailableRuns: activeWorkers.reduce(
                (sum, worker) => sum + (worker.availableRuns || 0),
                0,
              ),
            } as WorkerService;
          },
        );

        return {
          ...res.data,
          rows: filteredRows,
          services,
        };
      } catch (error) {
        toast({
          title: 'Error fetching workers',

          variant: 'destructive',
          error,
        });
        return {
          rows: [],
          pagination: { current_page: 0, num_pages: 0 },
          services: [],
        };
      }
    },
    refetchInterval,
  });

  const updateWorkerMutation = useMutation({
    mutationKey: ['worker:update', tenant],
    mutationFn: async ({ workerId, data }: UpdateWorkerParams) => {
      if (!tenant) {
        throw new Error('Tenant not found');
      }
      try {
        const res = await api.workerUpdate(workerId, data);
        return res.data;
      } catch (error) {
        toast({
          title: 'Error updating worker',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: () => {
      listWorkersQuery.refetch();
    },
  });

  // Bulk update mutation
  const bulkUpdateWorkersMutation = useMutation({
    mutationKey: ['worker:bulkUpdate', tenant],
    mutationFn: async ({
      workerIds,
      data,
      serviceName,
    }: BulkUpdateWorkersParams) => {
      if (!tenant) {
        throw new Error('Tenant not found');
      }

      // If service name is provided, get all worker IDs for that service
      let targetWorkerIds = workerIds;
      if (serviceName && !workerIds.length) {
        const workers = listWorkersQuery.data?.rows || [];
        targetWorkerIds = workers
          .filter((worker: Worker) => worker.name === serviceName)
          .map((worker: Worker) => worker.metadata.id);
      }

      // Execute all updates in parallel
      try {
        await Promise.all(
          targetWorkerIds.map((workerId) => api.workerUpdate(workerId, data)),
        );
      } catch (error) {
        toast({
          title: 'Error bulk updating workers',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ['worker:list'],
      });
    },
  });

  const value = {
    data: listWorkersQuery.data?.rows || [],
    paginationData: listWorkersQuery.data?.pagination,
    isLoading: listWorkersQuery.isLoading,
    update: updateWorkerMutation,
    bulkUpdate: bulkUpdateWorkersMutation,
    filters,
    pagination,
    services: listWorkersQuery.data?.services || [],
  };

  return createElement(WorkersContext.Provider, { value }, children);
}

export function WorkersProvider({
  children,
  refetchInterval,
}: WorkersProviderProps) {
  return (
    <FilterProvider<WorkersFilters>
      initialFilters={{
        search: undefined,
        sortBy: undefined,
        sortDirection: undefined,
        fromDate: undefined,
        toDate: undefined,
        status: undefined,
      }}
    >
      <PaginationProvider initialPage={1} initialPageSize={50}>
        <WorkersProviderContent refetchInterval={refetchInterval}>
          {children}
        </WorkersProviderContent>
      </PaginationProvider>
    </FilterProvider>
  );
}
