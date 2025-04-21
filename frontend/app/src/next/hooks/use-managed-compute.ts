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
import {
  useState,
  createContext,
  useContext,
  PropsWithChildren,
  createElement,
} from 'react';

// Types for filters and pagination
interface ManagedComputeFilters {
  search?: string;
  sortBy?: string;
  sortDirection?: 'asc' | 'desc';
  fromDate?: string;
  toDate?: string;
}

interface ManagedComputePagination {
  currentPage: number;
  pageSize: number;
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
  pagination?: ManagedWorkerList['pagination'];
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

  // Added from context
  filters: ManagedComputeFilters;
  setFilters: (filters: ManagedComputeFilters) => void;
  paginationState: ManagedComputePagination;
  setPagination: (pagination: ManagedComputePagination) => void;
}

interface UseManagedComputeOptions {
  refetchInterval?: number;
  initialFilters?: ManagedComputeFilters;
  initialPagination?: ManagedComputePagination;
}

export default function useManagedCompute({
  refetchInterval,
  initialFilters = {},
  initialPagination = { currentPage: 1, pageSize: 10 },
}: UseManagedComputeOptions = {}): ManagedComputeState {
  const { tenant } = useTenant();

  // State from the former context
  const [filters, setFilters] = useState<ManagedComputeFilters>(initialFilters);
  const [paginationState, setPagination] =
    useState<ManagedComputePagination>(initialPagination);

  const listManagedComputeQuery = useQuery({
    queryKey: [
      'managed-compute:list',
      tenant,
      filters.search,
      filters.sortBy,
      filters.sortDirection,
      filters.fromDate,
      filters.toDate,
      paginationState.currentPage,
      paginationState.pageSize,
    ],
    queryFn: async () => {
      if (!tenant) {
        return { rows: [], pagination: { current_page: 0, num_pages: 0 } };
      }

      // Build query params
      const queryParams: Record<string, any> = {
        page: paginationState.currentPage,
        limit: paginationState.pageSize,
      };

      if (filters.sortBy) {
        queryParams.orderBy = filters.sortBy;
        queryParams.orderDirection = filters.sortDirection || 'asc';
      }

      const res = await cloudApi.managedWorkerList(tenant?.metadata.id || '');

      // Client-side filtering for search if API doesn't support it
      let filteredRows = res.data.rows || [];
      if (filters.search) {
        const searchLower = filters.search.toLowerCase();
        filteredRows = filteredRows.filter((worker: ManagedWorker) =>
          worker.name?.toLowerCase().includes(searchLower),
        );
      }

      // Client-side date filtering
      if (filters.fromDate) {
        const fromDate = new Date(filters.fromDate);
        filteredRows = filteredRows.filter((worker: ManagedWorker) => {
          const createdAt = new Date(worker.metadata.createdAt);
          return createdAt >= fromDate;
        });
      }

      if (filters.toDate) {
        const toDate = new Date(filters.toDate);
        filteredRows = filteredRows.filter((worker: ManagedWorker) => {
          const createdAt = new Date(worker.metadata.createdAt);
          return createdAt <= toDate;
        });
      }

      // Client-side sorting if API doesn't support it
      if (filters.sortBy) {
        filteredRows.sort((a: ManagedWorker, b: ManagedWorker) => {
          let valueA: any;
          let valueB: any;

          switch (filters.sortBy) {
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

          const direction = filters.sortDirection === 'desc' ? -1 : 1;
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

  return {
    data: listManagedComputeQuery.data?.rows || [],
    pagination: listManagedComputeQuery.data?.pagination,
    isLoading: listManagedComputeQuery.isLoading,
    create: createManagedComputeMutation,
    update: updateManagedComputeMutation,
    delete: deleteManagedComputeMutation,

    // Added from context
    filters,
    setFilters,
    paginationState,
    setPagination,
  };
}

// Context implementation (to maintain compatibility with components)
interface ManagedComputeContextType extends ManagedComputeState {}

const ManagedComputeContext = createContext<
  ManagedComputeContextType | undefined
>(undefined);

export const useManagedComputeContext = () => {
  const context = useContext(ManagedComputeContext);
  if (context === undefined) {
    throw new Error(
      'useManagedComputeContext must be used within a ManagedComputeProvider',
    );
  }
  return context;
};

interface ManagedComputeProviderProps extends PropsWithChildren {
  options?: UseManagedComputeOptions;
}

export function ManagedComputeProvider(props: ManagedComputeProviderProps) {
  const { children, options = {} } = props;
  const managedComputeState = useManagedCompute(options);

  return createElement(
    ManagedComputeContext.Provider,
    { value: managedComputeState },
    children,
  );
}
