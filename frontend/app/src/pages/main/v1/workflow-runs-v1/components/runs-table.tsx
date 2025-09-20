import { useCallback, useEffect, useMemo, useState } from 'react';
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
import { TabOption } from '../$run/v2components/step-run-detail/step-run-detail';
import { useSidePanel } from '@/hooks/use-side-panel';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { TriggerWorkflowForm } from '../../workflows/$workflow/components/trigger-workflow-form';
import { Toaster } from '@/components/v1/ui/toaster';
import { useQuery } from '@tanstack/react-query';
import { queries } from '@/lib/api';

import { getCreatedAfterFromTimeRange } from '../hooks/use-runs-table-state';
import { AdditionalMetadataProp } from '../hooks/use-runs-table-filters';
import { useRunsContext } from '../hooks/runs-provider';

import { TableActions } from './task-runs-table/table-actions';
import { ConfirmActionModal } from '../../task-runs-v1/actions';
import { DocsButton } from '@/components/v1/docs/docs-button';
import { docsPages } from '@/lib/generated/docs';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';

export interface RunsTableProps {
  headerClassName?: string;
}

const GetWorkflowChart = () => {
  const { tenantId } = useCurrentTenantId();
  const { refetchInterval } = useRefetchInterval();

  const {
    state: { createdAfter, finishedBefore },
    filters: { setCustomTimeRange },
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
    refetchInterval,
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
  const sidePanel = useSidePanel();
  const { setIsFrozen } = useRefetchInterval();

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
    isRefetching,
    metrics,
    tenantMetrics,
    actionModalParams,
    selectedActionType,
    display: {
      hideMetrics,
      hideCounts,
      hideColumnToggle,
      hidePagination,
      hideFlatten,
    },
    actions: {
      updatePagination,
      updateFilters,
      updateUIState,
      updateTableState,
      refetchRuns,
      refetchMetrics,
      getRowId,
    },
  } = useRunsContext();

  const [selectedAdditionalMetaRunId, setSelectedAdditionalMetaRunId] =
    useState<string | null>(null);

  const handleTaskRunIdClick = useCallback(
    (taskRunId: string) => {
      sidePanel.open({
        type: 'task-run-details',
        content: {
          taskRunId,
          defaultOpenTab: TabOption.Output,
          showViewTaskRunButton: true,
        },
      });
    },
    [sidePanel],
  );

  const handleAdditionalMetadataOpenChange = useCallback(
    (rowId: string, open: boolean) => {
      if (open) {
        setSelectedAdditionalMetaRunId(rowId);
        setIsFrozen(true);
      } else {
        setSelectedAdditionalMetaRunId(null);
        setIsFrozen(false);
      }
    },
    [setIsFrozen],
  );

  const handleAdditionalMetadataClick = useCallback(
    (m: AdditionalMetadataProp) => {
      filters.setAdditionalMetadata(m);
    },
    [filters],
  );

  const tableColumns = useMemo(
    () =>
      columns(
        tenantId,
        selectedAdditionalMetaRunId,
        handleAdditionalMetadataClick,
        handleTaskRunIdClick,
        handleAdditionalMetadataOpenChange,
      ),
    [
      tenantId,
      selectedAdditionalMetaRunId,
      handleAdditionalMetadataClick,
      handleTaskRunIdClick,
      handleAdditionalMetadataOpenChange,
    ],
  );

  const handleRefresh = useCallback(() => {
    refetchRuns();
    refetchMetrics();
  }, [refetchRuns, refetchMetrics]);

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
    <div className="flex flex-col h-full overflow-hidden gap-y-2">
      <Toaster />
      {selectedActionType && (
        <ConfirmActionModal
          actionType={selectedActionType}
          params={actionModalParams}
        />
      )}

      <TriggerWorkflowForm
        defaultWorkflow={undefined}
        show={state.triggerWorkflow}
        onClose={() => updateUIState({ triggerWorkflow: false })}
      />

      {!hideMetrics && (
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

      {!hideMetrics && <GetWorkflowChart />}

      <div className="flex-1 min-h-0">
        <DataTable
          emptyState={
            <div className="w-full h-full flex flex-col gap-y-4 text-foreground py-8 justify-center items-center">
              <p className="text-lg font-semibold">No runs found</p>
              <div className="w-fit">
                <DocsButton
                  doc={docsPages.home['your-first-task']}
                  label={'Learn more about tasks'}
                  size="full"
                  variant="outline"
                />
              </div>
            </div>
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
          leftActions={[
            ...(!hideCounts
              ? [
                  <div key="metrics" className="flex justify-start mr-auto">
                    {metrics.length > 0 ? (
                      <V1WorkflowRunsMetricsView />
                    ) : (
                      <Skeleton className="max-w-[800px] w-[40vw] h-8" />
                    )}
                  </div>,
                ]
              : []),
          ]}
          rightActions={[
            <TableActions
              key="table-actions"
              onTriggerWorkflow={() => updateUIState({ triggerWorkflow: true })}
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
          pagination={hidePagination ? undefined : state.pagination}
          setPagination={
            hidePagination
              ? undefined
              : (updaterOrValue) => {
                  if (typeof updaterOrValue === 'function') {
                    updatePagination(updaterOrValue(state.pagination));
                  } else {
                    updatePagination(updaterOrValue);
                  }
                }
          }
          onSetPageSize={
            hidePagination
              ? undefined
              : (size) =>
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
          pageCount={hidePagination ? undefined : numPages}
          showColumnToggle={!hideColumnToggle}
          getSubRows={(row) => row.children || []}
          getRowId={getRowId}
          headerClassName={headerClassName}
          hideFlatten={hideFlatten}
          columnKeyToName={TaskRunColumn}
          onRefetch={handleRefresh}
          isRefetching={isRefetching}
        />
      </div>
    </div>
  );
}
