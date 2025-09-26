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

import { AdditionalMetadataProp } from '../hooks/use-runs-table-filters';
import { useRunsContext } from '../hooks/runs-provider';

import { DocsButton } from '@/components/v1/docs/docs-button';
import { docsPages } from '@/lib/generated/docs';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { Loading } from '@/components/v1/ui/loading';

export interface RunsTableProps {
  headerClassName?: string;
}

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
    filters,
    toolbarFilters,
    tableRows,
    numPages,
    isRunsLoading,
    isRunsFetching,
    isStatusCountsFetching,
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
    display: {
      hideMetrics,
      hideCounts,
      hideColumnToggle,
      hidePagination,
      hiddenFilters,
    },
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
    if (filters.isCustomTimeRange) {
      return;
    }

    const interval = setInterval(() => {
      filters.updateCurrentTimeWindow();
    }, 60 * 1000);

    return () => clearInterval(interval);
  }, [filters, filters.isCustomTimeRange, filters.updateCurrentTimeWindow]);

  const hasLoaded = !isRunsLoading && !isStatusCountsLoading;
  const isFetching = !hasLoaded && (isRunsFetching || isStatusCountsFetching);

  return (
    <div className="flex flex-col h-full overflow-hidden gap-y-2">
      <Toaster />

      <TriggerWorkflowForm
        defaultWorkflow={undefined}
        show={showTriggerWorkflow}
        onClose={() => setShowTriggerWorkflow(false)}
      />

      {!hideMetrics && (
        <Dialog open={showQueueMetrics} onOpenChange={setShowQueueMetrics}>
          <DialogContent className="w-fit max-w-[80%] min-w-[500px]">
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
          columnVisibility={columnVisibility}
          setColumnVisibility={setColumnVisibility}
          data={tableRows}
          filters={toolbarFilters}
          leftActions={[
            ...(!hideCounts
              ? [
                  <div key="metrics" className="flex justify-start mr-auto">
                    {runStatusCounts.length > 0 ? (
                      <V1WorkflowRunsMetricsView />
                    ) : (
                      <Skeleton className="max-w-[800px] w-[40vw] h-8" />
                    )}
                  </div>,
                ]
              : []),
          ]}
          columnFilters={filters.columnFilters}
          setColumnFilters={(updaterOrValue) => {
            if (typeof updaterOrValue === 'function') {
              filters.setColumnFilters(updaterOrValue(filters.columnFilters));
            } else {
              filters.setColumnFilters(updaterOrValue);
            }
          }}
          pagination={hidePagination ? undefined : pagination}
          setPagination={setPagination}
          onSetPageSize={setPageSize}
          rowSelection={rowSelection}
          setRowSelection={setRowSelection}
          pageCount={hidePagination ? undefined : numPages}
          showColumnToggle={!hideColumnToggle}
          getSubRows={(row) => row.children || []}
          getRowId={getRowId}
          headerClassName={headerClassName}
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
