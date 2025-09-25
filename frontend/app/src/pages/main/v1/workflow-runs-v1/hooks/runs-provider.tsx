import { createContext, useContext, useMemo, useState } from 'react';
import { RowSelectionState, VisibilityState } from '@tanstack/react-table';
import { useRunsTableFilters } from './use-runs-table-filters';
import { useToolbarFilters } from './use-toolbar-filters';
import { useRuns } from './use-runs';
import { useMetrics } from './use-metrics';
import { V1TaskRunMetrics, V1TaskSummary } from '@/lib/api';
import { PaginationState, Updater } from '@tanstack/react-table';
import {
  ActionType,
  BaseTaskRunActionParams,
} from '../../task-runs-v1/actions';
import { TaskRunColumnKeys } from '../components/v1/task-runs-columns';

type DisplayProps = {
  hideMetrics?: boolean;
  hideCounts?: boolean;
  hideDateFilter?: boolean;
  hideTriggerRunButton?: boolean;
  hideCancelAndReplayButtons?: boolean;
  hideColumnToggle?: boolean;
  hiddenFilters?: TaskRunColumnKeys[];
  hidePagination?: boolean;
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
  display?: DisplayProps;
  runFilters?: RunFilteringProps;
};

type RunsContextType = {
  actions: {
    setIsActionModalOpen: (isOpen: boolean) => void;
    setIsActionDropdownOpen: (isOpen: boolean) => void;
    setSelectedActionType: (actionType: ActionType | null) => void;
    refetchRuns: () => void;
    refetchMetrics: () => void;
    getRowId: (row: V1TaskSummary) => string;
    setPagination: (updater: Updater<PaginationState>) => void;
    setPageSize: (size: number) => void;
    setColumnVisibility: (updater: Updater<VisibilityState>) => void;
    setRowSelection: (updater: Updater<RowSelectionState>) => void;
    setShowTriggerWorkflow: (trigger: boolean) => void;
    setShowQueueMetrics: (show: boolean) => void;
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
  isRefetching: boolean;
  runStatusCounts: V1TaskRunMetrics;
  queueMetrics: object;
  isActionModalOpen: boolean;
  isActionDropdownOpen: boolean;
  selectedActionType: ActionType | null;
  actionModalParams: BaseTaskRunActionParams;
  display: DisplayProps;
  pagination: PaginationState;
  columnVisibility: VisibilityState;
  rowSelection: RowSelectionState;
  showTriggerWorkflow: boolean;
  showQueueMetrics: boolean;
};

const RunsContext = createContext<RunsContextType | null>(null);

export const RunsProvider = ({
  tableKey,
  children,
  disableTaskRunPagination = false,
  initColumnVisibility = {},
  filterVisibility = {},
  display,
  runFilters,
}: RunsProviderProps) => {
  const [isActionModalOpen, setIsActionModalOpen] = useState(false);
  const [isActionDropdownOpen, setIsActionDropdownOpen] = useState(false);
  const [selectedActionType, setSelectedActionType] =
    useState<ActionType | null>(null);

  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});
  const [showQueueMetrics, setShowQueueMetrics] = useState(false);
  const [showTriggerWorkflow, setShowTriggerWorkflow] = useState(false);

  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({
    ...initColumnVisibility,
    parentTaskExternalId: false, // Always hidden, used for filtering only
    flattenDAGs: false, // Always hidden, used for filtering only
  });

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
    hiddenFilters = [],
  } = display ?? {};

  const filters = useRunsTableFilters(tableKey, {
    workflowIds: workflowId ? [workflowId] : undefined,
  });

  const toolbarFilters = useToolbarFilters({
    filterVisibility,
    filterActions: filters,
  });

  const workflow = workflowId || (filters.apiFilters.workflowIds ?? [])[0];
  const flattenDAGs = filters.apiFilters.flattenDAGs;

  const {
    tableRows,
    selectedRuns,
    numPages,
    isLoading: isRunsLoading,
    isFetching: isRunsFetching,
    refetch: refetchRuns,
    getRowId,
    isRefetching: isRunsRefetching,
    pagination,
    setPagination,
    setPageSize,
  } = useRuns({
    key: tableKey,
    rowSelection,
    createdAfter: filters.apiFilters.since,
    finishedBefore: filters.apiFilters.until,
    statuses: filters.apiFilters.statuses,
    additionalMetadata: filters.apiFilters.additionalMetadata,
    workerId,
    workflowIds:
      filters.apiFilters.workflowIds || (workflow ? [workflow] : undefined),
    parentTaskExternalId,
    triggeringEventExternalId,
    disablePagination: disableTaskRunPagination,
    onlyTasks: !!workerId || flattenDAGs,
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
    runStatusCounts,
    queueMetrics,
    isLoading: isMetricsLoading,
    isFetching: isMetricsFetching,
    refetch: refetchMetrics,
    isRefetching: isMetricsRefetching,
  } = useMetrics({
    workflow,
    parentTaskExternalId,
    createdAfter: filters.apiFilters.since,
    additionalMetadata: filters.apiFilters.additionalMetadata,
  });

  const isRefetching = isRunsRefetching || isMetricsRefetching;

  const value = useMemo<RunsContextType>(
    () => ({
      filters,
      toolbarFilters,
      tableRows,
      selectedRuns,
      numPages,
      isRunsLoading,
      isRunsFetching,
      isMetricsLoading,
      isMetricsFetching,
      isRefetching,
      runStatusCounts,
      queueMetrics,
      isActionModalOpen,
      isActionDropdownOpen,
      actionModalParams,
      selectedActionType,
      pagination,
      columnVisibility,
      rowSelection,
      showTriggerWorkflow,
      showQueueMetrics,
      display: {
        hideMetrics,
        hideCounts,
        hideDateFilter,
        hideTriggerRunButton,
        hideCancelAndReplayButtons,
        hideColumnToggle,
        hidePagination: disableTaskRunPagination,
        hiddenFilters,
      },
      actions: {
        setIsActionModalOpen,
        setIsActionDropdownOpen,
        setSelectedActionType,
        refetchRuns,
        refetchMetrics,
        getRowId,
        setPagination,
        setPageSize,
        setColumnVisibility,
        setRowSelection,
        setShowQueueMetrics,
        setShowTriggerWorkflow,
      },
    }),
    [
      filters,
      toolbarFilters,
      tableRows,
      selectedRuns,
      numPages,
      isRunsLoading,
      isRunsFetching,
      isMetricsLoading,
      isMetricsFetching,
      runStatusCounts,
      queueMetrics,
      isActionModalOpen,
      isActionDropdownOpen,
      hideMetrics,
      hideCounts,
      hideDateFilter,
      hideTriggerRunButton,
      hiddenFilters,
      actionModalParams,
      selectedActionType,
      setIsActionModalOpen,
      setIsActionDropdownOpen,
      setSelectedActionType,
      refetchRuns,
      refetchMetrics,
      getRowId,
      setPageSize,
      pagination,
      setPagination,
      hideCancelAndReplayButtons,
      hideColumnToggle,
      disableTaskRunPagination,
      isRefetching,
      setShowQueueMetrics,
      showQueueMetrics,
      setShowTriggerWorkflow,
      setRowSelection,
      columnVisibility,
      setColumnVisibility,
      rowSelection,
      showTriggerWorkflow,
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
