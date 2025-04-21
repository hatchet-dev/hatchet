import { createContext, useContext, useCallback, useMemo } from 'react';
import api from '@/lib/api';
import {
  V1TaskSummary,
  V1TriggerWorkflowRunRequest,
  V1WorkflowRunDetails,
  V1TaskStatus,
  V1TaskRunMetrics,
} from '@/lib/api/generated/data-contracts';
import {
  useQuery,
  useMutation,
  UseMutationResult,
} from '@tanstack/react-query';
import useTenant from './use-tenant';
import {
  PaginationProvider,
  PaginationProviderProps,
  usePagination,
} from '@/next/hooks/utils/use-pagination';
import {
  TimeFilterProvider,
  useTimeFilters,
} from '@/next/hooks/utils/use-time-filters';
import { FilterProvider, useFilters } from '@/next/hooks/utils/use-filters';

// Types for filters
export interface RunsFilters {
  statuses?: V1TaskStatus[];
  additional_metadata?: string[];
  workflow_ids?: string[];
  worker_id?: string;
  only_tasks?: boolean;
  parent_task_external_id?: string;
  is_root_task?: boolean;
}

// Create run params
interface CreateRunParams {
  workflowId: string;
  data: V1TriggerWorkflowRunRequest;
}

interface RunsState {
  data: V1TaskSummary[];
  metrics: {
    data: V1TaskRunMetrics;
    isLoading: boolean;
  };
  isLoading: boolean;
  create: UseMutationResult<
    V1WorkflowRunDetails,
    Error,
    CreateRunParams,
    unknown
  >;
  cancel: UseMutationResult<
    unknown,
    Error,
    { tasks: V1TaskSummary[] },
    unknown
  >;
  replay: UseMutationResult<
    unknown,
    Error,
    { tasks: V1TaskSummary[] },
    unknown
  >;
  triggerNow: UseMutationResult<
    V1WorkflowRunDetails,
    Error,
    {
      workflowName: string;
      input: object;
      additionalMetadata: object;
    },
    unknown
  >;
  refetch: () => Promise<unknown>;
  filters: ReturnType<typeof useFilters<RunsFilters>>;
  pagination: ReturnType<typeof usePagination>;
  timeFilter: ReturnType<typeof useTimeFilters>;
}

interface RunsProviderProps {
  children: React.ReactNode;
  initialFilters?: RunsFilters;
  initialPagination?: PaginationProviderProps;
}

const RunsContext = createContext<RunsState | null>(null);

export function useRuns() {
  const context = useContext(RunsContext);
  if (!context) {
    throw new Error('useRuns must be used within a RunsProvider');
  }
  return context;
}

export function RunsProvider({
  children,
  initialFilters,
  initialPagination = {
    initialPageSize: 50,
  },
}: RunsProviderProps) {
  return (
    <FilterProvider initialFilters={initialFilters}>
      <TimeFilterProvider>
        <PaginationProvider {...initialPagination}>
          <RunsProviderContent>{children}</RunsProviderContent>
        </PaginationProvider>
      </TimeFilterProvider>
    </FilterProvider>
  );
}

