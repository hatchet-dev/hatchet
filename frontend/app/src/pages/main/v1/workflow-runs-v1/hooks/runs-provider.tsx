import { createContext, useContext, useMemo, useState } from 'react';
import {
  getWorkflowIdsFromFilters,
  RunsTableState,
  useRunsTableState,
} from './use-runs-table-state';
import { useRunsTableFilters } from './use-runs-table-filters';
import { useToolbarFilters } from './use-toolbar-filters';
import { useRuns } from './use-runs';
import { useMetrics } from './use-metrics';
import { TaskRunColumn } from '../components/v1/task-runs-columns';
import { V1TaskRunMetrics, V1TaskSummary } from '@/lib/api';
import { PaginationState } from '@tanstack/react-table';

type DisplayProps = {
  showMetrics: boolean;
  showCounts: boolean;
  showDateFilter: boolean;
  showTriggerRunButton: boolean;
  showCancelAndReplayButtons: boolean;
  showColumnToggle: boolean;
};

type RunFilteringProps = {
  workflowId?: string;
  parentTaskExternalId?: string;
  workerId?: string;
  triggeringEventExternalId?: string;
};

type RunsProviderProps = {
  tableKey: string;
  children: React.ReactNode;
  disableTaskRunPagination?: boolean;
  initColumnVisibility?: Record<string, boolean>;
  filterVisibility?: Record<string, boolean>;
  refetchInterval?: number;
  display: DisplayProps;
  runFilters: RunFilteringProps;
};

type RunsContextType = {
  state: RunsTableState;
  actions: {
    updatePagination: (pagination: PaginationState) => void;
    updateFilters: (filters: any) => void;
    updateUIState: (
      uiState: Partial<
        Pick<
          RunsTableState,
          'viewQueueMetrics' | 'triggerWorkflow' | 'taskRunDetailSheet'
        >
      >,
    ) => void;
    updateTableState: (
      tableState: Partial<
        Pick<RunsTableState, 'rowSelection' | 'columnVisibility'>
      >,
    ) => void;
    resetState: () => void;
    setIsFrozen: (isFrozen: boolean) => void;
    refetchRuns: () => void;
    refetchMetrics: () => void;
    getRowId: (row: V1TaskSummary) => string;
  };
  filters: ReturnType<typeof useRunsTableFilters>;
  toolbarFilters: ReturnType<typeof useToolbarFilters>;
  tableRows: V1TaskSummary[];
  selectedRuns: V1TaskSummary[];
  numPages: number;
  isRunsLoading: boolean;
  isRunsFetching: boolean;
  isMetricsLoading: boolean;
  isMetricsFetching: boolean;
  metrics: V1TaskRunMetrics;
  tenantMetrics: object;
  isFrozen: boolean;
  display: {
    showMetrics: boolean;
    showCounts: boolean;
    showDateFilter: boolean;
    showTriggerRunButton: boolean;
    showCancelAndReplayButtons: boolean;
    showColumnToggle: boolean;
    showPagination: boolean;
    refetchInterval: number;
  };
};

const RunsContext = createContext<RunsContextType | null>(null);

