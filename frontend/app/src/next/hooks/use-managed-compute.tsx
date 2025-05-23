import { cloudApi } from '@/lib/api/api';
import {
  ManagedWorker,
  ManagedWorkerList,
  CreateManagedWorkerRequest,
  UpdateManagedWorkerRequest,
  MonthlyComputeCost,
} from '@/lib/api/generated/cloud/data-contracts';
import {
  useMutation,
  UseMutationResult,
  useQuery,
  UseQueryResult,
} from '@tanstack/react-query';
import { WorkerType } from '@/lib/api';
import { Worker } from '@/lib/api/generated/data-contracts';
import { WorkerPool } from './use-workers';
import { useCurrentTenantId } from './use-tenant';
import { createContext, useContext, PropsWithChildren, useMemo } from 'react';
import { FilterProvider, useFilters } from './utils/use-filters';
import { PaginationProvider, usePagination } from './utils/use-pagination';
import useApiMeta from './use-api-meta';
import { useWorkers } from './use-workers';
import { useToast } from './utils/use-toast';

interface ManagedComputeFilters {
  search?: string;
  sortBy?: string;
  sortDirection?: 'asc' | 'desc';
  fromDate?: string;
  toDate?: string;
}

interface CreateManagedComputeParams {
  data: CreateManagedWorkerRequest;
}

interface UpdateManagedComputeParams {
  managedWorkerId: string;
  data: UpdateManagedWorkerRequest;
}

interface ManagedComputeState {
  data?: ManagedWorker[];
  paginationData?: ManagedWorkerList['pagination'];
  isLoading: boolean;
  create: UseMutationResult<
    ManagedWorker,
    Error,
    CreateManagedComputeParams,
    unknown
  >;
  update: UseMutationResult<
    ManagedWorker,
    Error,
    UpdateManagedComputeParams,
    unknown
  >;
  delete: UseMutationResult<ManagedWorker, Error, string, unknown>;
  filters: ReturnType<typeof useFilters<ManagedComputeFilters>>;
  pagination: ReturnType<typeof usePagination>;
  costs: UseQueryResult<MonthlyComputeCost, Error>;
}

interface ManagedComputeProviderProps extends PropsWithChildren {
  refetchInterval?: number;
}

const ManagedComputeContext = createContext<ManagedComputeState | null>(null);

export function useManagedCompute() {
  const context = useContext(ManagedComputeContext);
  if (!context) {
    throw new Error(
      'useManagedCompute must be used within a ManagedComputeProvider',
    );
  }
  return context;
}

