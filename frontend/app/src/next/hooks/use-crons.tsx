import api, {
  CronWorkflows,
  CreateCronWorkflowTriggerRequest,
  CronWorkflowsList,
} from '@/lib/api';
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

// Types for filters and pagination
export interface CronsFilters {
  workflowId?: string;
  additionalMetadata?: string[];
}

// Update cron params
interface UpdateCronParams {
  cronId: string;
  workflowId: string;
  data: CreateCronWorkflowTriggerRequest;
}

// Create cron params
interface CreateCronParams {
  workflowId: string;
  data: CreateCronWorkflowTriggerRequest;
}

// Main hook return type
interface CronsState {
  data?: CronWorkflowsList['rows'];
  paginationData?: CronWorkflowsList['pagination'];
  isLoading: boolean;
  update: UseMutationResult<CronWorkflows, Error, UpdateCronParams, unknown>;
  create: UseMutationResult<CronWorkflows, Error, CreateCronParams, unknown>;
  delete: UseMutationResult<void, Error, string, unknown>;
  filters: ReturnType<typeof useFilters<CronsFilters>>;
  pagination: ReturnType<typeof usePagination>;
}

interface CronsProviderProps extends PropsWithChildren {
  refetchInterval?: number;
}

const CronsContext = createContext<CronsState | null>(null);

export function useCrons() {
  const context = useContext(CronsContext);
  if (!context) {
    throw new Error('useCrons must be used within a CronsProvider');
  }
  return context;
}

function CronsProviderContent({
  children,
  refetchInterval,
}: CronsProviderProps) {
  const { tenant } = useTenant();
  const queryClient = useQueryClient();
  const filters = useFilters<CronsFilters>();
  const pagination = usePagination();

  const listCronsQuery = useQuery({
    queryKey: ['cron:list', tenant, filters.filters, pagination],
    queryFn: async () => {
      if (!tenant) {
        return { rows: [], pagination: { current_page: 0, num_pages: 0 } };
      }

      const queryParams: Record<string, any> = {
        limit: pagination.pageSize,
        offset: Math.max(0, (pagination.currentPage - 1) * pagination.pageSize),
        ...filters.filters,
      };

      const res = await api.cronWorkflowList(
        tenant?.metadata.id || '',
        queryParams,
      );

      return res.data;
    },
    refetchInterval,
  });

  // Create implementation
  const createCronMutation = useMutation({
    mutationKey: ['cron:create', tenant],
    mutationFn: async ({ workflowId, data }: CreateCronParams) => {
      if (!tenant) {
        throw new Error('Tenant not found');
      }

      const res = await api.cronWorkflowTriggerCreate(
        tenant.metadata.id,
        workflowId,
        data,
      );

      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cron:list'] });
    },
  });

  // Delete implementation
  const deleteCronMutation = useMutation({
    mutationKey: ['cron:delete', tenant],
    mutationFn: async (cronId: string) => {
      if (!tenant) {
        throw new Error('Tenant not found');
      }

      await api.workflowCronDelete(tenant.metadata.id, cronId);
    },
    onSuccess: () => {
      listCronsQuery.refetch();
    },
  });

  const updateCronMutation = useMutation({
    mutationKey: ['cron:update', tenant],
    mutationFn: async ({ cronId, workflowId, data }: UpdateCronParams) => {
      if (!tenant) {
        throw new Error('Tenant not found');
      }

      // First delete the existing cron
      await api.workflowCronDelete(tenant.metadata.id, cronId);

      // Then create a new one with the updated data
      const res = await api.cronWorkflowTriggerCreate(
        tenant.metadata.id,
        workflowId,
        data,
      );

      return res.data;
    },
    onSuccess: () => {
      listCronsQuery.refetch();
    },
  });

  const value = {
    data: listCronsQuery.data?.rows || [],
    paginationData: listCronsQuery.data?.pagination,
    isLoading: listCronsQuery.isLoading,
    update: updateCronMutation,
    create: createCronMutation,
    delete: deleteCronMutation,
    filters,
    pagination,
  };

  return createElement(CronsContext.Provider, { value }, children);
}

export function CronsProvider({
  children,
  refetchInterval,
}: CronsProviderProps) {
  return (
    <FilterProvider<CronsFilters>
      initialFilters={{
        workflowId: undefined,
        additionalMetadata: [],
      }}
    >
      <PaginationProvider initialPage={1} initialPageSize={50}>
        <CronsProviderContent refetchInterval={refetchInterval}>
          {children}
        </CronsProviderContent>
      </PaginationProvider>
    </FilterProvider>
  );
}
