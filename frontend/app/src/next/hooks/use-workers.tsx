import api, { Worker, UpdateWorkerRequest, WorkerList } from '@/lib/api';
import {
  useMutation,
  UseMutationResult,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import { useCurrentTenantId } from './use-tenant';
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

export interface WorkerPool {
  id?: string;
  name: string;
  type: Worker['type'];
  workers: Worker[];
  activeCount: number;
  pausedCount: number;
  inactiveCount: number;
  totalMaxRuns: number;
  totalAvailableRuns: number;
  actions: string[];
}

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
  poolName?: string; // Optional parameter to update all workers in a pool
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
  pools: WorkerPool[];
}

interface WorkersProviderProps extends PropsWithChildren {
  refetchInterval?: number;
  status?: 'ACTIVE' | 'INACTIVE' | 'PAUSED';
  workerName?: string;
}

const WorkersContext = createContext<WorkersState | null>(null);

export function useWorkers() {
  const context = useContext(WorkersContext);
  if (!context) {
    throw new Error('useWorkers must be used within a WorkersProvider');
  }
  return context;
}

const statusToInt = (status: WorkersFilters['status']) => {
  switch (status) {
    case 'ACTIVE':
      return 1;
    case 'INACTIVE':
      return 2;
    case 'PAUSED':
      return 3;
    case undefined:
      return 4;
    default:
      // eslint-disable-next-line no-case-declarations
      const exhaustiveCheck: never = status;
      throw new Error(`Unhandled action type: ${exhaustiveCheck}`);
  }
};

function WorkersProviderContent({
  children,
  refetchInterval,
}: WorkersProviderProps) {
  const { tenantId } = useCurrentTenantId();
  const queryClient = useQueryClient();
  const filters = useFilters<WorkersFilters>();
  const pagination = usePagination();
  const { toast } = useToast();

  const listWorkersQuery = useQuery({
    queryKey: ['worker:list', tenantId, filters.filters, pagination],
    queryFn: async () => {
      try {
        const res = await api.workerList(tenantId);

        const searchLower = filters.filters.search?.toLowerCase();
        const fromDate = filters.filters.fromDate
          ? new Date(filters.filters.fromDate)
          : undefined;
        const toDate = filters.filters?.toDate
          ? new Date(filters.filters.toDate)
          : undefined;

        const filteredRows = (res?.data?.rows || [])
          .filter(
            (worker) =>
              !searchLower || worker.name.toLowerCase().includes(searchLower),
          )
          .filter((worker) => {
            if (!worker.lastHeartbeatAt || !fromDate) {
              return true;
            }
            const lastHeartbeatAt = new Date(worker.lastHeartbeatAt);
            return lastHeartbeatAt >= fromDate;
          })
          .filter((worker) => {
            if (!worker.lastHeartbeatAt || !toDate) {
              return true;
            }
            const lastHeartbeatAt = new Date(worker.lastHeartbeatAt);
            return lastHeartbeatAt <= toDate;
          })
          .filter(
            (worker) =>
              !filters.filters.status ||
              worker.status === filters.filters.status,
          )
          .sort((a, b) => {
            if (!filters.filters.sortBy) {
              const statusA = statusToInt(a.status);
              const statusB = statusToInt(b.status);

              if (statusA < statusB) {
                return -1;
              }
              if (statusA > statusB) {
                return 1;
              }

              const lastHeartbeatA = a.lastHeartbeatAt;
              const lastHeartbeatB = b.lastHeartbeatAt;

              if (lastHeartbeatA && lastHeartbeatB) {
                const dateA = new Date(lastHeartbeatA).getTime();
                const dateB = new Date(lastHeartbeatB).getTime();

                return dateB - dateA;
              }

              return 0;
            }

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

        const pools: WorkerPool[] = Object.entries(groupedByName).map(
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
              actions: [
                ...new Set(workers.flatMap((w) => w.actions || [])),
              ].sort((a, b) => a.localeCompare(b)),
            };
          },
        );

        return {
          ...res.data,
          rows: filteredRows,
          pools,
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
          pools: [],
        };
      }
    },
    refetchInterval,
  });

  const updateWorkerMutation = useMutation({
    mutationKey: ['worker:update', tenantId],
    mutationFn: async ({ workerId, data }: UpdateWorkerParams) => {
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
    mutationKey: ['worker:bulkUpdate', tenantId],
    mutationFn: async ({
      workerIds,
      data,
      poolName,
    }: BulkUpdateWorkersParams) => {
      // If pool name is provided, get all worker IDs for that pool
      let targetWorkerIds = workerIds;
      if (poolName && !workerIds.length) {
        const workers = listWorkersQuery.data?.rows || [];
        targetWorkerIds = workers
          .filter((worker: Worker) => worker.name === poolName)
          .map((worker: Worker) => worker.metadata.id);
      }

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
    pools: listWorkersQuery.data?.pools || [],
  };

  return createElement(WorkersContext.Provider, { value }, children);
}

export function WorkersProvider({
  children,
  refetchInterval,
  status,
  workerName,
}: WorkersProviderProps) {
  return (
    <FilterProvider<WorkersFilters>
      initialFilters={{
        search: workerName,
        sortBy: undefined,
        sortDirection: undefined,
        fromDate: undefined,
        toDate: undefined,
        status,
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