function RunsProviderContent({ children }: { children: React.ReactNode }) {
  const { tenant } = useTenant();

  const filters = useFilters<RunsFilters>();
  const pagination = usePagination();
  const timeFilter = useTimeFilters();

  const listRunsQuery = useQuery({
    queryKey: [
      'v1:workflow-run:list',
      tenant,
      filters.filters,
      timeFilter.filters,
      pagination,
    ],
    queryFn: async () => {
      if (!tenant) {
        pagination.setNumPages(1);
        return { rows: [], pagination: { current_page: 0, num_pages: 0 } };
      }

      const since =
        timeFilter.filters.createdAfter ||
        new Date(Date.now() - 1000 * 60 * 60 * 24).toISOString();
      const until =
        timeFilter.filters.createdBefore ||
        new Date(Date.now() + 1000 * 60 * 60 * 24).toISOString();

      const query = {
        offset: Math.max(0, (pagination.currentPage - 1) * pagination.pageSize),
        limit: pagination.pageSize,
        since,
        until,
        ...filters.filters,
        only_tasks: !!filters.filters.only_tasks,
      };

      const res = (await api.v1WorkflowRunList(tenant.metadata.id, query)).data;
      pagination.setNumPages(res.pagination?.num_pages || 1);
      return res;
    },
  });

  const metricsRunsQuery = useQuery({
    queryKey: ['v1:workflow-run:metrics', tenant, filters.filters, pagination],
    queryFn: async () => {
      if (!tenant) {
        return [] as V1TaskRunMetrics;
      }

      const since =
        timeFilter.filters.createdAfter ||
        new Date(Date.now() - 1000 * 60 * 60 * 24).toISOString();
      const until =
        timeFilter.filters.createdBefore ||
        new Date(Date.now() + 1000 * 60 * 60 * 24).toISOString();

      const query = {
        offset: Math.max(0, (pagination.currentPage - 1) * pagination.pageSize),
        limit: pagination.pageSize,
        since,
        until,
        ...filters.filters,
        only_tasks: !!filters.filters.only_tasks,
      };

      const res = (
        await api.v1TaskListStatusMetrics(tenant.metadata.id, {
          ...query,
          workflow_ids: filters.filters.workflow_ids,
        })
      ).data;

      return res;
    },
  });

  const createRunMutation = useMutation({
    mutationKey: ['v1:workflow-run:create', tenant],
    mutationFn: async ({ data }: CreateRunParams) => {
      if (!tenant) {
        throw new Error('Tenant not found');
      }
      const res = await api.v1WorkflowRunCreate(tenant.metadata.id, data);
      return res.data;
    },
    onSuccess: () => {
      listRunsQuery.refetch();
    },
  });

  const cancelRunMutation = useMutation({
    mutationKey: ['run:cancel', tenant],
    mutationFn: async ({ tasks }: { tasks: V1TaskSummary[] }) => {
      if (!tenant) {
        throw new Error('Tenant not found');
      }
      const res = await api.v1TaskCancel(tenant.metadata.id, {
        externalIds: tasks.map((run) => run.taskExternalId),
      });
      return res.data;
    },
    onSuccess: () => {
      listRunsQuery.refetch();
    },
  });

  const replayRunMutation = useMutation({
    mutationKey: ['run:replay', tenant],
    mutationFn: async ({ tasks }: { tasks: V1TaskSummary[] }) => {
      if (!tenant) {
        throw new Error('Tenant not found');
      }
      const res = await api.v1TaskReplay(tenant.metadata.id, {
        externalIds: tasks.map((run) => run.taskExternalId),
      });
      return res.data;
    },
    onSuccess: () => {
      listRunsQuery.refetch();
    },
  });

  const triggerNowMutation = useMutation({
    mutationKey: ['workflow-run:create', tenant?.metadata.id],
    mutationFn: async (data: {
      workflowName: string;
      input: object;
      additionalMetadata: object;
    }) => {
      if (!tenant) {
        throw new Error('Tenant not found');
      }
      const res = await api.v1WorkflowRunCreate(tenant.metadata.id, {
        workflowName: data.workflowName,
        input: data.input,
        additionalMetadata: data.additionalMetadata,
      });
      return res.data;
    },
    onSuccess: () => {
      listRunsQuery.refetch();
    },
  });

  const refetch = useCallback(async () => {
    return Promise.all([listRunsQuery.refetch(), metricsRunsQuery.refetch()]);
  }, [listRunsQuery, metricsRunsQuery]);

  const value = useMemo(
    () => ({
      data: listRunsQuery.data?.rows || [],
      metrics: {
        data: metricsRunsQuery.data || [],
        isLoading: metricsRunsQuery.isLoading,
      },
      isLoading: listRunsQuery.isLoading,
      create: createRunMutation,
      cancel: cancelRunMutation,
      replay: replayRunMutation,
      triggerNow: triggerNowMutation,
      refetch,
      filters,
      pagination,
      timeFilter,
    }),
    [
      listRunsQuery.data,
      listRunsQuery.isLoading,
      metricsRunsQuery.data,
      metricsRunsQuery.isLoading,
      createRunMutation,
      cancelRunMutation,
      replayRunMutation,
      triggerNowMutation,
      refetch,
      filters,
      pagination,
      timeFilter,
    ],
  );

  return <RunsContext.Provider value={value}>{children}</RunsContext.Provider>;
}
