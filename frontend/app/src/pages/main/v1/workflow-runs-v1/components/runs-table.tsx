import { useCallback, useEffect, useState, useMemo, useRef } from 'react';
import { VisibilityState } from '@tanstack/react-table';
import { DataTable } from '@/components/v1/molecules/data-table/data-table.tsx';
import { columns, TaskRunColumn } from './v1/task-runs-columns';
import { V1WorkflowRunsMetricsView } from './task-runs-metrics';
import { Skeleton } from '@/components/v1/ui/skeleton';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { Separator } from '@/components/v1/ui/separator';
import {
  DataPoint,
  ZoomableChart,
} from '@/components/v1/molecules/charts/zoomable';
import { Sheet, SheetContent } from '@/components/v1/ui/sheet';
import {
  TabOption,
  TaskRunDetail,
} from '../$run/v2components/step-run-detail/step-run-detail';
import { IntroDocsEmptyState } from '@/pages/onboarding/intro-docs-empty-state';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { TriggerWorkflowForm } from '../../workflows/$workflow/components/trigger-workflow-form';
import { useToast } from '@/components/v1/hooks/use-toast';
import { Toaster } from '@/components/v1/ui/toaster';
import { useQuery } from '@tanstack/react-query';
import { queries } from '@/lib/api';

import {
  useRunsTableState,
  TimeWindow,
  RunsTableState,
  getWorkflowIdFromFilters,
  getCreatedAfterFromTimeRange,
} from '../hooks/use-runs-table-state';
import {
  AdditionalMetadataProp,
  useRunsTableFilters,
} from '../hooks/use-runs-table-filters';
import { useRuns } from '../hooks/use-runs';
import { useMetrics } from '../hooks/metrics';
import { useToolbarFilters } from '../hooks/toolbar-filters';

import { TableHeader } from './task-runs-table/table-header';
import { TableActions } from './task-runs-table/table-actions';

export interface RunsTableProps {
  // Important: the key is used to identify a single instance of
  // the table's state, so that we can have multiple independent
  // tables stored in state (in the URL) at the same time. E.g.
  // this is helpful for showing child runs in the side sheet while
  // still showing the main runs view in the background.
  tableKey: string;

  createdAfter?: string;
  createdBefore?: string;
  workflowId?: string;
  workerId?: string;
  parentTaskExternalId?: string;
  triggeringEventExternalId?: string;
  initColumnVisibility?: VisibilityState;

  filterVisibility?: { [key: string]: boolean };
  showMetrics?: boolean;
  showCounts?: boolean;
  showDateFilter?: boolean;
  showTriggerRunButton?: boolean;
  disableTaskRunPagination?: boolean;

  headerClassName?: string;

  refetchInterval?: number;
}

const GetWorkflowChart = ({
  createdAfter,
  finishedBefore,
  refetchInterval,
  zoom,
  pauseRefetch = false,
}: {
  createdAfter?: string;
  finishedBefore?: string;
  refetchInterval?: number;
  zoom: (startTime: string, endTime: string) => void;
  pauseRefetch?: boolean;
}) => {
  const { tenantId } = useCurrentTenantId();
  const workflowRunEventsMetricsQuery = useQuery({
    ...queries.v1TaskRuns.pointMetrics(tenantId, {
      createdAfter,
      finishedBefore,
    }),
    placeholderData: (prev) => prev,
    refetchInterval: pauseRefetch ? false : refetchInterval,
  });

  if (workflowRunEventsMetricsQuery.isLoading) {
    return <Skeleton className="w-full h-36" />;
  }

  return (
    <ZoomableChart
      kind="bar"
      data={
        workflowRunEventsMetricsQuery.data?.results?.map(
          (result): DataPoint<'SUCCEEDED' | 'FAILED'> => ({
            date: result.time,
            SUCCEEDED: result.SUCCEEDED,
            FAILED: result.FAILED,
          }),
        ) || []
      }
      colors={{
        SUCCEEDED: 'rgb(34 197 94 / 0.5)',
        FAILED: 'hsl(var(--destructive))',
      }}
      zoom={zoom}
      showYAxis={false}
    />
  );
};

