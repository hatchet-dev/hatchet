import api, { Worker, UpdateWorkerRequest, WorkerList } from '@/lib/api';
import {
  useMutation,
  UseMutationResult,
  useQuery,
  useQueryClient,
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
interface WorkersFilters {
  search?: string;
  sortBy?: string;
  sortDirection?: 'asc' | 'desc';
  fromDate?: string;
  toDate?: string;
  status?: 'ACTIVE' | 'INACTIVE' | 'PAUSED';
}

interface WorkersPagination {
  currentPage: number;
  pageSize: number;
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
  pagination?: WorkerList['pagination'];
  isLoading: boolean;
  update: UseMutationResult<Worker, Error, UpdateWorkerParams, unknown>;
  bulkUpdate: UseMutationResult<void, Error, BulkUpdateWorkersParams, unknown>;

  // Added from context
  filters: WorkersFilters;
  setFilters: (filters: WorkersFilters) => void;
  paginationState: WorkersPagination;
  setPagination: (pagination: WorkersPagination) => void;
}

interface UseWorkersOptions {
  refetchInterval?: number;
  initialFilters?: WorkersFilters;
  initialPagination?: WorkersPagination;
}

export default function useWorkers({
  refetchInterval,
  initialFilters = {},
  initialPagination = { currentPage: 1, pageSize: 10 },
}: UseWorkersOptions = {}): WorkersState {
  const { tenant } = useTenant();
  const queryClient = useQueryClient();
  // State from the former context
  const [filters, setFilters] = useState<WorkersFilters>(initialFilters);
  const [paginationState, setPagination] =
    useState<WorkersPagination>(initialPagination);

  const listWorkersQuery = useQuery({
    queryKey: [
      'worker:list',
      tenant,
      filters.search,
      filters.sortBy,
      filters.sortDirection,
      filters.fromDate,
      filters.toDate,
      filters.status,
      paginationState.currentPage,
      paginationState.pageSize,
    ],
    queryFn: async () => {
      if (!tenant) {
        return { rows: [], pagination: { current_page: 0, num_pages: 0 } };
      }

      const res = await api.workerList(tenant?.metadata.id || '');

      const sorted = (res?.data?.rows || []).sort((a, b) => {
        const aCreatedAt = new Date(a.metadata.createdAt);
        const bCreatedAt = new Date(b.metadata.createdAt);
        return bCreatedAt.getTime() - aCreatedAt.getTime();
      });

      // Client-side filtering for search if API doesn't support it
      let filteredRows = sorted || [];
      if (filters.search) {
        const searchLower = filters.search.toLowerCase();
        filteredRows = filteredRows.filter((worker) =>
          worker.name.toLowerCase().includes(searchLower),
        );
      }

      // Client-side date filtering
      if (filters.fromDate) {
        const fromDate = new Date(filters.fromDate);
        filteredRows = filteredRows.filter((worker) => {
          if (!worker.lastHeartbeatAt) {
            return true;
          }
          const lastHeartbeatAt = new Date(worker.lastHeartbeatAt);
          return lastHeartbeatAt >= fromDate;
        });
      }

      if (filters.toDate) {
        const toDate = new Date(filters.toDate);
        filteredRows = filteredRows.filter((worker) => {
          if (!worker.lastHeartbeatAt) {
            return true;
          }
          const lastHeartbeatAt = new Date(worker.lastHeartbeatAt);
          return lastHeartbeatAt <= toDate;
        });
      }

      // Filter by status
      if (filters.status) {
        filteredRows = filteredRows.filter(
          (worker) => worker.status === filters.status,
        );
      }

      // Client-side sorting if API doesn't support it
      if (filters.sortBy) {
        filteredRows.sort((a, b) => {
          let valueA: any;
          let valueB: any;

          switch (filters.sortBy) {
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

  const updateWorkerMutation = useMutation({
    mutationKey: ['worker:update', tenant],
    mutationFn: async ({ workerId, data }: UpdateWorkerParams) => {
      if (!tenant) {
        throw new Error('Tenant not found');
      }
      const res = await api.workerUpdate(workerId, data);
      return res.data;
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
      await Promise.all(
        targetWorkerIds.map((workerId) => api.workerUpdate(workerId, data)),
      );
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ['worker:list'],
      });
    },
  });

  return {
    data: listWorkersQuery.data?.rows || [],
    pagination: listWorkersQuery.data?.pagination,
    isLoading: listWorkersQuery.isLoading,
    update: updateWorkerMutation,
    bulkUpdate: bulkUpdateWorkersMutation,

    // Added from context
    filters,
    setFilters,
    paginationState,
    setPagination,
  };
}

// Context implementation (to maintain compatibility with components)
interface WorkersContextType extends WorkersState {}

const WorkersContext = createContext<WorkersContextType | undefined>(undefined);

export const useWorkersContext = () => {
  const context = useContext(WorkersContext);
  if (context === undefined) {
    throw new Error('useWorkersContext must be used within a WorkersProvider');
  }
  return context;
};

interface WorkersProviderProps extends PropsWithChildren {
  options?: UseWorkersOptions;
}

export function WorkersProvider(props: WorkersProviderProps) {
  const { children, options = {} } = props;
  const workersState = useWorkers(options);

  return createElement(
    WorkersContext.Provider,
    { value: workersState },
    children,
  );
}
