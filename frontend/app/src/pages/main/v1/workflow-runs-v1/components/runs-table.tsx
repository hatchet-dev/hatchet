import { useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { DataTable } from '@/components/v1/molecules/data-table/data-table.tsx';
import { columns } from './v1/task-runs-columns';
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

import { getCreatedAfterFromTimeRange } from '../hooks/use-runs-table-state';
import { AdditionalMetadataProp } from '../hooks/use-runs-table-filters';
import { useRunsContext } from '../hooks/runs-provider';

import { TableActions } from './task-runs-table/table-actions';
import { TimeFilter } from './task-runs-table/time-filter';

export interface RunsTableProps {
  headerClassName?: string;
}

const GetWorkflowChart = () => {
  const { tenantId } = useCurrentTenantId();

  const {
    state: { createdAfter, finishedBefore, hasOpenUI },
    filters: { setCustomTimeRange },
    display: { refetchInterval },
  } = useRunsContext();

  const zoom = useCallback(
    (createdAfter: string, createdBefore: string) => {
      setCustomTimeRange({
        start: createdAfter,
        end: createdBefore,
      });
    },
    [setCustomTimeRange],
  );

  const workflowRunEventsMetricsQuery = useQuery({
    ...queries.v1TaskRuns.pointMetrics(tenantId, {
      createdAfter,
      finishedBefore,
    }),
    placeholderData: (prev) => prev,
    refetchInterval: hasOpenUI ? false : refetchInterval,
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

export function RunsTable({ headerClassName }: RunsTableProps) {
  const { tenantId } = useCurrentTenantId();
  const { toast } = useToast();
  const [, setSearchParams] = useSearchParams();

  const {
    state,
    filters,
    toolbarFilters,
    tableRows,
    numPages,
    isRunsLoading,
    isRunsFetching,
    isMetricsLoading,
    isMetricsFetching,
    metrics,
    tenantMetrics,
    display: { showMetrics, showCounts, showColumnToggle, showPagination },
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
  } = useRunsContext();

  const [taskIdsPendingAction, setTaskIdsPendingAction] = useState<string[]>(
    [],
  );
  const [rotate, setRotate] = useState(false);
  const [selectedAdditionalMetaRunId, setSelectedAdditionalMetaRunId] = useState<string | null>(null);

  const handleTaskRunIdClick = useCallback(
    (taskRunId: string) => {
      updateUIState({
        taskRunDetailSheet: {
          taskRunId,
          isOpen: true,
        },
      });
    },
    [updateUIState],
  );

  const handleSetSelectedAdditionalMetaRunId = useCallback(
    (runId: string | null) => {
      setSelectedAdditionalMetaRunId(runId);
    },
    [],
  );

  const handleAdditionalMetadataOpenChange = useCallback(
    (rowId: string, open: boolean) => {
      if (open) {
        setSelectedAdditionalMetaRunId(rowId);
      } else {
        setSelectedAdditionalMetaRunId(null);
      }
    },
    [],
  );

  const handleAdditionalMetadataClick = useCallback(
    (m: AdditionalMetadataProp) => {
      setIsFrozen(true);
      filters.setAdditionalMetadata(m);
    },
    [setIsFrozen, filters],
  );

  const tableColumns = useMemo(
    () =>
      columns(
        tenantId,
        selectedAdditionalMetaRunId,
        handleSetSelectedAdditionalMetaRunId,
        handleAdditionalMetadataClick,
        handleTaskRunIdClick,
        handleAdditionalMetadataOpenChange,
      ),
    [
      tenantId,
      selectedAdditionalMetaRunId,
      handleSetSelectedAdditionalMetaRunId,
      handleAdditionalMetadataClick,
      handleTaskRunIdClick,
      handleAdditionalMetadataOpenChange,
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

      {showMetrics && (
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

      <TimeFilter />

      {showMetrics && <GetWorkflowChart />}

      {showCounts && (
        <div className="flex flex-row justify-between items-center my-4">
          {metrics.length > 0 ? (
            <V1WorkflowRunsMetricsView />
          ) : (
            <Skeleton className="max-w-[800px] w-[40vw] h-8" />
          )}
        </div>
      )}

      {state.taskRunDetailSheet.isOpen && (
        <Sheet
          open={state.taskRunDetailSheet.isOpen}
          onOpenChange={(isOpen) => {
            if (!isOpen && state.taskRunDetailSheet.taskRunId) {
              // Clear the child runs table state when sheet closes
              const childTableKey = `table_child-runs-${state.taskRunDetailSheet.taskRunId}`;
              setSearchParams(
                (prev) => {
                  const newParams = new URLSearchParams(prev);
                  newParams.delete(childTableKey);
                  return newParams;
                },
                { replace: true },
              );
            }
            updateUIState({
              taskRunDetailSheet: isOpen
                ? {
                    isOpen: true,
                    taskRunId: state.taskRunDetailSheet.taskRunId!,
                  }
                : { isOpen: false },
            });
          }}
        >
          <SheetContent className="w-fit min-w-[56rem] max-w-4xl sm:max-w-2xl z-[60] h-full overflow-auto">
            <TaskRunDetail
              taskRunId={state.taskRunDetailSheet.taskRunId!}
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
              taskIdsPendingAction={taskIdsPendingAction}
              onRefresh={handleRefresh}
              onActionProcessed={handleActionProcessed}
              onTriggerWorkflow={() => updateUIState({ triggerWorkflow: true })}
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
          pagination={showPagination ? state.pagination : undefined}
          setPagination={
            showPagination
              ? (updaterOrValue) => {
                  if (typeof updaterOrValue === 'function') {
                    updatePagination(updaterOrValue(state.pagination));
                  } else {
                    updatePagination(updaterOrValue);
                  }
                }
              : undefined
          }
          onSetPageSize={
            showPagination
              ? (size) =>
                  updatePagination({ ...state.pagination, pageSize: size })
              : undefined
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
          pageCount={showPagination ? numPages : undefined}
          showColumnToggle={showColumnToggle}
          getSubRows={(row) => row.children || []}
          getRowId={getRowId}
          onToolbarReset={resetState}
          headerClassName={headerClassName}
        />
      </div>
    </div>
  );
}
