import { DataTable } from '@/components/v1/molecules/data-table/data-table.tsx';
import { columns } from './v1/task-runs-columns';
import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  ColumnFiltersState,
  PaginationState,
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
import { useAtom } from 'jotai';
import { lastTimeRangeAtom } from '@/lib/atoms';
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
import {
  CancelTaskRunButton,
  useCancelTaskRuns,
} from '../../task-runs-v1/cancellation';
import {
  ReplayTaskRunButton,
  useReplayTaskRuns,
} from '../../task-runs-v1/replays';

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

type UseColumnFiltersProps = {
  filterVisibility: { [key: string]: boolean };
  createdAfter?: string;
};

const useColumnFilters = ({
  filterVisibility,
  createdAfter: createdAfterProp,
}: UseColumnFiltersProps) => {
  const [searchParams, setSearchParams] = useSearchParams();
  const { workflowKeys, workflowKeysIsLoading, workflowKeysError } =
    useWorkflow();

  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>(() => {
    const filtersParam = searchParams.get('filters');
    if (filtersParam) {
      return JSON.parse(filtersParam);
    }
    return [];
  });

  const workflow = useMemo<string | undefined>(() => {
    const filter = columnFilters.find((filter) => filter.id === 'Workflow');

    if (!filter) {
      return;
    }

    const vals = filter?.value as Array<string>;
    return vals[0];
  }, [columnFilters]);

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
    },
    {
      columnId: 'Metadata',
      title: 'Metadata',
      type: ToolbarType.KeyValue,
    },
  ].filter((filter) => filterVisibility[filter.columnId] != false);

  const [defaultTimeRange, setDefaultTimeRange] = useAtom(lastTimeRangeAtom);
  const [stepDetailSheetState, setStepDetailSheetState] =
    useState<StepDetailSheetState>({
      isOpen: false,
      taskRunId: undefined,
    });

  // customTimeRange does not get set in the atom,
  const [customTimeRange, setCustomTimeRange] = useState<string[] | undefined>(
    () => {
      const timeRangeParam = searchParams.get('customTimeRange');
      if (timeRangeParam) {
        return timeRangeParam.split(',').map((param) => {
          return new Date(param).toISOString();
        });
      }
      return undefined;
    },
  );

  const [createdAfter, setCreatedAfter] = useState<string | undefined>(
    createdAfterProp ||
      getCreatedAfterFromTimeRange(defaultTimeRange) ||
      new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
  );

  const [finishedBefore, setFinishedBefore] = useState<string | undefined>();

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

  // create a timer which updates the createdAfter date every minute
  useEffect(() => {
    const interval = setInterval(() => {
      if (customTimeRange) {
        return;
      }

      setCreatedAfter(
        getCreatedAfterFromTimeRange(defaultTimeRange) ||
          new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
      );
    }, 60 * 1000);

    return () => clearInterval(interval);
  }, [defaultTimeRange, customTimeRange]);

  // whenever the time range changes, update the createdAfter date
  useEffect(() => {
    if (customTimeRange && customTimeRange.length === 2) {
      setCreatedAfter(customTimeRange[0]);
      setFinishedBefore(customTimeRange[1]);
    } else if (defaultTimeRange) {
      setCreatedAfter(getCreatedAfterFromTimeRange(defaultTimeRange));
      setFinishedBefore(undefined);
    }
  }, [defaultTimeRange, customTimeRange, setCreatedAfter]);

  const [pagination, setPagination] = useState<PaginationState>(() => {
    const pageIndex = Number(searchParams.get('pageIndex')) || 0;
    const pageSize = Number(searchParams.get('pageSize')) || 50;
    return { pageIndex, pageSize };
  });

  const setPageSize = (newPageSize: number) => {
    setPagination((prev) => ({
      ...prev,
      pageSize: newPageSize,
    }));
  };

  useEffect(() => {
    const newSearchParams = new URLSearchParams(searchParams);
    if (sorting.length) {
      newSearchParams.set(
        'orderDirection',
        sorting.map((s) => `${s.id}:${s.desc ? 'desc' : 'asc'}`).join(','),
      );
    } else {
      newSearchParams.delete('orderDirection');
    }
    if (columnFilters.length) {
      newSearchParams.set('filters', JSON.stringify(columnFilters));
    } else {
      newSearchParams.delete('filters');
    }
    newSearchParams.set('pageIndex', pagination.pageIndex.toString());
    newSearchParams.set('pageSize', pagination.pageSize.toString());

    if (customTimeRange && customTimeRange.length === 2) {
      newSearchParams.set('customTimeRange', customTimeRange?.join(','));
    } else {
      newSearchParams.delete('customTimeRange');
    }

    if (newSearchParams.toString() !== searchParams.toString()) {
      setSearchParams(newSearchParams, { replace: true });
    }
  }, [
    sorting,
    columnFilters,
    pagination,
    customTimeRange,
    setSearchParams,
    searchParams,
  ]);

  const offset = useMemo(() => {
    if (!pagination) {
      return;
    }

    return pagination.pageIndex * pagination.pageSize;
  }, [pagination]);

  const statuses = useMemo(() => {
    const filter = columnFilters.find((filter) => filter.id === 'status');

    if (!filter) {
      return;
    }

    return filter?.value as Array<V1TaskStatus>;
  }, [columnFilters]);

  const v1TaskFilters = {
    since:
      createdAfter || new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
    until: finishedBefore,
    statuses: columnFilters.find((f) => f.id === 'status')
      ?.value as V1TaskStatus[],
    workflowIds: workflow ? [workflow] : undefined,
    additionalMetadata: columnFilters.find((f) => f.id === 'Metadata')
      ?.value as string[],
  };

  const onAdditionalMetadataClick = ({
    key,
    value,
  }: AdditionalMetadataClick) => {
    setColumnFilters((prev) => {
      const metadataFilter = prev.find((filter) => filter.id === 'Metadata');
      if (metadataFilter) {
        prev = prev.filter((filter) => filter.id !== 'Metadata');
      }
      return [
        ...prev,
        {
          id: 'Metadata',
          value: [`${key}:${value}`],
        },
      ];
    });
  };

  return {
    workflow,
    workflowKeys,
    workflowKeysIsLoading,
    workflowKeysError,
    columnFilters,
    setColumnFilters,
    filters,
    sorting,
    setSorting,
    pagination,
    setPagination,
    offset,
    statuses,
    customTimeRange,
    setCustomTimeRange,
    createdAfter,
    setCreatedAfter,
    finishedBefore,
    setFinishedBefore,
    stepDetailSheetState,
    setStepDetailSheetState,
    setPageSize,
    setDefaultTimeRange,
    v1TaskFilters,
    defaultTimeRange,
    onAdditionalMetadataClick,
  };
};