export function RunsTable({
  tableKey,
  workflowId,
  workerId,
  parentTaskExternalId,
  triggeringEventExternalId,
  createdAfter: createdAfterProp,
  initColumnVisibility = {},
  filterVisibility = {},
  refetchInterval = 5000,
  showMetrics = false,
  showCounts = true,
  showDateFilter = true,
  disableTaskRunPagination = false,
  showTriggerRunButton = true,
  headerClassName,
}: RunsTableProps) {
  const { tenantId } = useCurrentTenantId();
  const { toast } = useToast();

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
  const filtersRef = useRef(filters);
  filtersRef.current = filters;
  const [taskIdsPendingAction, setTaskIdsPendingAction] = useState<string[]>(
    [],
  );
  const [rotate, setRotate] = useState(false);

  const toolbarFilters = useToolbarFilters({ filterVisibility });

  const workflow = workflowId || getWorkflowIdFromFilters(state.columnFilters);
  const derivedParentTaskExternalId =
    parentTaskExternalId || state.parentTaskExternalId;
  const [isFrozen, setIsFrozen] = useState(false);

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

  const handleTaskRunIdClick = useCallback(
    (taskRunId: string) => {
      updateUIState({
        stepDetailSheet: {
          taskRunId,
          isOpen: true,
        },
      });
    },
    [updateUIState],
  );

  const handleSetSelectedAdditionalMetaRunId = useCallback(
    (runId: string | null) => {
      updateUIState({ selectedAdditionalMetaRunId: runId || undefined });
    },
    [updateUIState],
  );

  const handleAdditionalMetadataClick = useCallback(
    (m: AdditionalMetadataProp) => {
      setIsFrozen(true);
      filtersRef.current.setAdditionalMetadata(m);
    },
    [setIsFrozen],
  );

  const tableColumns = useMemo(
    () =>
      columns(
        tenantId,
        state.selectedAdditionalMetaRunId || null,
        handleSetSelectedAdditionalMetaRunId,
        handleAdditionalMetadataClick,
        handleTaskRunIdClick,
      ),
    [
      tenantId,
      state.selectedAdditionalMetaRunId,
      handleSetSelectedAdditionalMetaRunId,
      handleAdditionalMetadataClick,
      handleTaskRunIdClick,
    ],
  );

  const handleRefresh = useCallback(() => {
    refetchRuns();
    refetchMetrics();
    setRotate(!rotate);
  }, [refetchRuns, refetchMetrics, rotate]);

  const handleActionProcessed = useCallback(
    (action: 'cancel' | 'replay', ids: string[]) => {
      const prefix = action === 'cancel' ? 'Canceling' : 'Replaying';
      const count = ids.length;

      setTaskIdsPendingAction(ids);
      const t = toast({
        title: `${prefix} ${count} task run${count > 1 ? 's' : ''}`,
        description: `This may take a few seconds. You don't need to hit ${action} again.`,
      });

      setTimeout(() => {
        setTaskIdsPendingAction([]);
        t.dismiss();
      }, 5000);
    },
    [toast],
  );

  const handleTimeWindowChange = useCallback(
    (value: TimeWindow | 'custom') => {
      if (value !== 'custom') {
        filters.setTimeWindow(value);
      } else {
        updateFilters({ isCustomTimeRange: true });
      }
    },
    [filters, updateFilters],
  );

  useEffect(() => {
    if (state.isCustomTimeRange) {
      return;
    }

    const interval = setInterval(() => {
      updateFilters({
        createdAfter: getCreatedAfterFromTimeRange(state.timeWindow),
      });
    }, 60 * 1000);

    return () => clearInterval(interval);
  }, [state.isCustomTimeRange, state.timeWindow, updateFilters]);

  const hasLoaded = !isRunsLoading && !isMetricsLoading;
  const isFetching = !hasLoaded && (isRunsFetching || isMetricsFetching);

  return (
    <div className="flex flex-col h-full overflow-hidden">
      <Toaster />

      <TriggerWorkflowForm
        defaultWorkflow={undefined}
        show={state.triggerWorkflow}
        onClose={() => updateUIState({ triggerWorkflow: false })}
      />

      {showMetrics && !derivedParentTaskExternalId && (
        <Dialog
          open={state.viewQueueMetrics}
          onOpenChange={(open) => {
            if (!open) {
              updateUIState({ viewQueueMetrics: false });
            }
          }}
        >
          <DialogContent className="w-fit max-w-[80%] min-w-[500px]">
            <DialogHeader>
              <DialogTitle>Queue Metrics</DialogTitle>
            </DialogHeader>
            <Separator />
            {tenantMetrics && (
              <CodeHighlighter
                language="json"
                className="max-h-[400px] overflow-y-auto"
                code={JSON.stringify(tenantMetrics || '{}', null, 2)}
              />
            )}
            {isMetricsLoading && 'Loading...'}
          </DialogContent>
        </Dialog>
      )}

      <TableHeader
        timeWindow={state.timeWindow}
        isCustomTimeRange={state.isCustomTimeRange}
        createdAfter={state.createdAfter}
        finishedBefore={state.finishedBefore}
        onTimeWindowChange={handleTimeWindowChange}
        onCreatedAfterChange={(date) => updateFilters({ createdAfter: date })}
        onFinishedBeforeChange={(date) =>
          updateFilters({ finishedBefore: date })
        }
        onClearTimeRange={() => filters.setCustomTimeRange(null)}
        showDateFilter={showDateFilter && !createdAfterProp}
        hasParentFilter={!!derivedParentTaskExternalId}
      />

      {showMetrics && !derivedParentTaskExternalId && (
        <GetWorkflowChart
          createdAfter={state.createdAfter}
          zoom={(createdAfter, createdBefore) => {
            filters.setCustomTimeRange({
              start: createdAfter,
              end: createdBefore,
            });
          }}
          finishedBefore={state.finishedBefore}
          refetchInterval={refetchInterval}
          pauseRefetch={state.hasOpenUI}
        />
      )}

      {showCounts && (
        <div className="flex flex-row justify-between items-center my-4">
          {metrics.length > 0 ? (
            <V1WorkflowRunsMetricsView
              metrics={metrics}
              onViewQueueMetricsClick={() => {
                updateUIState({ viewQueueMetrics: true });
              }}
              showQueueMetrics={showMetrics}
              onClick={filters.setStatus}
            />
          ) : (
            <Skeleton className="max-w-[800px] w-[40vw] h-8" />
          )}
        </div>
      )}

      {state.stepDetailSheet.taskRunId && (
        <Sheet
          open={state.stepDetailSheet.isOpen}
          onOpenChange={(isOpen) =>
            updateUIState({
              stepDetailSheet: { ...state.stepDetailSheet, isOpen },
            })
          }
        >
          <SheetContent className="w-fit min-w-[56rem] max-w-4xl sm:max-w-2xl z-[60] h-full overflow-auto">
            <TaskRunDetail
              taskRunId={state.stepDetailSheet.taskRunId}
              defaultOpenTab={TabOption.Output}
              showViewTaskRunButton
            />
          </SheetContent>
        </Sheet>
      )}

      <div className="flex-1 min-h-0">
        <DataTable
          emptyState={
            <IntroDocsEmptyState
              link="/home/your-first-task"
              title="No Runs Found"
              linkPreambleText="To learn more about how workflows function in Hatchet,"
              linkText="check out our documentation."
            />
          }
          isLoading={isFetching}
          columns={tableColumns}
          columnVisibility={state.columnVisibility}
          setColumnVisibility={(visibility) => {
            if (typeof visibility === 'function') {
              updateTableState({
                columnVisibility: visibility(state.columnVisibility),
              });
            } else {
              updateTableState({ columnVisibility: visibility });
            }
          }}
          data={tableRows}
          filters={toolbarFilters}
          actions={[
            <TableActions
              key="table-actions"
              hasRowsSelected={state.hasRowsSelected}
              hasFiltersApplied={state.hasFiltersApplied}
              selectedRuns={selectedRuns}
              apiFilters={filters.apiFilters}
              taskIdsPendingAction={taskIdsPendingAction}
              onRefresh={handleRefresh}
              onActionProcessed={handleActionProcessed}
              onTriggerWorkflow={() => updateUIState({ triggerWorkflow: true })}
              showTriggerRunButton={showTriggerRunButton}
              rotate={rotate}
              toast={toast}
            />,
          ]}
          columnFilters={state.columnFilters}
          setColumnFilters={(updaterOrValue) => {
            if (typeof updaterOrValue === 'function') {
              filters.setColumnFilters(updaterOrValue(state.columnFilters));
            } else {
              filters.setColumnFilters(updaterOrValue);
            }
          }}
          pagination={state.pagination}
          setPagination={(updaterOrValue) => {
            if (typeof updaterOrValue === 'function') {
              updatePagination(updaterOrValue(state.pagination));
            } else {
              updatePagination(updaterOrValue);
            }
          }}
          onSetPageSize={(size) =>
            updatePagination({ ...state.pagination, pageSize: size })
          }
          rowSelection={state.rowSelection}
          setRowSelection={(updaterOrValue) => {
            if (typeof updaterOrValue === 'function') {
              updateTableState({
                rowSelection: updaterOrValue(state.rowSelection),
              });
            } else {
              updateTableState({ rowSelection: updaterOrValue });
            }
          }}
          pageCount={numPages}
          showColumnToggle={true}
          getSubRows={(row) => row.children || []}
          getRowId={getRowId}
          onToolbarReset={resetState}
          headerClassName={headerClassName}
        />
      </div>
    </div>
  );
}
