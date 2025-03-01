import { DataTable } from '@/components/v1/molecules/data-table/data-table.tsx';
import { columns } from './v1/task-runs-columns';
import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  RowSelectionState,
  SortingState,
  VisibilityState,
} from '@tanstack/react-table';
import { useQuery } from '@tanstack/react-query';
import invariant from 'tiny-invariant';
import { queries, V1TaskStatus, V1TaskSummary } from '@/lib/api';
import { TenantContextType } from '@/lib/outlet';
import { useOutletContext, useSearchParams } from 'react-router-dom';
import {
  FilterOption,
  ToolbarFilters,
  ToolbarType,
} from '@/components/v1/molecules/data-table/data-table-toolbar';
import { Button } from '@/components/v1/ui/button';
import { ArrowPathIcon, XCircleIcon } from '@heroicons/react/24/outline';
import { V1WorkflowRunsMetricsView } from './task-runs-metrics';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
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
import { DateTimePicker } from '@/components/v1/molecules/time-picker/date-time-picker';
import { AdditionalMetadataClick } from '../../events/components/additional-metadata';
import { Sheet, SheetContent } from '@/components/v1/ui/sheet';
import {
  TabOption,
  TaskRunDetail,
} from '../$run/v2components/step-run-detail/step-run-detail';
import { TaskRunActionButton } from '../../task-runs-v1/actions';
import { useColumnFilters } from '../hooks/use-column-filters';
import { usePagination } from '../hooks/use-pagination';
import { useTaskRuns } from '../hooks/use-task-runs';

export interface TaskRunsTableProps {
  createdAfter?: string;
  createdBefore?: string;
  workflowId?: string;
  workerId?: string;
  parentWorkflowRunId?: string;
  parentStepRunId?: string;
  initColumnVisibility?: VisibilityState;
  filterVisibility?: { [key: string]: boolean };
  refetchInterval?: number;
  showMetrics?: boolean;
  showCounts?: boolean;
}

export const getCreatedAfterFromTimeRange = (timeRange?: string) => {
  switch (timeRange) {
    case '1h':
      return new Date(Date.now() - 60 * 60 * 1000).toISOString();
    case '6h':
      return new Date(Date.now() - 6 * 60 * 60 * 1000).toISOString();
    case '1d':
      return new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString();
    case '7d':
      return new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString();
  }
};

type StepDetailSheetState = {
  isOpen: boolean;
  taskRunId: string | undefined;
};

const useWorkflow = () => {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const {
    data: workflowKeys,
    isLoading: workflowKeysIsLoading,
    error: workflowKeysError,
  } = useQuery({
    ...queries.workflows.list(tenant.metadata.id, { limit: 200 }),
  });

  return {
    workflowKeys,
    workflowKeysIsLoading,
    workflowKeysError,
  };
};