const useTaskRunRows = ({
  filterVisibility,
  createdAfter: createdAfterProp,
  workerId,
  refetchInterval,
  rowSelection,
}: UseColumnFiltersProps & {
  workerId?: string;
  refetchInterval: number;
  rowSelection: RowSelectionState;
}) => {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const filters = useColumnFilters({
    filterVisibility,
    createdAfter: createdAfterProp,
  });

  const listTasksQuery = useQuery({
    ...queries.v1WorkflowRuns.list(tenant.metadata.id, {
      offset: filters.offset,
      limit: filters.pagination.pageSize,
      statuses: filters.statuses,
      workflow_ids: filters.workflow ? [filters.workflow] : [],
      since:
        filters.createdAfter ||
        new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
      until: filters.finishedBefore,
      additional_metadata: filters.columnFilters.find(
        (f) => f.id === 'Metadata',
      )?.value as string[],
      worker_id: workerId,
      only_tasks: !!workerId,
    }),
    placeholderData: (prev) => prev,
    refetchInterval,
  });

  const tasks = listTasksQuery.data;
  const tableRows = tasks?.rows || [];

  const selectedRuns = useMemo(() => {
    return Object.entries(rowSelection)
      .filter(([, selected]) => !!selected)
      .map(([id]) => {
        const findRow = (rows: V1TaskSummary[]): V1TaskSummary | undefined => {
          for (const row of rows) {
            if (row.metadata.id === id) {
              return row;
            }
            if (row.children) {
              const childRow = findRow(row.children);
              if (childRow) {
                return childRow;
              }
            }
          }
          return undefined;
        };

        return findRow(tableRows);
      })
      .filter((row) => row !== undefined) as V1TaskSummary[];
  }, [rowSelection, tableRows]);

  return {
    tableRows,
    selectedRuns,
    refetchRuns: listTasksQuery.refetch,
    isLoading: listTasksQuery.isFetching || listTasksQuery.isLoading,
    isError: listTasksQuery.isError,
    numPages: tasks?.pagination.num_pages || 0,
  };
};

