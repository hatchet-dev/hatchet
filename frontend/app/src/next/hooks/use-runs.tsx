import {
  createContext,
  useContext,
  createElement,
  PropsWithChildren,
} from 'react';
import api from '../lib/api';
import {
  V1TaskSummary,
  V1TaskSummaryList,
  V1TriggerWorkflowRunRequest,
  V1WorkflowRunDetails,
  V1TaskStatus,
  V1TaskRunMetrics,
} from '../lib/api/generated/data-contracts';
import {
  useQuery,
  useMutation,
  UseMutationResult,
} from '@tanstack/react-query';
import useTenant from './use-tenant';
import { PaginationManager, PaginationManagerNoOp } from './use-pagination';

// Define the RunQuery type to match the API parameter structure
type RunQuery = Parameters<typeof api.v1WorkflowRunList>[1];

// Types for filters and pagination
export interface RunsFilters {
  createdAfter?: string;
  createdBefore?: string;
  statuses?: V1TaskStatus[];
  additional_metadata?: string[];
  workflows_ids?: string[];
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
  pagination?: V1TaskSummaryList['pagination'];
  isLoading: boolean;
  create: UseMutationResult<
    V1WorkflowRunDetails,
    Error,
    CreateRunParams,
    unknown
  >;

  filters: RunsFilters;
  refetch: () => Promise<unknown>;
}

interface RunsProviderProps {
  filters?: RunsFilters;
  pagination?: PaginationManager;
  refetchInterval?: number;
}

const RunsContext = createContext<RunsState | null>(null);

export function useRuns({
  filters = {},
  pagination = PaginationManagerNoOp,
  refetchInterval,
}: RunsProviderProps) {
  const { tenant } = useTenant();

  const listRunsQuery = useQuery({
    queryKey: ['v1:workflow-run:list', tenant, filters, pagination],
    queryFn: async () => {
      if (!tenant) {
        pagination?.setNumPages(1);
        return { rows: [], pagination: { current_page: 0, num_pages: 0 } };
      }

      // TODO: createdAfter should always be set, and rename this
      const since =
        filters.createdAfter ||
        new Date(Date.now() - 1000 * 60 * 60 * 24).toISOString();
      const until =
        filters.createdBefore ||
        new Date(Date.now() + 1000 * 60 * 60 * 24).toISOString();

      // Convert pagination and filters to API query format
      const query: RunQuery = {
        offset: Math.max(0, (pagination.currentPage - 1) * pagination.pageSize),
        limit: pagination.pageSize,
        since: since,
        until: until,
        ...filters,
        only_tasks: !!filters.only_tasks,
      };

      const res = (await api.v1WorkflowRunList(tenant.metadata.id, query)).data;

      pagination?.setNumPages(res.pagination?.num_pages || 1);

      return res;
    },
    refetchInterval,
  });

  const metricsRunsQuery = useQuery({
    queryKey: ['v1:workflow-run:metrics', tenant, filters, pagination],
    queryFn: async () => {
      if (!tenant) {
        return [] as V1TaskRunMetrics;
      }

      // TODO: createdAfter should always be set, and rename this
      const since =
        filters.createdAfter ||
        new Date(Date.now() - 1000 * 60 * 60 * 24).toISOString();
      const until =
        filters.createdBefore ||
        new Date(Date.now() + 1000 * 60 * 60 * 24).toISOString();

      // Convert pagination and filters to API query format
      const query: RunQuery = {
        offset: Math.max(0, (pagination.currentPage - 1) * pagination.pageSize),
        limit: pagination.pageSize,
        since: since,
        until: until,
        ...filters,
        only_tasks: !!filters.only_tasks,
      };
      // cloudApi.workflowRunEventsGetMetrics
      const res = (
        await api.v1TaskListStatusMetrics(tenant.metadata.id, {
          ...query,
          workflow_ids: filters.workflows_ids,
        })
      ).data;

      return res;
    },
  });
  // Create workflow run implementation
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

  return {
    data: listRunsQuery.data?.rows || [],
    metrics: {
      data: metricsRunsQuery.data || [],
      isLoading: metricsRunsQuery.isLoading,
    },
    pagination: listRunsQuery.data?.pagination,
    isLoading: listRunsQuery.isLoading,
    create: createRunMutation,
    cancel: cancelRunMutation,
    replay: replayRunMutation,
    triggerNow: triggerNowMutation,
    refetch: async () => {
      return Promise.all([listRunsQuery.refetch(), metricsRunsQuery.refetch()]);
    },

    // Filter state management
    filters,
  };
}

export function useRunsContext(): RunsState {
  const context = useContext(RunsContext);
  if (!context) {
    throw new Error('useRuns must be used within a RunsProvider');
  }
  return context;
}

export function RunsProvider({
  children,
  refetchInterval,
  ...props
}: RunsProviderProps & PropsWithChildren) {
  const runsState = useRuns({
    refetchInterval,
    ...props,
  });

  return createElement(
    RunsContext.Provider,
    {
      value: runsState,
    },
    children,
  );
}
