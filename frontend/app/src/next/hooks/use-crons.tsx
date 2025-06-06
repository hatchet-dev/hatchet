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
  const { tenantId } = useCurrentTenantId();
  const queryClient = useQueryClient();
  const filters = useFilters<CronsFilters>();
  const pagination = usePagination();
  const { toast } = useToast();

  const listCronsQuery = useQuery({
    queryKey: ['cron:list', tenantId, filters.filters, pagination],
    queryFn: async () => {
      try {
        const queryParams: Record<string, number | string | string[]> = {
          limit: pagination.pageSize,
          offset: Math.max(
            0,
            (pagination.currentPage - 1) * pagination.pageSize,
          ),
          ...filters.filters,
        };

        const res = await api.cronWorkflowList(tenantId, queryParams);

        return res.data;
      } catch (error) {
        toast({
          title: 'Error fetching crons',

          variant: 'destructive',
          error,
        });
        return { rows: [], pagination: { current_page: 0, num_pages: 0 } };
      }
    },
    refetchInterval,
  });

  // Create implementation
  const createCronMutation = useMutation({
    mutationKey: ['cron:create', tenantId],
    mutationFn: async ({ workflowId, data }: CreateCronParams) => {
      try {
        const res = await api.cronWorkflowTriggerCreate(
          tenantId,
          workflowId,
          data,
        );

        return res.data;
      } catch (error) {
        toast({
          title: 'Error creating cron',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['cron:list'] });
    },
  });

  // Delete implementation
  const deleteCronMutation = useMutation({
    mutationKey: ['cron:delete', tenantId],
    mutationFn: async (cronId: string) => {
      try {
        await api.workflowCronDelete(tenantId, cronId);
      } catch (error) {
        toast({
          title: 'Error deleting cron',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: async () => {
      await listCronsQuery.refetch();
    },
  });

  const updateCronMutation = useMutation({
    mutationKey: ['cron:update', tenantId],
    mutationFn: async ({ cronId, workflowId, data }: UpdateCronParams) => {
      try {
        // First delete the existing cron
        await api.workflowCronDelete(tenantId, cronId);

        // Then create a new one with the updated data
        const res = await api.cronWorkflowTriggerCreate(
          tenantId,
          workflowId,
          data,
        );

        return res.data;
      } catch (error) {
        toast({
          title: 'Error updating cron',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: async () => {
      await listCronsQuery.refetch();
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