function ManagedComputeProviderContent({
  children,
  refetchInterval,
}: ManagedComputeProviderProps) {
  const { cloud, isCloud } = useApiMeta();
  const { tenantId } = useCurrentTenantId();
  const filters = useFilters<ManagedComputeFilters>();
  const pagination = usePagination();
  const { toast } = useToast();

  const listManagedComputeQuery = useQuery({
    queryKey: [
      'managed-compute:list',
      tenantId,
      filters.filters.search,
      filters.filters.sortBy,
      filters.filters.sortDirection,
      filters.filters.fromDate,
      filters.filters.toDate,
      pagination.currentPage,
      pagination.pageSize,
    ],
    queryFn: async () => {
      if (!cloud) {
        return { rows: [], pagination: { current_page: 0, num_pages: 0 } };
      }

      try {
        // Build query params
        const queryParams: Record<string, any> = {
          page: pagination.currentPage,
          limit: pagination.pageSize,
        };

        if (filters.filters.sortBy) {
          queryParams.orderBy = filters.filters.sortBy;
          queryParams.orderDirection = filters.filters.sortDirection || 'asc';
        }

        const res = await cloudApi.managedWorkerList(tenantId);

        // Client-side filtering for search if API doesn't support it
        let filteredRows = res.data.rows || [];
        if (filters.filters.search) {
          const searchLower = filters.filters.search.toLowerCase();
          filteredRows = filteredRows.filter((worker: ManagedWorker) =>
            worker.name?.toLowerCase().includes(searchLower),
          );
        }

        // Client-side date filtering
        if (filters.filters.fromDate) {
          const fromDate = new Date(filters.filters.fromDate);
          filteredRows = filteredRows.filter((worker: ManagedWorker) => {
            const createdAt = new Date(worker.metadata.createdAt);
            return createdAt >= fromDate;
          });
        }

        if (filters.filters.toDate) {
          const toDate = new Date(filters.filters.toDate);
          filteredRows = filteredRows.filter((worker: ManagedWorker) => {
            const createdAt = new Date(worker.metadata.createdAt);
            return createdAt <= toDate;
          });
        }

        // Client-side sorting if API doesn't support it
        if (filters.filters.sortBy) {
          filteredRows.sort((a: ManagedWorker, b: ManagedWorker) => {
            let valueA: any;
            let valueB: any;

            switch (filters.filters.sortBy) {
              case 'name':
                valueA = a.name;
                valueB = b.name;
                break;
              case 'createdAt':
                valueA = new Date(a.metadata.createdAt).getTime();
                valueB = new Date(b.metadata.createdAt).getTime();
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

        return {
          ...res.data,
          rows: filteredRows,
        };
      } catch (error) {
        toast({
          title: 'Error fetching managed compute',

          variant: 'destructive',
          error,
        });
        return { rows: [], pagination: { current_page: 0, num_pages: 0 } };
      }
    },
    enabled: isCloud,
    refetchInterval,
  });

  // Create implementation
  const createManagedComputeMutation = useMutation({
    mutationKey: ['managed-compute:create', tenantId],
    mutationFn: async ({ data }: CreateManagedComputeParams) => {
      try {
        // Validate that only one of numReplicas or autoscaling is set
        if (data.runtimeConfig?.autoscaling) {
          data.runtimeConfig.numReplicas = undefined;
        } else if (data.runtimeConfig) {
          data.runtimeConfig.autoscaling = undefined;
        }

        const res = await cloudApi.managedWorkerCreate(tenantId, data);
        return res.data;
      } catch (error) {
        toast({
          title: 'Error creating managed compute',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: () => {
      listManagedComputeQuery.refetch();
    },
  });

  // Update implementation
  const updateManagedComputeMutation = useMutation({
    mutationKey: ['managed-compute:update', tenantId],
    mutationFn: async ({
      managedWorkerId,
      data,
    }: UpdateManagedComputeParams) => {
      try {
        // Validate that only one of numReplicas or autoscaling is set
        if (data.runtimeConfig?.autoscaling) {
          data.runtimeConfig.numReplicas = undefined;
        } else if (data.runtimeConfig) {
          data.runtimeConfig.autoscaling = undefined;
        }

        const res = await cloudApi.managedWorkerUpdate(managedWorkerId, data);
        return res.data;
      } catch (error) {
        toast({
          title: 'Error updating managed compute',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: () => {
      listManagedComputeQuery.refetch();
    },
  });

  // Delete implementation
  const deleteManagedComputeMutation = useMutation({
    mutationKey: ['managed-compute:delete', tenantId],
    mutationFn: async (managedWorkerId: string) => {
      try {
        const res = await cloudApi.managedWorkerDelete(managedWorkerId);
        return res.data;
      } catch (error) {
        toast({
          title: 'Error deleting managed compute',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: () => {
      listManagedComputeQuery.refetch();
    },
  });

  const costsQuery = useQuery({
    queryKey: ['managed-compute:costs', tenantId],
    queryFn: async () => {
      try {
        return (await cloudApi.computeCostGet(tenantId)).data;
      } catch (error) {
        toast({
          title: 'Error fetching compute costs',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    enabled: isCloud,
  });

  const value = {
    data: listManagedComputeQuery.data?.rows || [],
    paginationData: listManagedComputeQuery.data?.pagination,
    isLoading: listManagedComputeQuery.isLoading,
    create: createManagedComputeMutation,
    update: updateManagedComputeMutation,
    delete: deleteManagedComputeMutation,
    filters,
    pagination,
    costs: costsQuery,
  };

  return (
    <ManagedComputeContext.Provider value={value}>
      {children}
    </ManagedComputeContext.Provider>
  );
}

export function ManagedComputeProvider({
  children,
  refetchInterval,
}: ManagedComputeProviderProps) {
  return (
    <FilterProvider<ManagedComputeFilters>
      initialFilters={{
        search: undefined,
        sortBy: undefined,
        sortDirection: 'asc',
        fromDate: undefined,
        toDate: undefined,
      }}
    >
      <PaginationProvider initialPage={1} initialPageSize={50}>
        <ManagedComputeProviderContent refetchInterval={refetchInterval}>
          {children}
        </ManagedComputeProviderContent>
      </PaginationProvider>
    </FilterProvider>
  );
}

const mapManagedWorkerToWorkerPool = (worker: ManagedWorker): WorkerPool => {
  const mappedWorker: Worker = {
    metadata: worker.metadata,
    name: worker.name,
    type: WorkerType.MANAGED,
    status: 'ACTIVE', // Default to ACTIVE since ManagedWorker doesn't have status
    maxRuns: 0, // Default to 0 since ManagedWorker doesn't have maxRuns
    availableRuns: 0, // Default to 0 since ManagedWorker doesn't have availableRuns
  };

  return {
    name: worker.name || '',
    id: worker.metadata.id || '',
    type: WorkerType.MANAGED,
    workers: [mappedWorker],
    activeCount: 1, // Managed workers are always considered active // TODO
    pausedCount: 0,
    inactiveCount: 0,
    totalMaxRuns: 0,
    totalAvailableRuns: 0,
    actions: [],
  };
};

const createPoolUniqueKey = (pool: WorkerPool) => {
  if (!pool.actions) {
    return pool.name;
  }

  return pool.actions.join(';');
};

export const useUnifiedWorkerPools = () => {
  const { pools: regularPools, isLoading: workersIsLoading } = useWorkers();
  const { data: managedCompute, isLoading: managedComputeIsLoading } =
    useManagedCompute();

  const pools = useMemo(() => {
    const managedComputePools = (managedCompute || []).map((worker) => {
      return mapManagedWorkerToWorkerPool(worker);
    });

    const allPools = [...regularPools, ...managedComputePools];
    const uniquePools = allPools.reduce(
      (acc, pool) => {
        const key = createPoolUniqueKey(pool);
        if (!acc[key]) {
          acc[key] = pool;
        } else {
          const existing = acc[key];
          acc[key] = {
            ...existing,
            workers: [...existing.workers, ...pool.workers],
            activeCount: existing.activeCount + pool.activeCount,
            inactiveCount: existing.inactiveCount + pool.inactiveCount,
            pausedCount: existing.pausedCount + pool.pausedCount,
            totalMaxRuns: existing.totalMaxRuns + pool.totalMaxRuns,
            totalAvailableRuns:
              existing.totalAvailableRuns + pool.totalAvailableRuns,
          };
        }
        return acc;
      },
      {} as Record<string, WorkerPool>,
    );

    return Object.values(uniquePools);
  }, [regularPools, managedCompute]);

  return {
    pools,
    isLoading: workersIsLoading || managedComputeIsLoading,
  };
};
