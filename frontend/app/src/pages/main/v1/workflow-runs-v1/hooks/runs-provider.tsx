import { createContext, useContext, useMemo, useState } from 'react';
import {
  getFlattenDAGsFromFilters,
  getWorkflowIdsFromFilters,
  RunsTableState,
  useRunsTableState,
} from './use-runs-table-state';
import { useRunsTableFilters } from './use-runs-table-filters';
import { useToolbarFilters } from './use-toolbar-filters';
import { useRuns } from './use-runs';
import { useMetrics } from './use-metrics';
import { workflowKey } from '../components/v1/task-runs-columns';
import { V1TaskRunMetrics, V1TaskSummary } from '@/lib/api';
import { PaginationState } from '@tanstack/react-table';
import {
  ActionType,
  BaseTaskRunActionParams,
} from '../../task-runs-v1/actions';
import { LabeledRefetchInterval, RefetchInterval } from '@/lib/api/api';

type DisplayProps = {
  hideMetrics?: boolean;
  hideCounts?: boolean;
  hideDateFilter?: boolean;
  hideTriggerRunButton?: boolean;
  hideCancelAndReplayButtons?: boolean;
  hideColumnToggle?: boolean;
  hideFlatten?: boolean;
  hidePagination?: boolean;
  refetchInterval: LabeledRefetchInterval;
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
  refetchInterval?: LabeledRefetchInterval;
  display?: DisplayProps;
  runFilters?: RunFilteringProps;
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
    setIsActionModalOpen: (isOpen: boolean) => void;
    setIsActionDropdownOpen: (isOpen: boolean) => void;
    setSelectedActionType: (actionType: ActionType | null) => void;
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
  isActionModalOpen: boolean;
  isActionDropdownOpen: boolean;
  selectedActionType: ActionType | null;
  actionModalParams: BaseTaskRunActionParams;
  display: DisplayProps;
};

const RunsContext = createContext<RunsContextType | null>(null);

export const RunsProvider = ({
  tableKey,
  children,
  disableTaskRunPagination = false,
  initColumnVisibility = {},
  filterVisibility = {},
  refetchInterval = RefetchInterval.off,
  display,
  runFilters,
}: RunsProviderProps) => {
  const [isFrozen, setIsFrozen] = useState(false);
  const [isActionModalOpen, setIsActionModalOpen] = useState(false);
  const [isActionDropdownOpen, setIsActionDropdownOpen] = useState(false);
  const [selectedActionType, setSelectedActionType] =
    useState<ActionType | null>(null);

  const {
    workflowId,
    parentTaskExternalId,
    workerId,
    triggeringEventExternalId,
  } = runFilters ?? {};

  const {
    hideMetrics = false,
    hideCounts = false,
    hideDateFilter = false,
    hideTriggerRunButton = false,
    hideCancelAndReplayButtons = false,
    hideColumnToggle = false,
    hideFlatten = false,
  } = display ?? {};

  const initialState = useMemo(() => {
    const baseState: Partial<RunsTableState> = {
      columnVisibility: {
        ...initColumnVisibility,
        parentTaskExternalId: false, // Always hidden, used for filtering only
        flattenDAGs: false, // Always hidden, used for filtering only
      },
    };

    if (workflowId) {
      baseState.columnFilters = [{ id: workflowKey, value: workflowId }];
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

  const toolbarFilters = useToolbarFilters({
    filterVisibility,
    state,
    filterActions: filters,
  });

  const workflow =
    workflowId || getWorkflowIdsFromFilters(state.columnFilters)[0];
  const flattenDAGs = getFlattenDAGsFromFilters(state.columnFilters);

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
    pauseRefetch: isFrozen,
    onlyTasks: !!workerId || flattenDAGs,
    refetchInterval,
  });

  const actionModalParams = useMemo(
    () =>
      selectedRuns.length > 0
        ? { externalIds: selectedRuns.map((run) => run?.metadata.id) }
        : {
            filter: {
              ...filters.apiFilters,
              since: filters.apiFilters.since || '',
            },
          },
    [selectedRuns, filters.apiFilters],
  );

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
    pauseRefetch: isFrozen,
    additionalMetadata: filters.apiFilters.additionalMetadata,
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
      isActionModalOpen,
      isActionDropdownOpen,
      actionModalParams,
      selectedActionType,
      display: {
        hideMetrics,
        hideCounts,
        hideDateFilter,
        hideTriggerRunButton,
        hideCancelAndReplayButtons,
        hideColumnToggle,
        hidePagination: disableTaskRunPagination,
        hideFlatten,
        refetchInterval,
      },
      actions: {
        updatePagination,
        updateFilters,
        updateUIState,
        updateTableState,
        resetState,
        setIsFrozen,
        setIsActionModalOpen,
        setIsActionDropdownOpen,
        setSelectedActionType,
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
      isActionModalOpen,
      isActionDropdownOpen,
      hideMetrics,
      hideCounts,
      hideDateFilter,
      hideTriggerRunButton,
      hideFlatten,
      actionModalParams,
      selectedActionType,
      refetchInterval,
      updatePagination,
      updateFilters,
      updateUIState,
      updateTableState,
      resetState,
      setIsFrozen,
      setIsActionModalOpen,
      setIsActionDropdownOpen,
      setSelectedActionType,
      refetchRuns,
      refetchMetrics,
      getRowId,
      hideCancelAndReplayButtons,
      hideColumnToggle,
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
