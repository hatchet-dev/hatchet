import { createContext, useContext, useCallback, useMemo } from 'react';
import api from '@/lib/api';
import {
  V1TaskSummary,
  V1TriggerWorkflowRunRequest,
  V1WorkflowRunDetails,
  V1TaskStatus,
  V1TaskRunMetrics,
  V1TaskPointMetrics,
  TenantStepRunQueueMetrics,
} from '@/lib/api/generated/data-contracts';
import {
  useQuery,
  useMutation,
  UseMutationResult,
  UseQueryResult,
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
import { endOfMinute, startOfMinute } from 'date-fns';

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

type BulkMutation =
  | {
      bulk?: never;
      tasks: V1TaskSummary[];
    }
  | {
      bulk: boolean;
      tasks?: never;
    };

interface RunsState {
  data: V1TaskSummary[];
  count: number;
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
  cancel: UseMutationResult<unknown, Error, BulkMutation, unknown>;
  replay: UseMutationResult<unknown, Error, BulkMutation, unknown>;
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
  timeRange: ReturnType<typeof useTimeFilters>;
  histogram: UseQueryResult<V1TaskPointMetrics, Error>;
  queueMetrics: UseQueryResult<TenantStepRunQueueMetrics, Error>;
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
  const timeRange = useTimeFilters();
  const refetchInterval = 1000 * 5; // 5 seconds

  const listRunsQuery = useQuery({
    queryKey: [
      'v1:workflow-run:list',
      tenant,
      filters.filters,
      timeRange.filters.startTime,
      timeRange.filters.endTime || endOfMinute(new Date()).toISOString(),
      pagination,
    ],
    queryFn: async () => {
      if (!tenant) {
        pagination.setNumPages(1);
        return { rows: [], pagination: { current_page: 0, num_pages: 0 } };
      }

      const since = timeRange.filters.startTime
        ? startOfMinute(new Date(timeRange.filters.startTime)).toISOString()
        : startOfMinute(
            new Date(Date.now() - 1000 * 60 * 60 * 24),
          ).toISOString();
      const until = timeRange.filters.endTime
        ? endOfMinute(new Date(timeRange.filters.endTime)).toISOString()
        : endOfMinute(new Date()).toISOString();

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
    placeholderData: (prev: any) => prev,
    refetchInterval,
  });

  const metricsRunsQuery = useQuery({
    queryKey: [
      'v1:workflow-run:metrics',
      tenant,
      filters.filters,
      pagination,
      timeRange.filters.startTime,
      timeRange.filters.endTime || endOfMinute(new Date()).toISOString(),
    ],
    queryFn: async () => {
      if (!tenant) {
        return [] as V1TaskRunMetrics;
      }

      const since =
        timeRange.filters.startTime ||
        startOfMinute(new Date(Date.now() - 1000 * 60 * 60 * 24)).toISOString();
      const until =
        timeRange.filters.endTime || endOfMinute(new Date()).toISOString();

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
    placeholderData: (prev: any) => prev,
    refetchInterval,
  });

  const histogramQuery = useQuery({
    queryKey: ['v1:workflow-run:metrics', tenant, timeRange.filters],
    queryFn: async () => {
      if (!tenant) {
        return [] as V1TaskPointMetrics;
      }

      const res = (
        await api.v1TaskGetPointMetrics(tenant.metadata.id, {
          createdAfter: timeRange.filters.startTime,
          finishedBefore: timeRange.filters.endTime, // TODO: THIS ISN'T CORRECT
        })
      ).data;

      return res;
    },
    placeholderData: (prev: any) => prev,
    enabled: !!tenant?.metadata.id,
    refetchInterval,
  });

  const queueMetricsQuery = useQuery({
    queryKey: [
      'v1:workflow-run:queue-metrics',
      tenant,
      filters.filters,
      pagination,
    ],
    queryFn: async () => {
      if (!tenant) {
        return [] as TenantStepRunQueueMetrics;
      }

      const res = (await api.tenantGetStepRunQueueMetrics(tenant.metadata.id))
        .data;

      return res;
    },
    refetchInterval,
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
    mutationFn: async ({ tasks, bulk }: BulkMutation) => {
      if (!tenant) {
        throw new Error('Tenant not found');
      }

      if (tasks) {
        const res = await api.v1TaskCancel(tenant.metadata.id, {
          externalIds: tasks.map((run) => run.taskExternalId),
        });
        return res.data;
      } else if (bulk) {
        const res = await api.v1TaskCancel(tenant.metadata.id, {
          filter: {
            ...filters.filters,
            since: timeRange.filters.startTime || new Date().toISOString(),
            until: timeRange.filters.endTime,
          },
        });
        return res.data;
      }
    },
    onSuccess: () => {
      listRunsQuery.refetch();
    },
  });

  const replayRunMutation = useMutation({
    mutationKey: ['run:replay', tenant],
    mutationFn: async ({ tasks, bulk }: BulkMutation) => {
      if (!tenant) {
        throw new Error('Tenant not found');
      }

      if (tasks) {
        const res = await api.v1TaskReplay(tenant.metadata.id, {
          externalIds: tasks.map((run) => run.taskExternalId),
        });
        return res.data;
      } else if (bulk) {
        const res = await api.v1TaskReplay(tenant.metadata.id, {
          filter: {
            ...filters.filters,
            since: timeRange.filters.startTime || new Date().toISOString(),
            until: timeRange.filters.endTime,
          },
        });
        return res.data;
      }
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
    return Promise.all([
      listRunsQuery.refetch(),
      metricsRunsQuery.refetch(),
      histogramQuery.refetch(),
    ]);
  }, [listRunsQuery, metricsRunsQuery, histogramQuery]);

  const count = useMemo(() => {
    // TODO this is returning an inconsistent count with the number of runs in the table
    return (
      metricsRunsQuery.data
        ?.filter(
          (metric) =>
            (metric.status && !filters.filters.statuses) ||
            filters.filters.statuses?.includes(metric.status),
        )
        .reduce((acc, metric) => acc + metric.count, 0) || 0
    );
  }, [metricsRunsQuery.data, filters.filters.statuses]);

  const value = useMemo(
    () => ({
      data: listRunsQuery.data?.rows || [],
      count,
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
      timeRange,
      histogram: histogramQuery,
      queueMetrics: queueMetricsQuery,
    }),
    [
      listRunsQuery.data?.rows,
      listRunsQuery.isLoading,
      count,
      metricsRunsQuery.data,
      metricsRunsQuery.isLoading,
      createRunMutation,
      cancelRunMutation,
      replayRunMutation,
      triggerNowMutation,
      refetch,
      filters,
      pagination,
      timeRange,
      histogramQuery,
      queueMetricsQuery,
    ],
  );

  return <RunsContext.Provider value={value}>{children}</RunsContext.Provider>;
}
