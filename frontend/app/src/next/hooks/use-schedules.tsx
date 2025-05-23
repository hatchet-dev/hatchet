import api, {
  ScheduledWorkflows,
  ScheduledWorkflowsList,
  ScheduleWorkflowRunRequest,
  ScheduledRunStatus,
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
export interface SchedulesFilters {
  statuses?: ScheduledRunStatus[];
  workflowId?: string;
  parentWorkflowRunId?: string;
  parentStepRunId?: string;
  additionalMetadata?: string[];
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
  paginationData?: ScheduledWorkflowsList['pagination'];
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
  filters: ReturnType<typeof useFilters<SchedulesFilters>>;
  pagination: ReturnType<typeof usePagination>;
}

interface SchedulesProviderProps extends PropsWithChildren {
  refetchInterval?: number;
}

const SchedulesContext = createContext<SchedulesState | null>(null);

export function useSchedules() {
  const context = useContext(SchedulesContext);
  if (!context) {
    throw new Error('useSchedules must be used within a SchedulesProvider');
  }
  return context;
}

function SchedulesProviderContent({
  children,
  refetchInterval,
}: SchedulesProviderProps) {
  const { tenantId } = useCurrentTenantId();
  const queryClient = useQueryClient();
  const filters = useFilters<SchedulesFilters>();
  const pagination = usePagination();
  const { toast } = useToast();

  const listSchedulesQuery = useQuery({
    queryKey: ['schedule:list', tenantId, filters.filters, pagination],
    queryFn: async () => {
      try {
        // Build query params
        const queryParams: Parameters<typeof api.workflowScheduledList>[1] = {
          limit: pagination.pageSize,
          offset: Math.max(
            0,
            (pagination.currentPage - 1) * pagination.pageSize,
          ),
          ...filters.filters,
        };

        const res = await api.workflowScheduledList(tenantId, queryParams);

        return res.data;
      } catch (error) {
        toast({
          title: 'Error fetching schedules',

          variant: 'destructive',
          error,
        });
        return { rows: [], pagination: { current_page: 0, num_pages: 0 } };
      }
    },
    refetchInterval,
  });

  // Create implementation
  const createScheduleMutation = useMutation({
    mutationKey: ['schedule:create', tenantId],
    mutationFn: async ({ workflowName, data }: CreateScheduleParams) => {
      try {
        const res = await api.scheduledWorkflowRunCreate(
          tenantId,
          workflowName,
          data,
        );

        return res.data;
      } catch (error) {
        toast({
          title: 'Error creating schedule',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['schedule:list'] });
    },
  });

  // Delete implementation
  const deleteScheduleMutation = useMutation({
    mutationKey: ['schedule:delete', tenantId],
    mutationFn: async (scheduleId: string) => {
      try {
        await api.workflowScheduledDelete(tenantId, scheduleId);
      } catch (error) {
        toast({
          title: 'Error deleting schedule',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: () => {
      listSchedulesQuery.refetch();
    },
  });

  // Update implementation that deletes and recreates the scheduled workflow
  const updateScheduleMutation = useMutation({
    mutationKey: ['schedule:update', tenantId],
    mutationFn: async ({
      scheduleId,
      workflowId,
      data,
    }: UpdateScheduleParams) => {
      try {
        // First delete the existing schedule
        await api.workflowScheduledDelete(tenantId, scheduleId);

        // Then create a new one with the updated data
        const res = await api.scheduledWorkflowRunCreate(
          tenantId,
          workflowId,
          data,
        );

        return res.data;
      } catch (error) {
        toast({
          title: 'Error updating schedule',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: () => {
      listSchedulesQuery.refetch();
    },
  });

  const value = {
    data: listSchedulesQuery.data?.rows || [],
    paginationData: listSchedulesQuery.data?.pagination,
    isLoading: listSchedulesQuery.isLoading,
    update: updateScheduleMutation,
    create: createScheduleMutation,
    delete: deleteScheduleMutation,
    filters,
    pagination,
  };

  return createElement(SchedulesContext.Provider, { value }, children);
}

export function SchedulesProvider({
  children,
  refetchInterval,
}: SchedulesProviderProps) {
  return (
    <FilterProvider<SchedulesFilters>
      initialFilters={{
        statuses: [],
        workflowId: undefined,
        parentWorkflowRunId: undefined,
        parentStepRunId: undefined,
        additionalMetadata: [],
      }}
    >
      <PaginationProvider initialPage={1} initialPageSize={50}>
        <SchedulesProviderContent refetchInterval={refetchInterval}>
          {children}
        </SchedulesProviderContent>
      </PaginationProvider>
    </FilterProvider>
  );
}
