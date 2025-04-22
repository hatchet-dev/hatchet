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
import useTenant from './use-tenant';
import { createContext, useContext, PropsWithChildren } from 'react';
import { FilterProvider, useFilters } from './utils/use-filters';
import { PaginationProvider, usePagination } from './utils/use-pagination';

// Types for filters and pagination
interface ManagedComputeFilters {
  search?: string;
  sortBy?: string;
  sortDirection?: 'asc' | 'desc';
  fromDate?: string;
  toDate?: string;
}

// Create params
interface CreateManagedComputeParams {
  data: CreateManagedWorkerRequest;
}

// Update params
interface UpdateManagedComputeParams {
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
      if (!tenant) {
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
      <PaginationProvider initialPage={1} initialPageSize={10}>
        <ManagedComputeProviderContent refetchInterval={refetchInterval}>
          {children}
        </ManagedComputeProviderContent>
      </PaginationProvider>
    </FilterProvider>
  );
}