export const RunsProvider = ({
  tableKey,
  children,
  disableTaskRunPagination = false,
  initColumnVisibility = {},
  filterVisibility = {},
  refetchInterval = 5000,
  display,
  runFilters,
}: RunsProviderProps) => {
  const {
    workflowId,
    parentTaskExternalId,
    workerId,
    triggeringEventExternalId,
  } = runFilters;

  const {
    showMetrics = false,
    showCounts = true,
    showDateFilter = true,
    showTriggerRunButton = true,
    showCancelAndReplayButtons = true,
    showColumnToggle = true,
  } = display;
  const [isFrozen, setIsFrozen] = useState(false);

  const initialState = useMemo(() => {
    const baseState: Partial<RunsTableState> = {
      columnVisibility: {
        ...initColumnVisibility,
        parentTaskExternalId: false, // Always hidden, used for filtering only
      },
    };

    if (workflowId) {
      baseState.columnFilters = [
        { id: TaskRunColumn.workflow, value: workflowId },
      ];
    }

    if (parentTaskExternalId) {
      baseState.parentTaskExternalId = parentTaskExternalId;
    }

    return baseState;
  }, [workflowId, parentTaskExternalId, initColumnVisibility]);

  const {
    state,
    updatePagination,
    updateFilters,
    updateUIState,
    updateTableState,
    resetState,
  } = useRunsTableState(tableKey, initialState);

  const filters = useRunsTableFilters(state, updateFilters);

  const toolbarFilters = useToolbarFilters({ filterVisibility });

  const workflow =
    workflowId || getWorkflowIdsFromFilters(state.columnFilters)[0];
  const derivedParentTaskExternalId =
    parentTaskExternalId || state.parentTaskExternalId;

  const {
    tableRows,
    selectedRuns,
    numPages,
    isLoading: isRunsLoading,
    isFetching: isRunsFetching,
    refetch: refetchRuns,
    getRowId,
  } = useRuns({
    rowSelection: state.rowSelection,
    pagination: state.pagination,
    createdAfter: state.createdAfter,
    finishedBefore: state.finishedBefore,
    statuses: filters.apiFilters.statuses,
    additionalMetadata: filters.apiFilters.additionalMetadata,
    workerId,
    workflowIds:
      filters.apiFilters.workflowIds || (workflow ? [workflow] : undefined),
    parentTaskExternalId: derivedParentTaskExternalId,
    triggeringEventExternalId,
    disablePagination: disableTaskRunPagination,
    pauseRefetch: state.hasOpenUI || isFrozen,
  });

  const {
    metrics,
    tenantMetrics,
    isLoading: isMetricsLoading,
    isFetching: isMetricsFetching,
    refetch: refetchMetrics,
  } = useMetrics({
    workflow,
    parentTaskExternalId: derivedParentTaskExternalId,
    createdAfter: state.createdAfter,
    refetchInterval,
    pauseRefetch: state.hasOpenUI || isFrozen,
  });

  const value = useMemo<RunsContextType>(
    () => ({
      state,
      filters,
      toolbarFilters,
      tableRows,
      selectedRuns,
      numPages,
      isRunsLoading,
      isRunsFetching,
      isMetricsLoading,
      isMetricsFetching,
      metrics,
      tenantMetrics,
      isFrozen,
      display: {
        showMetrics,
        showCounts,
        showDateFilter,
        showTriggerRunButton,
        showCancelAndReplayButtons,
        showColumnToggle,
        showPagination: !disableTaskRunPagination,
        refetchInterval,
      },
      actions: {
        updatePagination,
        updateFilters,
        updateUIState,
        updateTableState,
        resetState,
        setIsFrozen,
        refetchRuns,
        refetchMetrics,
        getRowId,
      },
    }),
    [
      state,
      filters,
      toolbarFilters,
      tableRows,
      selectedRuns,
      numPages,
      isRunsLoading,
      isRunsFetching,
      isMetricsLoading,
      isMetricsFetching,
      metrics,
      tenantMetrics,
      isFrozen,
      showMetrics,
      showCounts,
      showDateFilter,
      showTriggerRunButton,
      refetchInterval,
      updatePagination,
      updateFilters,
      updateUIState,
      updateTableState,
      resetState,
      setIsFrozen,
      refetchRuns,
      refetchMetrics,
      getRowId,
      showCancelAndReplayButtons,
      showColumnToggle,
      disableTaskRunPagination,
    ],
  );

  return <RunsContext.Provider value={value}>{children}</RunsContext.Provider>;
};

export const useRunsContext = () => {
  const context = useContext(RunsContext);

  if (!context) {
    throw new Error('useRunsContext must be used within a RunsProvider');
  }

  return context;
};
