import { DataTable } from '@/components/v1/molecules/data-table/data-table.tsx';
import { useEffect, useMemo, useState } from 'react';
import {
  ColumnFiltersState,
  PaginationState,
  RowSelectionState,
  SortingState,
  VisibilityState,
} from '@tanstack/react-table';
import { useQuery } from '@tanstack/react-query';
import invariant from 'tiny-invariant';
import {
  ScheduledRunStatus,
  ScheduledWorkflows,
  ScheduledWorkflowsOrderByField,
  WorkflowRunOrderByDirection,
  queries,
} from '@/lib/api';
import { TenantContextType } from '@/lib/outlet';
import { useOutletContext, useSearchParams } from 'react-router-dom';
import {
  FilterOption,
  ToolbarFilters,
  ToolbarType,
} from '@/components/v1/molecules/data-table/data-table-toolbar';
import { Button } from '@/components/v1/ui/button';
import { ArrowPathIcon } from '@heroicons/react/24/outline';
import { columns } from './scheduled-runs-columns';
import { DeleteScheduledRun } from './delete-scheduled-runs';

export interface ScheduledWorkflowRunsTableProps {
  createdAfter?: string;
  createdBefore?: string;
  workflowId?: string;
  parentWorkflowRunId?: string;
  parentStepRunId?: string;
  initColumnVisibility?: VisibilityState;
  filterVisibility?: { [key: string]: boolean };
  refetchInterval?: number;
  showMetrics?: boolean;
}

export function ScheduledRunsTable({
  workflowId,
  initColumnVisibility = {
    createdAt: false,
  },
  filterVisibility = {},
  parentWorkflowRunId,
  parentStepRunId,
  refetchInterval = 5000,
}: ScheduledWorkflowRunsTableProps) {
  const [searchParams, setSearchParams] = useSearchParams();
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

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

    if (newSearchParams.toString() !== searchParams.toString()) {
      setSearchParams(newSearchParams, { replace: true });
    }
  }, [sorting, columnFilters, pagination, setSearchParams, searchParams]);

  const [pageSize, setPageSize] = useState<number>(50);

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

    return filter?.value as Array<ScheduledRunStatus>;
  }, [columnFilters]);

  const AdditionalMetadataFilter = useMemo(() => {
    const filter = columnFilters.find((filter) => filter.id === 'Metadata');

    if (!filter) {
      return;
    }

    return filter?.value as Array<string>;
  }, [columnFilters]);

  const orderByDirection = useMemo(():
    | WorkflowRunOrderByDirection
    | undefined => {
    if (!sorting.length) {
      return;
    }

    return sorting[0]?.desc
      ? WorkflowRunOrderByDirection.DESC
      : WorkflowRunOrderByDirection.ASC;
  }, [sorting]);

  const orderByField = useMemo(():
    | ScheduledWorkflowsOrderByField
    | undefined => {
    if (!sorting.length) {
      return;
    }

    switch (sorting[0]?.id) {
      case 'createdAt':
        return ScheduledWorkflowsOrderByField.CreatedAt;
      case 'triggerAt':
      default:
        return ScheduledWorkflowsOrderByField.TriggerAt;
    }
  }, [sorting]);

  const listWorkflowRunsQuery = useQuery({
    ...queries.scheduledRuns.list(tenant.metadata.id, {
      offset,
      limit: pageSize,
      statuses,
      workflowId: workflow,
      parentWorkflowRunId,
      parentStepRunId,
      orderByDirection,
      orderByField,
      additionalMetadata: AdditionalMetadataFilter,
    }),
    placeholderData: (prev) => prev,
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

  // const selectedRuns = useMemo(() => {
  //   return Object.entries(rowSelection)
  //     .filter(([, selected]) => !!selected)
  //     .map(([id]) => (listWorkflowRunsQuery.data?.rows || [])[Number(id)]);
  // }, [listWorkflowRunsQuery.data?.rows, rowSelection]);

  // const { handleApiError } = useApiError({});

  // const cancelWorkflowRunMutation = useMutation({
  //   mutationKey: ['workflow-run:cancel', tenant.metadata.id, selectedRuns],
  //   mutationFn: async () => {
  //     const tenantId = tenant.metadata.id;
  //     const workflowRunIds = selectedRuns.map((wr) => wr.metadata.id);

  //     invariant(tenantId, 'has tenantId');
  //     invariant(workflowRunIds, 'has runIds');

  //     const res = await api.workflowRunCancel(tenantId, {
  //       workflowRunIds,
  //     });

  //     return res.data;
  //   },
  //   onSuccess: () => {
  //     queryClient.invalidateQueries({
  //       queryKey: queries.workflowRuns.list(tenant.metadata.id, {}).queryKey,
  //     });
  //   },
  //   onError: handleApiError,
  // });

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
        value: ScheduledRunStatus.SCHEDULED,
        label: 'Scheduled',
      },
      {
        value: ScheduledRunStatus.SUCCEEDED,
        label: 'Succeeded',
      },
      {
        value: ScheduledRunStatus.FAILED,
        label: 'Failed',
      },
      {
        value: ScheduledRunStatus.RUNNING,
        label: 'Running',
      },
      {
        value: ScheduledRunStatus.QUEUED,
        label: 'Queued',
      },
      {
        value: ScheduledRunStatus.PENDING,
        label: 'Pending',
      },
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
    listWorkflowRunsQuery.refetch();
  };

  const actions = [
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

  const [showScheduledRunRevoke, setShowScheduledRunRevoke] = useState<
    ScheduledWorkflows | undefined
  >(undefined);

  const isLoading = listWorkflowRunsQuery.isFetching || workflowKeysIsLoading;

  return (
    <>
      <DeleteScheduledRun
        tenant={tenant.metadata.id}
        scheduledRun={showScheduledRunRevoke}
        setShowScheduledRunRevoke={setShowScheduledRunRevoke}
        onSuccess={() => {
          refetch();
          setShowScheduledRunRevoke(undefined);
        }}
      />
      <DataTable
        emptyState={<>No workflow runs found with the given filters.</>}
        error={workflowKeysError}
        isLoading={isLoading}
        columns={columns({
          onDeleteClick: (row) => {
            setShowScheduledRunRevoke(row);
          },
        })}
        columnVisibility={columnVisibility}
        setColumnVisibility={setColumnVisibility}
        data={listWorkflowRunsQuery.data?.rows || []}
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
        pageCount={listWorkflowRunsQuery.data?.pagination?.num_pages || 0}
        showColumnToggle={true}
      />
    </>
  );
}
