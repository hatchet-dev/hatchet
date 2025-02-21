import { DataTable } from '@/components/v1/molecules/data-table/data-table.tsx';
import { columns } from './v2/workflow-runs-columns';
import { useEffect, useMemo, useState } from 'react';
import {
  ColumnFiltersState,
  PaginationState,
  RowSelectionState,
  SortingState,
  VisibilityState,
} from '@tanstack/react-table';
import { useMutation, useQuery } from '@tanstack/react-query';
import invariant from 'tiny-invariant';
import api, {
  queries,
  ReplayWorkflowRunsRequest,
  V2TaskStatus,
  V2WorkflowRun,
} from '@/lib/api';
import { TenantContextType } from '@/lib/outlet';
import { useOutletContext, useSearchParams } from 'react-router-dom';
import {
  FilterOption,
  ToolbarFilters,
  ToolbarType,
} from '@/components/v1/molecules/data-table/data-table-toolbar';
import { Button } from '@/components/v1/ui/button';
import {
  ArrowPathIcon,
  ArrowPathRoundedSquareIcon,
  XCircleIcon,
  XMarkIcon,
} from '@heroicons/react/24/outline';
import { V2WorkflowRunsMetricsView } from './workflow-runs-metrics';
import queryClient from '@/query-client';
import { useApiError } from '@/lib/hooks';
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

export interface WorkflowRunsTableProps {
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

// TODO: Clean this up
export type ListableWorkflowRun = V2WorkflowRun & {
  workflowName: string | undefined;
  triggeredBy: string;
  workflowVersionId: string;
};

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

export function WorkflowRunsTable({
  workflowId,
  workerId,
  createdAfter: createdAfterProp,
  initColumnVisibility = {},
  filterVisibility = {},
  refetchInterval = 5000,
  showMetrics = false,
  showCounts = true,
}: WorkflowRunsTableProps) {
  const [searchParams, setSearchParams] = useSearchParams();
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const [viewQueueMetrics, setViewQueueMetrics] = useState(false);

  const [defaultTimeRange, setDefaultTimeRange] = useAtom(lastTimeRangeAtom);

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

  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>(() => {
    const filtersParam = searchParams.get('filters');
    if (filtersParam) {
      return JSON.parse(filtersParam);
    }
    return [];
  });

  const [columnVisibility, setColumnVisibility] =
    useState<VisibilityState>(initColumnVisibility);

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

  const workflow = useMemo<string | undefined>(() => {
    if (workflowId) {
      return workflowId;
    }

    const filter = columnFilters.find((filter) => filter.id === 'Workflow');

    if (!filter) {
      return;
    }

    const vals = filter?.value as Array<string>;
    return vals[0];
  }, [columnFilters, workflowId]);

  const statuses = useMemo(() => {
    const filter = columnFilters.find((filter) => filter.id === 'status');

    if (!filter) {
      return;
    }

    return filter?.value as Array<V2TaskStatus>;
  }, [columnFilters]);

  const listTasksQuery = useQuery({
    ...queries.v2Tasks.list(tenant.metadata.id, {
      offset,
      limit: pagination.pageSize,
      statuses,
      workflow_ids: workflow ? [workflow] : [],
      worker_id: workerId,
      since:
        createdAfter ||
        new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
      additional_metadata: columnFilters.find((f) => f.id === 'Metadata')
        ?.value as string[],
    }),
    placeholderData: (prev) => prev,
    refetchInterval,
  });

  const metricsQuery = useQuery({
    ...queries.v2TaskRuns.metrics(tenant.metadata.id, {
      since:
        createdAfter ||
        new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
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

  const {
    data: workflowKeys,
    isLoading: workflowKeysIsLoading,
    error: workflowKeysError,
  } = useQuery({
    ...queries.workflows.list(tenant.metadata.id, { limit: 200 }),
  });

  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});

  const selectedRuns = useMemo(() => {
    return Object.entries(rowSelection)
      .filter(([, selected]) => !!selected)
      .map(([id]) => (listTasksQuery.data?.rows || [])[Number(id)]);
  }, [listTasksQuery.data?.rows, rowSelection]);

  const { handleApiError } = useApiError({});

  const cancelWorkflowRunMutation = useMutation({
    mutationKey: ['workflow-run:cancel', tenant.metadata.id, selectedRuns],
    mutationFn: async () => {
      const tenantId = tenant.metadata.id;
      const workflowRunIds = selectedRuns.map((wr) => wr.metadata.id);

      invariant(tenantId, 'has tenantId');
      invariant(workflowRunIds, 'has runIds');

      const res = await api.workflowRunCancel(tenantId, {
        workflowRunIds,
      });

      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queries.workflowRuns.list(tenant.metadata.id, {}).queryKey,
      });
    },
    onError: handleApiError,
  });

  const replayWorkflowRunsMutation = useMutation({
    mutationKey: ['workflow-run:update:replay', tenant.metadata.id],
    mutationFn: async (data: ReplayWorkflowRunsRequest) => {
      await api.workflowRunUpdateReplay(tenant.metadata.id, data);
    },
    onSuccess: () => {
      setRowSelection({});

      // bit hacky, but workflow run statuses aren't updated immediately after replay
      setTimeout(() => {
        listTasksQuery.refetch();
      }, 1000);
    },
    onError: handleApiError,
  });

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
        value: V2TaskStatus.COMPLETED,
        label: 'Succeeded',
      },
      {
        value: V2TaskStatus.FAILED,
        label: 'Failed',
      },
      {
        value: V2TaskStatus.RUNNING,
        label: 'Running',
      },
      {
        value: V2TaskStatus.QUEUED,
        label: 'Queued',
      },
      // {
      //   value: V2TaskStatus.CANCELLED,
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

