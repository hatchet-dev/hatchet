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
import { PaginationManager, PaginationManagerNoOp } from './use-pagination';

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
  pagination?: CronWorkflowsList['pagination'];
  isLoading: boolean;
  update: UseMutationResult<CronWorkflows, Error, UpdateCronParams, unknown>;
  create: UseMutationResult<CronWorkflows, Error, CreateCronParams, unknown>;
  delete: UseMutationResult<void, Error, string, unknown>;
}

interface UseCronsOptions {
  refetchInterval?: number;
  filters?: CronsFilters;
  paginationManager?: PaginationManager;
}

export default function useCrons({
  refetchInterval,
  filters = {},
  paginationManager = PaginationManagerNoOp,
}: UseCronsOptions = {}): CronsState {
  const { tenant } = useTenant();
  const queryClient = useQueryClient();

  const listCronsQuery = useQuery({
    queryKey: ['cron:list', tenant, filters, paginationManager],
    queryFn: async () => {
      if (!tenant) {
        paginationManager?.setNumPages(1);
        return { rows: [], pagination: { current_page: 0, num_pages: 0 } };
      }

      const queryParams: Record<string, any> = {
        limit: paginationManager.pageSize,
        offset:
          (paginationManager.currentPage - 1) * paginationManager.pageSize,
        ...filters,
      };

      const res = await api.cronWorkflowList(
        tenant?.metadata.id || '',
        queryParams,
      );
      paginationManager.setNumPages(res.data.pagination?.num_pages || 1);

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

  return {
    data: listCronsQuery.data?.rows || [],
    pagination: listCronsQuery.data?.pagination,
    isLoading: listCronsQuery.isLoading,
    update: updateCronMutation,
    create: createCronMutation,
    delete: deleteCronMutation,
  };
}

// Context implementation (to maintain compatibility with components)
interface CronsContextType extends CronsState {}

const CronsContext = createContext<CronsContextType | undefined>(undefined);

export const useCronsContext = () => {
  const context = useContext(CronsContext);
  if (context === undefined) {
    throw new Error('useCronsContext must be used within a CronsProvider');
  }
  return context;
};

interface CronsProviderProps extends PropsWithChildren {
  options?: UseCronsOptions;
}

export function CronsProvider(props: CronsProviderProps) {
  const { children, options = {} } = props;
  const cronsState = useCrons(options);

  return createElement(CronsContext.Provider, { value: cronsState }, children);
}
