import { TabOption } from '../$run/v2components/step-run-detail/step-run-detail';
import { TriggerWorkflowForm } from '../../workflows/$workflow/components/trigger-workflow-form';
import { useRunsContext } from '../hooks/runs-provider';
import { AdditionalMetadataProp } from '../hooks/use-runs-table-filters';
import { RunsEmptyGraphic } from './runs-empty-graphic';
import { V1WorkflowRunsMetricsView } from './task-runs-metrics';
import { columns, TaskRunColumn } from './v1/task-runs-columns';
import {
  DataPoint,
  ZoomableChart,
} from '@/components/v1/molecules/charts/zoomable';
import { DataTable } from '@/components/v1/molecules/data-table/data-table.tsx';
import { EmptyState } from '@/components/v1/molecules/empty-state/empty-state';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Loading } from '@/components/v1/ui/loading';
import { Separator } from '@/components/v1/ui/separator';
import { Skeleton } from '@/components/v1/ui/skeleton';
import { Toaster } from '@/components/v1/ui/toaster';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { useSidePanel } from '@/hooks/use-side-panel';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { queries, V1TaskStatus } from '@/lib/api';
import { docsPages } from '@/lib/generated/docs';
import { useQuery } from '@tanstack/react-query';
import { useCallback, useEffect, useMemo, useState } from 'react';

const GetWorkflowChart = () => {
  const { tenantId } = useCurrentTenantId();
  const { refetchInterval } = useRefetchInterval();

  const {
    filters: { apiFilters, setCustomTimeRange },
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
      createdAfter: apiFilters.since,
      finishedBefore: apiFilters.until,
    }),
    placeholderData: (prev) => prev,
    refetchInterval,
  });

  if (workflowRunEventsMetricsQuery.isLoading) {
    return <Skeleton className="h-24 w-full" />;
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
      className="h-24 min-h-24"
    />
  );
};