  const [rotate, setRotate] = useState(false);

  const refetch = () => {
    listTasksQuery.refetch();
    tenantMetricsQuery.refetch();
    metricsQuery.refetch();
  };

  const actions = [
    <Button
      // disabled={!Object.values(rowSelection).some((selected) => !!selected)}
      disabled
      key="cancel"
      className="h-8 px-2 lg:px-3"
      size="sm"
      onClick={() => {
        cancelWorkflowRunMutation.mutate();
      }}
      variant={'outline'}
      aria-label="Cancel Selected Runs"
    >
      <XMarkIcon className={`mr-2 h-4 w-4 transition-transform`} />
      Cancel
    </Button>,
    <Button
      // disabled={!Object.values(rowSelection).some((selected) => !!selected)}
      disabled
      key="replay"
      className="h-8 px-2 lg:px-3"
      size="sm"
      onClick={() => {
        replayWorkflowRunsMutation.mutate({
          workflowRunIds: selectedRuns.map((run) => run.metadata.id),
        });
      }}
      variant={'outline'}
      aria-label="Replay Selected Runs"
    >
      <ArrowPathRoundedSquareIcon className="mr-2 h-4 w-4 transition-transform" />
      Replay
    </Button>,
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
    listTasksQuery.isFetching ||
    workflowKeysIsLoading ||
    metricsQuery.isLoading;

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

  const data: ListableWorkflowRun[] = (listTasksQuery.data?.rows || []).map(
    (row) => ({
      ...row,
      workflowVersionId: 'first version',
      triggeredBy: 'manual',
      workflowName: workflowKeys?.rows?.find(
        (r) => r.metadata.id == row.workflowId,
      )?.name,
    }),
  );

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
            {tenantMetricsQuery.data?.queues && (
              <CodeHighlighter
                language="json"
                code={JSON.stringify(
                  tenantMetricsQuery.data?.queues || '{}',
                  null,
                  2,
                )}
              />
            )}
            {tenantMetricsQuery.isLoading && (
              <Skeleton className="w-full h-36" />
            )}
          </DialogContent>
        </Dialog>
      )}
      {!createdAfterProp && (
        <div className="flex flex-row justify-end items-center my-4 gap-2">
          {customTimeRange && [
            <Button
              key="clear"
              onClick={() => {
                setCustomTimeRange(undefined);
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
              date={createdAfter ? new Date(createdAfter) : undefined}
              setDate={(date) => {
                setCreatedAfter(date?.toISOString());
              }}
            />,
            <DateTimePicker
              key="before"
              label="Before"
              date={finishedBefore ? new Date(finishedBefore) : undefined}
              setDate={(date) => {
                setFinishedBefore(date?.toISOString());
              }}
            />,
          ]}
          <Select
            value={customTimeRange ? 'custom' : defaultTimeRange}
            onValueChange={(value) => {
              if (value !== 'custom') {
                setDefaultTimeRange(value);
                setCustomTimeRange(undefined);
              } else {
                setCustomTimeRange([
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
          createdAfter={createdAfter}
          zoom={(createdAfter, createdBefore) => {
            setCustomTimeRange([createdAfter, createdBefore]);
          }}
          finishedBefore={finishedBefore}
          refetchInterval={refetchInterval}
        />
      )}
      {showCounts && (
        <div className="flex flex-row justify-between items-center my-4">
          {metrics.length > 0 ? (
            <V2WorkflowRunsMetricsView
              metrics={metrics}
              onViewQueueMetricsClick={() => {
                setViewQueueMetrics(true);
              }}
              showQueueMetrics={showMetrics}
              onClick={(status) => {
                setColumnFilters((prev) => {
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
      <DataTable
        emptyState={<>No workflow runs found with the given filters.</>}
        error={workflowKeysError}
        isLoading={isLoading}
        columns={columns(onAdditionalMetadataClick)}
        columnVisibility={columnVisibility}
        setColumnVisibility={setColumnVisibility}
        // TODO: This is a hack - fix this type
        data={data}
        filters={filters}
        actions={actions}
        sorting={sorting}
        setSorting={setSorting}
        columnFilters={columnFilters}
        setColumnFilters={setColumnFilters}
        pagination={pagination}
        setPagination={setPagination}
        onSetPageSize={setPageSize}
        rowSelection={rowSelection}
        setRowSelection={setRowSelection}
        pageCount={listTasksQuery.data?.pagination?.num_pages || 0}
        showColumnToggle={true}
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
    ...queries.v2TaskRuns.pointMetrics(tenantId, {
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
