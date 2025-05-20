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
  useQueryClient,
} from '@tanstack/react-query';
import { useTenant } from './use-tenant';
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
import { useToast } from './utils/use-toast';

// Types for filters
export interface RunsFilters {
  statuses?: V1TaskStatus[];
  additional_metadata?: string[];
  workflow_ids?: string[];
  worker_id?: string;
  only_tasks?: boolean;
  parent_task_external_id?: string;
  is_root_task?: boolean;
  triggering_event_external_id?: string;
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
  isRefetching: boolean;
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
  hasFilters: boolean;
  pagination: ReturnType<typeof usePagination>;
  timeRange: ReturnType<typeof useTimeFilters>;
  histogram: UseQueryResult<V1TaskPointMetrics, Error>;
  queueMetrics: UseQueryResult<TenantStepRunQueueMetrics, Error>;
}

interface RunsProviderProps {
  children: React.ReactNode;
  initialFilters?: RunsFilters;
  initialPagination?: PaginationProviderProps;
  initialTimeRange?: {
    startTime?: string;
    endTime?: string;
    activePreset?: '30m' | '1h' | '6h' | '24h' | '7d';
  };
  refetchInterval?: number;
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
  refetchInterval,
  initialTimeRange,
  initialPagination = {
    initialPageSize: 50,
  },
}: RunsProviderProps) {
  return (
    <FilterProvider initialFilters={initialFilters} type="state">
      <TimeFilterProvider initialTimeRange={initialTimeRange}>
        <PaginationProvider {...initialPagination}>
          <RunsProviderContent refetchInterval={refetchInterval}>
            {children}
          </RunsProviderContent>
        </PaginationProvider>
      </TimeFilterProvider>
    </FilterProvider>
  );
}

