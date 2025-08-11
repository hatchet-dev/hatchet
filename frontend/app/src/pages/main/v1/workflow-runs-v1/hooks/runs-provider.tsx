import { createContext, useContext, useMemo, useState } from 'react';
import {
  getWorkflowIdFromFilters,
  RunsTableState,
  useRunsTableState,
} from './use-runs-table-state';
import { useRunsTableFilters } from './use-runs-table-filters';
import { useToolbarFilters } from './use-toolbar-filters';
import { useRuns } from './use-runs';
import { useMetrics } from './use-metrics';
import { TaskRunColumn } from '../components/v1/task-runs-columns';
import { V1TaskSummary } from '@/lib/api';
import { PaginationState } from '@tanstack/react-table';

type ExtendedRunsTableState = RunsTableState & {
  hasRowsSelected: boolean;
  hasFiltersApplied: boolean;
  hasOpenUI: boolean;
};

type RunsProviderProps = {
  tableKey: string;
  children: React.ReactNode;
  workflowId?: string;
  parentTaskExternalId?: string;
  disableTaskRunPagination?: boolean;
  initColumnVisibility?: Record<string, boolean>;
  filterVisibility?: Record<string, boolean>;
  refetchInterval?: number;
  workerId?: string;
  triggeringEventExternalId?: string;
};

type RunsContextType = {
  state: ExtendedRunsTableState;
  actions: {
    updatePagination: (pagination: PaginationState) => void;
    updateFilters: (filters: any) => void;
    updateUIState: (
      uiState: Partial<
        Pick<
          RunsTableState,
          | 'selectedAdditionalMetaRunId'
          | 'viewQueueMetrics'
          | 'triggerWorkflow'
          | 'stepDetailSheet'
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
  metrics: any;
  tenantMetrics: any;
  isFrozen: boolean;
};

const RunsContext = createContext<RunsContextType | null>(null);

export const RunsProvider = ({
  tableKey,
  workflowId,
  parentTaskExternalId,
  disableTaskRunPagination = false,
  initColumnVisibility = {},
  filterVisibility = {},
  refetchInterval = 5000,
  workerId,
  triggeringEventExternalId,
  children,
}: RunsProviderProps) => {
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

  const workflow = workflowId || getWorkflowIdFromFilters(state.columnFilters);
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
    status: filters.apiFilters.statuses?.[0],
    additionalMetadata: filters.apiFilters.additionalMetadata,
    workerId,
    workflow,
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
      updatePagination,
      updateFilters,
      updateUIState,
      updateTableState,
      resetState,
      setIsFrozen,
      refetchRuns,
      refetchMetrics,
      getRowId,
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