const useMetrics = ({
  filterVisibility,
  createdAfter: createdAfterProp,
  refetchInterval,
}: UseColumnFiltersProps & {
  refetchInterval: number;
}) => {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const filters = useColumnFilters({
    filterVisibility,
    createdAfter: createdAfterProp,
  });

  const metricsQuery = useQuery({
    ...queries.v1TaskRuns.metrics(tenant.metadata.id, {
      since:
        filters.createdAfter ||
        new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
      workflow_ids: filters.workflow ? [filters.workflow] : [],
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

  return {
    metrics,
    tenantMetrics,
    isLoading: metricsQuery.isLoading || tenantMetricsQuery.isLoading,
    isError: metricsQuery.isError || tenantMetricsQuery.isError,
    refetch: () => {
      metricsQuery.refetch();
      tenantMetricsQuery.refetch();
    },
  };
};

export function TaskRunsTable({
  workerId,
  createdAfter: createdAfterProp,
  initColumnVisibility = {},
  filterVisibility = {},
  refetchInterval = 5000,
  showMetrics = false,
  showCounts = true,
}: TaskRunsTableProps) {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const filters = useColumnFilters({
    filterVisibility,
    createdAfter: createdAfterProp,
  });

  const [viewQueueMetrics, setViewQueueMetrics] = useState(false);
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});

  const [columnVisibility, setColumnVisibility] =
    useState<VisibilityState>(initColumnVisibility);

  const {
    tableRows,
    selectedRuns,
    refetchRuns,
    isLoading: isRunsLoading,
    isError: isRunsError,
    numPages,
  } = useTaskRunRows({
    filterVisibility,
    createdAfter: createdAfterProp,
    workerId,
    refetchInterval,
    rowSelection,
  });

  const {
    metrics,
    tenantMetrics,
    isLoading: isMetricsLoading,
    refetch: refetchMetrics,
  } = useMetrics({
    filterVisibility,
    createdAfter: createdAfterProp,
    refetchInterval,
  });

  const { handleCancelTaskRun } = useCancelTaskRuns();
  const { handleReplayTaskRun } = useReplayTaskRuns();

  const onTaskRunIdClick = useCallback(
    (taskRunId: string) => {
      filters.setStepDetailSheetState({
        taskRunId,
        isOpen: true,
      });
    },
    [filters],
  );

  const [rotate, setRotate] = useState(false);

  const refetch = () => {
    refetchRuns();
    refetchMetrics();
  };

  const hasRowsSelected = Object.values(rowSelection).some(
    (selected) => !!selected,
  );
  const hasTaskFiltersSelected = Object.values(filters.v1TaskFilters).some(
    (filter) => !!filter,
  );

  const actions = [
    <CancelTaskRunButton
      key="cancel"
      disabled={!(hasRowsSelected || hasTaskFiltersSelected)}
      handleCancelTaskRun={() => {
        const idsToCancel = selectedRuns.map((run) => run?.metadata.id);

        if (idsToCancel.length) {
          handleCancelTaskRun({
            externalIds: idsToCancel,
          });
        } else if (
          Object.values(filters.v1TaskFilters).some((filter) => !!filter)
        ) {
          handleCancelTaskRun({
            filter: filters.v1TaskFilters,
          });
        }
      }}
    />,
    <ReplayTaskRunButton
      key="replay"
      disabled={!(hasRowsSelected || hasTaskFiltersSelected)}
      handleReplayTaskRun={() => {
        const idsToReplay = selectedRuns.map((run) => run?.metadata.id);

        if (idsToReplay.length) {
          handleReplayTaskRun({
            externalIds: idsToReplay,
          });
        } else if (
          Object.values(filters.v1TaskFilters).some((filter) => !!filter)
        ) {
          handleReplayTaskRun({
            filter: filters.v1TaskFilters,
          });
        }
      }}
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

  const isLoading = isRunsLoading || isMetricsLoading;

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
            {isMetricsLoading && <Skeleton className="w-full h-36" />}
          </DialogContent>
        </Dialog>
      )}
      {!createdAfterProp && (
        <div className="flex flex-row justify-end items-center my-4 gap-2">
          {filters.customTimeRange && [
            <Button
              key="clear"
              onClick={() => {
                filters.setCustomTimeRange(undefined);
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
                filters.createdAfter
                  ? new Date(filters.createdAfter)
                  : undefined
              }
              setDate={(date) => {
                filters.setCreatedAfter(date?.toISOString());
              }}
            />,
            <DateTimePicker
              key="before"
              label="Before"
              date={
                filters.finishedBefore
                  ? new Date(filters.finishedBefore)
                  : undefined
              }
              setDate={(date) => {
                filters.setFinishedBefore(date?.toISOString());
              }}
            />,
          ]}
          <Select
            value={
              filters.customTimeRange ? 'custom' : filters.defaultTimeRange
            }
            onValueChange={(value) => {
              if (value !== 'custom') {
                filters.setDefaultTimeRange(value);
                filters.setCustomTimeRange(undefined);
              } else {
                filters.setCustomTimeRange([
                  getCreatedAfterFromTimeRange(value) ||
                    new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
                  new Date().toISOString(),
                ]);
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
          createdAfter={filters.createdAfter}
          zoom={(createdAfter, createdBefore) => {
            filters.setCustomTimeRange([createdAfter, createdBefore]);
          }}
          finishedBefore={filters.finishedBefore}
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
                filters.setColumnFilters((prev) => {
                  const statusFilter = prev.find(
                    (filter) => filter.id === 'status',
                  );
                  if (statusFilter) {
                    prev = prev.filter((filter) => filter.id !== 'status');
                  }

                  if (
                    JSON.stringify(statusFilter?.value) ===
                    JSON.stringify([status])
                  ) {
                    return prev;
                  }

                  return [
                    ...prev,
                    {
                      id: 'status',
                      value: [status],
                    },
                  ];
                });
              }}
            />
          ) : (
            <Skeleton className="max-w-[800px] w-[40vw] h-8" />
          )}
        </div>
      )}
      {filters.stepDetailSheetState.taskRunId && (
        <Sheet
          open={filters.stepDetailSheetState.isOpen}
          onOpenChange={(isOpen) =>
            filters.setStepDetailSheetState((prev) => ({
              ...prev,
              isOpen,
            }))
          }
        >
          <SheetContent className="w-fit min-w-[56rem] max-w-4xl sm:max-w-2xl z-[60]">
            <TaskRunDetail
              taskRunId={filters.stepDetailSheetState.taskRunId}
              defaultOpenTab={TabOption.Output}
              showViewTaskRunButton
            />
          </SheetContent>
        </Sheet>
      )}
      <DataTable
        emptyState={<>No workflow runs found with the given filters.</>}
        error={isRunsError ? Error('Something went wrong') : undefined}
        isLoading={isLoading}
        columns={columns(filters.onAdditionalMetadataClick, onTaskRunIdClick)}
        columnVisibility={columnVisibility}
        setColumnVisibility={setColumnVisibility}
        data={tableRows}
        filters={filters.filters}
        actions={actions}
        sorting={filters.sorting}
        setSorting={filters.setSorting}
        columnFilters={filters.columnFilters}
        setColumnFilters={filters.setColumnFilters}
        pagination={filters.pagination}
        setPagination={filters.setPagination}
        onSetPageSize={filters.setPageSize}
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