function RunsProviderContent({
  children,
  refetchInterval = 1000 * 5,
}: {
  children: React.ReactNode;
  refetchInterval?: number;
}) {
  const queryClient = useQueryClient();
  const { tenantId } = useTenant();
  const { toast } = useToast();

  const filters = useFilters<RunsFilters>();
  const pagination = usePagination();
  const timeRange = useTimeFilters();

  const listRunsQuery = useQuery({
    queryKey: [
      'v1:workflow-run:list',
      tenantId,
      filters.filters,
      timeRange.filters.startTime,
      timeRange.filters.endTime || endOfMinute(new Date()).toISOString(),
      pagination,
    ],
    queryFn: async () => {
      try {
        const since = timeRange.filters.startTime
          ? startOfMinute(new Date(timeRange.filters.startTime)).toISOString()
          : startOfMinute(
              new Date(Date.now() - 1000 * 60 * 60 * 24),
            ).toISOString();
        const until = timeRange.filters.endTime
          ? endOfMinute(new Date(timeRange.filters.endTime)).toISOString()
          : endOfMinute(new Date()).toISOString();

        const query = {
          offset: Math.max(
            0,
            (pagination.currentPage - 1) * pagination.pageSize,
          ),
          limit: pagination.pageSize,
          since,
          until,
          ...filters.filters,
          only_tasks: !!filters.filters.only_tasks,
        };

        const res = (await api.v1WorkflowRunList(tenantId, query)).data;
        pagination.setNumPages(res.pagination?.num_pages || 1);
        return res;
      } catch (error) {
        toast({
          title: 'Error fetching workflow runs',

          variant: 'destructive',
          error,
        });
        return { rows: [], pagination: { current_page: 0, num_pages: 0 } };
      }
    },
    placeholderData: (prev: any) => prev,
    refetchInterval: undefined,
  });

  const metricsRunsQuery = useQuery({
    queryKey: [
      'v1:workflow-run:metrics',
      tenantId,
      filters.filters,
      pagination,
      timeRange.filters.startTime,
      timeRange.filters.endTime || endOfMinute(new Date()).toISOString(),
    ],
    queryFn: async () => {
      try {
        const since =
          timeRange.filters.startTime ||
          startOfMinute(
            new Date(Date.now() - 1000 * 60 * 60 * 24),
          ).toISOString();
        const until =
          timeRange.filters.endTime || endOfMinute(new Date()).toISOString();

        const query = {
          offset: Math.max(
            0,
            (pagination.currentPage - 1) * pagination.pageSize,
          ),
          limit: pagination.pageSize,
          since,
          until,
          ...filters.filters,
          only_tasks: !!filters.filters.only_tasks,
        };

        const res = (
          await api.v1TaskListStatusMetrics(tenantId, {
            ...query,
            workflow_ids: filters.filters.workflow_ids,
          })
        ).data;

        return res;
      } catch (error) {
        toast({
          title: 'Error fetching workflow metrics',

          variant: 'destructive',
          error,
        });
        return [] as V1TaskRunMetrics;
      }
    },
    placeholderData: (prev: any) => prev,
    refetchInterval,
  });

  const histogramQuery = useQuery({
    queryKey: ['v1:workflow-run:metrics', tenantId, timeRange.filters],
    queryFn: async () => {
      try {
        const res = (
          await api.v1TaskGetPointMetrics(tenantId, {
            createdAfter: timeRange.filters.startTime,
            finishedBefore: timeRange.filters.endTime,
          })
        ).data;

        return res;
      } catch (error) {
        toast({
          title: 'Error fetching workflow histogram',

          variant: 'destructive',
          error,
        });
        return [] as V1TaskPointMetrics;
      }
    },
    placeholderData: (prev: any) => prev,
    refetchInterval,
  });

  const queueMetricsQuery = useQuery({
    queryKey: [
      'v1:workflow-run:queue-metrics',
      tenantId,
      filters.filters,
      pagination,
    ],
    queryFn: async () => {
      try {
        const res = (await api.tenantGetStepRunQueueMetrics(tenantId)).data;

        return res;
      } catch (error) {
        toast({
          title: 'Error fetching queue metrics',

          variant: 'destructive',
          error,
        });
        return [] as TenantStepRunQueueMetrics;
      }
    },
    refetchInterval,
  });

  const createRunMutation = useMutation({
    mutationKey: ['v1:workflow-run:create', tenantId],
    mutationFn: async ({ data }: CreateRunParams) => {
      try {
        const res = await api.v1WorkflowRunCreate(tenantId, data);
        return res.data;
      } catch (error) {
        toast({
          title: 'Error creating workflow run',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: () => {
      listRunsQuery.refetch();
    },
  });

  const cancelRunMutation = useMutation({
    mutationKey: ['run:cancel', tenantId],
    mutationFn: async ({ tasks, bulk }: BulkMutation) => {
      try {
        if (tasks) {
          const res = await api.v1TaskCancel(tenantId, {
            externalIds: tasks.map((run) => run.taskExternalId),
          });
          return res.data;
        } else if (bulk) {
          const res = await api.v1TaskCancel(tenantId, {
            filter: {
              ...filters.filters,
              since: timeRange.filters.startTime || new Date().toISOString(),
              until: timeRange.filters.endTime,
            },
          });
          return res.data;
        }
      } catch (error) {
        toast({
          title: 'Error canceling workflow run',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: () => {
      listRunsQuery.refetch();
    },
  });

  const replayRunMutation = useMutation({
    mutationKey: ['run:replay', tenantId],
    mutationFn: async ({ tasks, bulk }: BulkMutation) => {
      try {
        if (tasks) {
          const res = await api.v1TaskReplay(tenantId, {
            externalIds: tasks.map((run) => run.taskExternalId),
          });
          return res.data;
        } else if (bulk) {
          const res = await api.v1TaskReplay(tenantId, {
            filter: {
              ...filters.filters,
              since: timeRange.filters.startTime || new Date().toISOString(),
              until: timeRange.filters.endTime,
            },
          });
          return res.data;
        }
      } catch (error) {
        toast({
          title: 'Error replaying workflow run',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: () => {
      listRunsQuery.refetch();
      queryClient.invalidateQueries({
        queryKey: ['workflow-run-details:*'],
      });
    },
  });

  const triggerNowMutation = useMutation({
    mutationKey: ['workflow-run:create', tenantId],
    mutationFn: async (data: {
      workflowName: string;
      input: object;
      additionalMetadata: object;
    }) => {
      try {
        const res = await api.v1WorkflowRunCreate(tenantId, {
          workflowName: data.workflowName,
          input: data.input,
          additionalMetadata: data.additionalMetadata,
        });
        return res.data;
      } catch (error) {
        toast({
          title: 'Error triggering workflow run',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: () => {
      listRunsQuery.refetch();
      queryClient.invalidateQueries({
        queryKey: ['workflow-run-details:*'],
      });
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

  const hasFilters = useMemo(() => {
    return Object.keys(filters.filters).length > 2;
  }, [filters.filters]);

  const value = useMemo(
    () => ({
      data: listRunsQuery.data?.rows || [],
      count,
      metrics: {
        data: metricsRunsQuery.data || [],
        isLoading: metricsRunsQuery.isLoading,
      },
      isLoading: listRunsQuery.isLoading,
      isRefetching: listRunsQuery.isFetching,
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
      hasFilters,
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
      hasFilters,
    ],
  );

  return <RunsContext.Provider value={value}>{children}</RunsContext.Provider>;
}
