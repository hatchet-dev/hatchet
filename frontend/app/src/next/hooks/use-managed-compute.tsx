import { cloudApi } from '@/lib/api/api';
import {
  ManagedWorker,
  ManagedWorkerList,
  CreateManagedWorkerRequest,
  UpdateManagedWorkerRequest,
} from '@/lib/api/generated/cloud/data-contracts';
import {
  useMutation,
  UseMutationResult,
  useQuery,
} from '@tanstack/react-query';
import { WorkerType } from '@/lib/api';
import { Worker } from '@/lib/api/generated/data-contracts';
import { WorkerService } from './use-workers';
import useTenant from './use-tenant';
import { createContext, useContext, PropsWithChildren, useMemo } from 'react';
import { FilterProvider, useFilters } from './utils/use-filters';
import { PaginationProvider, usePagination } from './utils/use-pagination';
import useApiMeta from './use-api-meta';
import { useWorkers } from './use-workers';
// Types for filters and pagination
interface ManagedComputeFilters {
  search?: string;
  sortBy?: string;
  sortDirection?: 'asc' | 'desc';
  fromDate?: string;
  toDate?: string;
}

// Create params
export interface CreateManagedComputeParams {
  data: CreateManagedWorkerRequest;
}

// Update params
export interface UpdateManagedComputeParams {
  managedWorkerId: string;
  data: UpdateManagedWorkerRequest;
}

// Main hook return type
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
  const { cloud } = useApiMeta();
  const { tenant } = useTenant();
  const filters = useFilters<ManagedComputeFilters>();
  const pagination = usePagination();

  const listManagedComputeQuery = useQuery({
    queryKey: [
      'managed-compute:list',
      tenant,
      filters.filters.search,
      filters.filters.sortBy,
      filters.filters.sortDirection,
      filters.filters.fromDate,
      filters.filters.toDate,
      pagination.currentPage,
      pagination.pageSize,
    ],
    queryFn: async () => {
      if (!cloud || !tenant) {
        return { rows: [], pagination: { current_page: 0, num_pages: 0 } };
      }

      // Build query params
      const queryParams: Record<string, any> = {
        page: pagination.currentPage,
        limit: pagination.pageSize,
      };

      if (filters.filters.sortBy) {
        queryParams.orderBy = filters.filters.sortBy;
        queryParams.orderDirection = filters.filters.sortDirection || 'asc';
      }

      const res = await cloudApi.managedWorkerList(tenant?.metadata.id || '');

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
    },
    refetchInterval,
  });

  // Create implementation
  const createManagedComputeMutation = useMutation({
    mutationKey: ['managed-compute:create', tenant],
    mutationFn: async ({ data }: CreateManagedComputeParams) => {
      if (!tenant) {
        throw new Error('Tenant not found');
      }

      const res = await cloudApi.managedWorkerCreate(tenant.metadata.id, data);

      return res.data;
    },
    onSuccess: () => {
      listManagedComputeQuery.refetch();
    },
  });

  // Update implementation
  const updateManagedComputeMutation = useMutation({
    mutationKey: ['managed-compute:update', tenant],
    mutationFn: async ({
      managedWorkerId,
      data,
    }: UpdateManagedComputeParams) => {
      if (!tenant) {
        throw new Error('Tenant not found');
      }

      const res = await cloudApi.managedWorkerUpdate(managedWorkerId, data);

      return res.data;
    },
    onSuccess: () => {
      listManagedComputeQuery.refetch();
    },
  });

  // Delete implementation
  const deleteManagedComputeMutation = useMutation({
    mutationKey: ['managed-compute:delete', tenant],
    mutationFn: async (managedWorkerId: string) => {
      if (!tenant) {
        throw new Error('Tenant not found');
      }

      const res = await cloudApi.managedWorkerDelete(managedWorkerId);
      return res.data;
    },
    onSuccess: () => {
      listManagedComputeQuery.refetch();
    },
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

const mapManagedWorkerToWorkerService = (
  worker: ManagedWorker,
): WorkerService => {
  // Map ManagedWorker to WorkerService format
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
  } as WorkerService;
};

// Helper function to unify regular and managed workers into services
export const useUnifiedWorkerServices = () => {
  const { services: regularServices } = useWorkers();
  const { data: managedCompute } = useManagedCompute();

  return useMemo(() => {
    // Create services from managed compute workers
    const managedComputeServices = (managedCompute || []).map((worker) => {
      return mapManagedWorkerToWorkerService(worker);
    });

    // Combine and deduplicate services
    const allServices = [...regularServices, ...managedComputeServices];
    const uniqueServices = allServices.reduce(
      (acc, service) => {
        if (!acc[service.name]) {
          acc[service.name] = service;
        } else {
          // Merge services with the same name
          const existing = acc[service.name];
          acc[service.name] = {
            ...existing,
            workers: [...existing.workers, ...service.workers],
            activeCount: existing.activeCount + service.activeCount,
            inactiveCount: existing.inactiveCount + service.inactiveCount,
            pausedCount: existing.pausedCount + service.pausedCount,
            totalMaxRuns: existing.totalMaxRuns + service.totalMaxRuns,
            totalAvailableRuns:
              existing.totalAvailableRuns + service.totalAvailableRuns,
          };
        }
        return acc;
      },
      {} as Record<string, WorkerService>,
    );

    return Object.values(uniqueServices);
  }, [regularServices, managedCompute]);
};