export function RunsTable({ leftLabel }: { leftLabel?: string }) {
  const { tenantId } = useCurrentTenantId();
  const sidePanel = useSidePanel();
  const { setIsFrozen } = useRefetchInterval();

  const {
    filters,
    toolbarFilters,
    tableRows,
    numPages,
    isRunsLoading,
    isStatusCountsLoading,
    isQueueMetricsLoading,
    isRefetching,
    runStatusCounts,
    queueMetrics,
    actionModalParams,
    selectedActionType,
    pagination,
    columnVisibility,
    rowSelection,
    showTriggerWorkflow,
    showQueueMetrics,
    display: { hideMetrics, hideCounts, hideColumnToggle, hiddenFilters },
    actions: {
      refetchRuns,
      refetchMetrics,
      getRowId,
      setPageSize,
      setPagination,
      setColumnVisibility,
      setRowSelection,
      setShowTriggerWorkflow,
      setShowQueueMetrics,
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
          defaultOpenTab: TabOption.Activity,
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

  const handleIdempotencyKeyClick = useCallback(
    (idempotencyKey: string) => {
      filters.setIdempotencyKey(idempotencyKey);
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
        handleIdempotencyKeyClick,
      ),
    [
      tenantId,
      selectedAdditionalMetaRunId,
      handleAdditionalMetadataClick,
      handleTaskRunIdClick,
      handleAdditionalMetadataOpenChange,
      handleIdempotencyKeyClick,
    ],
  );

  const handleRefresh = useCallback(() => {
    refetchRuns();
    refetchMetrics();
  }, [refetchRuns, refetchMetrics]);

  useEffect(() => {
    if (filters.isCustomTimeRange) {
      return;
    }

    const interval = setInterval(() => {
      filters.updateCurrentTimeWindow();
    }, 60 * 1000);

    return () => clearInterval(interval);
  }, [filters, filters.isCustomTimeRange, filters.updateCurrentTimeWindow]);

  const isRunningFirstLoad = isRunsLoading || isStatusCountsLoading;

  const allStatusCount = Object.values(V1TaskStatus).length;
  const hasActiveFilters =
    (filters.apiFilters.statuses?.length ?? allStatusCount) < allStatusCount ||
    (filters.apiFilters.workflowIds?.length ?? 0) > 0 ||
    (filters.apiFilters.additionalMetadata?.length ?? 0) > 0 ||
    !!filters.apiFilters.runningFilter ||
    filters.isCustomTimeRange ||
    filters.timeWindow !== '1d';
  const isDefaultOneDayWindow =
    !filters.isCustomTimeRange && filters.timeWindow === '1d';

  const leftActions = [
    ...(!hideCounts
      ? [
          <div key="metrics" className="mr-auto flex justify-start">
            {runStatusCounts.length > 0 ? (
              <V1WorkflowRunsMetricsView />
            ) : (
              <Skeleton className="h-8 w-[40vw] max-w-[800px]" />
            )}
          </div>,
        ]
      : []),
    ...(leftLabel
      ? [
          <span
            key="left-label"
            className="mr-auto flex justify-start font-medium"
          >
            {leftLabel}
          </span>,
        ]
      : []),
  ];

  return (
    <div className="flex h-full flex-col gap-y-2 overflow-hidden">
      <Toaster />

      <TriggerWorkflowForm
        defaultWorkflow={undefined}
        show={showTriggerWorkflow}
        onClose={() => setShowTriggerWorkflow(false)}
      />

      {!hideMetrics && (
        <Dialog open={showQueueMetrics} onOpenChange={setShowQueueMetrics}>
          <DialogContent className="w-fit min-w-[500px] max-w-[80%]">
            <DialogHeader>
              <DialogTitle>Queue Metrics</DialogTitle>
            </DialogHeader>
            <Separator />
            {!queueMetrics || isQueueMetricsLoading ? (
              <Loading />
            ) : (
              <CodeHighlighter
                language="json"
                className="max-h-[400px] overflow-y-auto"
                code={JSON.stringify(queueMetrics || '{}', null, 2)}
              />
            )}
          </DialogContent>
        </Dialog>
      )}

      {!hideMetrics && <GetWorkflowChart />}

      <div className="min-h-0 flex-1">
        <DataTable
          emptyState={
            hasActiveFilters ? (
              <EmptyState
                graphic={<RunsEmptyGraphic />}
                title="No runs matching your filters"
                buttons={[
                  { label: 'Clear filters', onClick: filters.resetFilters },
                ]}
              />
            ) : (
              <EmptyState
                title="No runs found"
                description="Runs are individual executions of your tasks and workflows. Dispatch a task to see runs appear here."
                docPage={docsPages.v1.quickstart}
                docLabel="Learn about running tasks"
                buttons={
                  isDefaultOneDayWindow
                    ? [
                        {
                          label: 'Search past 7 days',
                          onClick: () => filters.setTimeWindow('7d'),
                        },
                      ]
                    : undefined
                }
              />
            )
          }
          isLoading={isRunningFirstLoad}
          columns={tableColumns}
          columnVisibility={columnVisibility}
          setColumnVisibility={setColumnVisibility}
          data={tableRows}
          filters={toolbarFilters}
          leftActions={leftActions}
          columnFilters={filters.columnFilters}
          setColumnFilters={(updaterOrValue) => {
            if (typeof updaterOrValue === 'function') {
              filters.setColumnFilters(updaterOrValue(filters.columnFilters));
            } else {
              filters.setColumnFilters(updaterOrValue);
            }
          }}
          pagination={pagination}
          setPagination={setPagination}
          onSetPageSize={setPageSize}
          rowSelection={rowSelection}
          setRowSelection={setRowSelection}
          pageCount={numPages}
          showColumnToggle={!hideColumnToggle}
          getSubRows={(row) => row.children || []}
          getRowId={getRowId}
          hiddenFilters={hiddenFilters}
          columnKeyToName={TaskRunColumn}
          refetchProps={{
            isRefetching,
            onRefetch: handleRefresh,
          }}
          tableActions={{
            showTableActions: true,
            onTriggerWorkflow: () => setShowTriggerWorkflow(true),
            selectedActionType,
            actionModalParams,
          }}
          onResetFilters={filters.resetFilters}
        />
      </div>
    </div>
  );
}
