import api, {
  ScheduledWorkflows,
  ScheduledWorkflowsList,
  ScheduleWorkflowRunRequest,
  ScheduledRunStatus,
  ScheduledWorkflowsOrderByField,
  WorkflowRunOrderByDirection,
} from '@/next/lib/api';
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
import {
  PaginationManagerNoOp,
  PaginationManager,
} from '@/next/components/ui/pagination';

// Types for filters and pagination
export interface SchedulesFilters {
  statuses?: ScheduledRunStatus[];
  workflowId?: string;
  parentWorkflowRunId?: string;
  parentStepRunId?: string;
  additionalMetadata?: string[];
}

export interface SchedulesSort {
  sortBy?: ScheduledWorkflowsOrderByField;
  sortDirection?: WorkflowRunOrderByDirection;
}

// Update schedule params
interface UpdateScheduleParams {
  scheduleId: string;
  workflowId: string;
  data: ScheduleWorkflowRunRequest;
}

// Create schedule params
interface CreateScheduleParams {
  workflowName: string;
  data: ScheduleWorkflowRunRequest;
}

// Main hook return type
interface SchedulesState {
  data?: ScheduledWorkflowsList['rows'];
  paginationResponse?: ScheduledWorkflowsList['pagination'];
  isLoading: boolean;
  update: UseMutationResult<
    ScheduledWorkflows,
    Error,
    UpdateScheduleParams,
    unknown
  >;
  create: UseMutationResult<
    ScheduledWorkflows,
    Error,
    CreateScheduleParams,
    unknown
  >;
  delete: UseMutationResult<void, Error, string, unknown>;

  // Filters state
  filters: SchedulesFilters;
}

interface UseSchedulesOptions {
  refetchInterval?: number;
  filters?: SchedulesFilters;
  sort?: SchedulesSort;
  paginationManager?: PaginationManager;
}

export default function useSchedules({
  refetchInterval,
  filters = {},
  sort = {},
  paginationManager: pagination = PaginationManagerNoOp,
}: UseSchedulesOptions = {}): SchedulesState {
  const { tenant } = useTenant();
  const queryClient = useQueryClient();

  // State for filters only

  const listSchedulesQuery = useQuery({
    queryKey: ['schedule:list', tenant, filters, sort, pagination],
    queryFn: async () => {
      if (!tenant) {
        const p = {
          rows: [],
          pagination: { current_page: 0, num_pages: 0 },
        };
        pagination?.setNumPages(p.pagination.num_pages);
        return p;
      }

      // Build query params
      const queryParams: Parameters<typeof api.workflowScheduledList>[1] = {
        limit: pagination?.pageSize || 10,
        offset: (pagination?.currentPage - 1) * pagination?.pageSize || 0,
        ...filters,
      };

      if (sort.sortBy) {
        queryParams.orderByField = sort.sortBy;
        queryParams.orderByDirection =
          sort.sortDirection || WorkflowRunOrderByDirection.ASC;
      }

      const res = await api.workflowScheduledList(
        tenant?.metadata.id || '',
        queryParams,
      );

      pagination?.setNumPages(res.data.pagination?.num_pages || 1);

      return res.data;
    },
    refetchInterval,
  });

  // Create implementation
  const createScheduleMutation = useMutation({
    mutationKey: ['schedule:create', tenant],
    mutationFn: async ({ workflowName, data }: CreateScheduleParams) => {
      if (!tenant) {
        throw new Error('Tenant not found');
      }

      const res = await api.scheduledWorkflowRunCreate(
        tenant.metadata.id,
        workflowName,
        data,
      );

      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['schedule:list'] });
    },
  });

  // Delete implementation
  const deleteScheduleMutation = useMutation({
    mutationKey: ['schedule:delete', tenant],
    mutationFn: async (scheduleId: string) => {
      if (!tenant) {
        throw new Error('Tenant not found');
      }

      await api.workflowScheduledDelete(tenant.metadata.id, scheduleId);
    },
    onSuccess: () => {
      listSchedulesQuery.refetch();
    },
  });

  // Update implementation that deletes and recreates the scheduled workflow
  const updateScheduleMutation = useMutation({
    mutationKey: ['schedule:update', tenant],
    mutationFn: async ({
      scheduleId,
      workflowId,
      data,
    }: UpdateScheduleParams) => {
      if (!tenant) {
        throw new Error('Tenant not found');
      }

      // First delete the existing schedule
      await api.workflowScheduledDelete(tenant.metadata.id, scheduleId);

      // Then create a new one with the updated data
      const res = await api.scheduledWorkflowRunCreate(
        tenant.metadata.id,
        workflowId,
        data,
      );

      return res.data;
    },
    onSuccess: () => {
      listSchedulesQuery.refetch();
    },
  });

  return {
    data: listSchedulesQuery.data?.rows || [],
    paginationResponse: listSchedulesQuery.data?.pagination,
    isLoading: listSchedulesQuery.isLoading,
    update: updateScheduleMutation,
    create: createScheduleMutation,
    delete: deleteScheduleMutation,
    filters,
  };
}

// Context implementation (to maintain compatibility with components)
interface SchedulesContextType extends SchedulesState {}

const SchedulesContext = createContext<SchedulesContextType | undefined>(
  undefined,
);

export const useSchedulesContext = () => {
  const context = useContext(SchedulesContext);
  if (context === undefined) {
    throw new Error(
      'useSchedulesContext must be used within a SchedulesProvider',
    );
  }
  return context;
};

interface SchedulesProviderProps extends PropsWithChildren {
  options?: UseSchedulesOptions;
}

export function SchedulesProvider(props: SchedulesProviderProps) {
  const { children, options = {} } = props;
  const schedulesState = useSchedules(options);

  return createElement(
    SchedulesContext.Provider,
    { value: schedulesState },
    children,
  );
}