export function TaskRunsTable({
  workflowId,
  workerId,
  createdAfter: createdAfterProp,
  initColumnVisibility = {},
  filterVisibility = {},
  refetchInterval = 5000,
  showMetrics = false,
  showCounts = true,
}: TaskRunsTableProps) {
  const [searchParams] = useSearchParams();
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const [viewQueueMetrics, setViewQueueMetrics] = useState(false);
  const [initialRenderTime] = useState(
    new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
  );

  const cf = useColumnFilters();

  const [stepDetailSheetState, setStepDetailSheetState] =
    useState<StepDetailSheetState>({
      isOpen: false,
      taskRunId: undefined,
    });

  // create a timer which updates the createdAfter date every minute
  useEffect(() => {
    const interval = setInterval(() => {
      if (cf.filters.isCustomTimeRange) {
        return;
      }

      cf.setCreatedAfter(
        getCreatedAfterFromTimeRange(cf.filters.defaultTimeRange) ||
          new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
      );
    }, 60 * 1000);

    return () => clearInterval(interval);
  }, [
    cf.filters.isCustomTimeRange,
    cf.filters.defaultTimeRange,
    cf.setCreatedAfter,
    cf,
  ]);

  const [sorting, setSorting] = useState<SortingState>(() => {
    const sortParam = searchParams.get('sort');
    if (sortParam) {
      return sortParam.split(',').map((param) => {
        const [id, desc] = param.split(':');
        return { id, desc: desc === 'desc' };
      });
    }
    return [];
  });

  const [columnVisibility, setColumnVisibility] =
    useState<VisibilityState>(initColumnVisibility);

  const { pagination, setPagination, setPageSize } = usePagination();

  const workflow = useMemo<string | undefined>(() => {
    if (workflowId) {
      return workflowId;
    }

    return cf.filters.workflowId;
  }, [cf.filters.workflowId, workflowId]);

  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});

  const {
    tableRows,
    selectedRuns,
    numPages,
    isLoading: taskRunsIsLoading,
    refetch: refetchTaskRuns,
  } = useTaskRuns({
    rowSelection,
    workerId,
    workflow,
  });

  const metricsQuery = useQuery({
    ...queries.v1TaskRuns.metrics(tenant.metadata.id, {
      since: cf.filters.createdAfter || initialRenderTime,
      workflow_ids: workflow ? [workflow] : [],
    }),
    placeholderData: (prev) => prev,
    refetchInterval,
  });

  const metrics = metricsQuery.data || [];

  const tenantMetricsQuery = useQuery({
    ...queries.metrics.getStepRunQueueMetrics(tenant.metadata.id),
    refetchInterval,
  });

  const tenantMetrics = tenantMetricsQuery.data?.queues || {};

  const { workflowKeys, workflowKeysError } = useWorkflow();

  const onTaskRunIdClick = useCallback((taskRunId: string) => {
    setStepDetailSheetState({
      taskRunId,
      isOpen: true,
    });
  }, []);

  const workflowKeyFilters = useMemo((): FilterOption[] => {
    return (
      workflowKeys?.rows?.map((key) => ({
        value: key.metadata.id,
        label: key.name,
      })) || []
    );
  }, [workflowKeys]);

  const workflowRunStatusFilters = useMemo((): FilterOption[] => {
    return [
      {
        value: V1TaskStatus.COMPLETED,
        label: 'Succeeded',
      },
      {
        value: V1TaskStatus.FAILED,
        label: 'Failed',
      },
      {
        value: V1TaskStatus.RUNNING,
        label: 'Running',
      },
      {
        value: V1TaskStatus.QUEUED,
        label: 'Queued',
      },
      // {
      //   value: V1TaskStatus.CANCELLED,
      //   label: 'Cancelled',
      // },
    ];
  }, []);

  const filters: ToolbarFilters = [
    {
      columnId: 'Workflow',
      title: 'Workflow',
      options: workflowKeyFilters,
      type: ToolbarType.Radio,
    },
    {
      columnId: 'status',
      title: 'Status',
      options: workflowRunStatusFilters,
      type: ToolbarType.Radio,
    },
    {
      columnId: 'additionalMetadata',
      title: 'Metadata',
      type: ToolbarType.KeyValue,
    },
  ].filter((filter) => filterVisibility[filter.columnId] != false);

  const [rotate, setRotate] = useState(false);

  const refetch = () => {
    refetchTaskRuns();
    metricsQuery.refetch();
    tenantMetricsQuery.refetch();
  };

  const v1TaskFilters = {
    since: cf.filters.createdAfter || initialRenderTime,
    until: cf.filters.finishedBefore,
    statuses: cf.filters.status ? [cf.filters.status] : undefined,
    workflowIds: workflow ? [workflow] : undefined,
    additionalMetadata: cf.filters.additionalMetadata,
  };

  const hasRowsSelected = Object.values(rowSelection).some(
    (selected) => !!selected,
  );
  const hasTaskFiltersSelected = Object.values(v1TaskFilters).some(
    (filter) => !!filter,
  );

  const actions = [
    <TaskRunActionButton
      key="cancel"
      actionType="cancel"
      disabled={!(hasRowsSelected || hasTaskFiltersSelected)}
      params={
        selectedRuns.length > 0
          ? { externalIds: selectedRuns.map((run) => run?.metadata.id) }
          : { filter: v1TaskFilters }
      }
    />,
    <TaskRunActionButton
      key="replay"
      actionType="replay"
      disabled={!(hasRowsSelected || hasTaskFiltersSelected)}
      params={
        selectedRuns.length > 0
          ? { externalIds: selectedRuns.map((run) => run?.metadata.id) }
          : { filter: v1TaskFilters }
      }
    />,
    <Button
      key="refresh"
      className="h-8 px-2 lg:px-3"
      size="sm"
      onClick={() => {
        refetch();
        setRotate(!rotate);
      }}
      variant={'outline'}
      aria-label="Refresh events list"
    >
      <ArrowPathIcon
        className={`h-4 w-4 transition-transform ${rotate ? 'rotate-180' : ''}`}
      />
    </Button>,
  ];

  const isLoading =
    taskRunsIsLoading ||
    metricsQuery.isLoading ||
    tenantMetricsQuery.isLoading ||
    metricsQuery.isFetching ||
    tenantMetricsQuery.isFetching;

  const onAdditionalMetadataClick = ({
    key,
    value,
  }: AdditionalMetadataClick) => {
    cf.setAdditionalMetadata(key, value);
  };

  const getRowId = useCallback((row: V1TaskSummary) => {
    return row.metadata.id;
  }, []);

  return (
    <>
      {showMetrics && (
        <Dialog
          open={viewQueueMetrics}
          onOpenChange={(open) => {
            if (!open) {
              setViewQueueMetrics(false);
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
                code={JSON.stringify(tenantMetrics || '{}', null, 2)}
              />
            )}
            {metricsQuery.isLoading && <Skeleton className="w-full h-36" />}
          </DialogContent>
        </Dialog>
      )}
      {!createdAfterProp && (
        <div className="flex flex-row justify-end items-center my-4 gap-2">
          {cf.filters.isCustomTimeRange && [
            <Button
              key="clear"
              onClick={() => {
                cf.setCustomTimeRange(undefined);
              }}
              variant="outline"
              size="sm"
              className="text-xs h-9 py-2"
            >
              <XCircleIcon className="h-[18px] w-[18px] mr-2" />
              Clear
            </Button>,
            <DateTimePicker
              key="after"
              label="After"
              date={
                cf.filters.createdAfter
                  ? new Date(cf.filters.createdAfter)
                  : undefined
              }
              setDate={(date) => {
                cf.setCreatedAfter(date?.toISOString());
              }}
            />,
            <DateTimePicker
              key="before"
              label="Before"
              date={
                cf.filters.finishedBefore
                  ? new Date(cf.filters.finishedBefore)
                  : undefined
              }
              setDate={(date) => {
                cf.setFinishedBefore(date?.toISOString());
              }}
            />,
          ]}
          <Select
            value={
              cf.filters.isCustomTimeRange
                ? 'custom'
                : cf.filters.defaultTimeRange
            }
            onValueChange={(value) => {
              if (value !== 'custom') {
                cf.setFilterValues([
                  { key: 'defaultTimeRange', value: value },
                  { key: 'isCustomTimeRange', value: undefined },
                  {
                    key: 'createdAfter',
                    value: getCreatedAfterFromTimeRange(value),
                  },
                  { key: 'finishedBefore', value: undefined },
                ]);
              } else {
                cf.setCustomTimeRange({
                  start:
                    getCreatedAfterFromTimeRange(value) ||
                    new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
                  end: new Date().toISOString(),
                });
              }
            }}
          >
            <SelectTrigger className="w-fit">
              <SelectValue id="timerange" placeholder="Choose time range" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="1h">1 hour</SelectItem>
              <SelectItem value="6h">6 hours</SelectItem>
              <SelectItem value="1d">1 day</SelectItem>
              <SelectItem value="7d">7 days</SelectItem>
              <SelectItem value="custom">Custom</SelectItem>
            </SelectContent>
          </Select>
        </div>
      )}
      {showMetrics && (
        <GetWorkflowChart
          tenantId={tenant.metadata.id}
          createdAfter={cf.filters.createdAfter}
          zoom={(createdAfter, createdBefore) => {
            cf.setCustomTimeRange({ start: createdAfter, end: createdBefore });
          }}
          finishedBefore={cf.filters.finishedBefore}
          refetchInterval={refetchInterval}
        />
      )}
      {showCounts && (
        <div className="flex flex-row justify-between items-center my-4">
          {metrics.length > 0 ? (
            <V1WorkflowRunsMetricsView
              metrics={metrics}
              onViewQueueMetricsClick={() => {
                setViewQueueMetrics(true);
              }}
              showQueueMetrics={showMetrics}
              onClick={(status) => {
                cf.setStatus(status);
              }}
            />
          ) : (
            <Skeleton className="max-w-[800px] w-[40vw] h-8" />
          )}
        </div>
      )}
      {stepDetailSheetState.taskRunId && (
        <Sheet
          open={stepDetailSheetState.isOpen}
          onOpenChange={(isOpen) =>
            setStepDetailSheetState((prev) => ({
              ...prev,
              isOpen,
            }))
          }
        >
          <SheetContent className="w-fit min-w-[56rem] max-w-4xl sm:max-w-2xl z-[60]">
            <TaskRunDetail
              taskRunId={stepDetailSheetState.taskRunId}
              defaultOpenTab={TabOption.Output}
              showViewTaskRunButton
            />
          </SheetContent>
        </Sheet>
      )}
      <DataTable
        emptyState={<>No workflow runs found with the given filters.</>}
        error={workflowKeysError}
        isLoading={isLoading}
        columns={columns(onAdditionalMetadataClick, onTaskRunIdClick)}
        columnVisibility={columnVisibility}
        setColumnVisibility={setColumnVisibility}
        data={tableRows}
        filters={filters}
        actions={actions}
        sorting={sorting}
        setSorting={setSorting}
        columnFilters={cf.filters.columnFilters}
        setColumnFilters={(updaterOrValue) => {
          cf.setColumnFilters(updaterOrValue);
        }}
        pagination={pagination}
        setPagination={setPagination}
        onSetPageSize={setPageSize}
        rowSelection={rowSelection}
        setRowSelection={setRowSelection}
        pageCount={numPages}
        showColumnToggle={true}
        getSubRows={(row) => row.children || []}
        getRowId={getRowId}
      />
    </>
  );
}

const GetWorkflowChart = ({
  tenantId,
  createdAfter,
  finishedBefore,
  refetchInterval,
  zoom,
}: {
  tenantId: string;
  createdAfter?: string;
  finishedBefore?: string;
  refetchInterval?: number;
  zoom: (startTime: string, endTime: string) => void;
}) => {
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
    <div className="">
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
    </div>
  );
};
