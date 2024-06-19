import { DataTable } from '@/components/molecules/data-table/data-table.tsx';
import { columns } from './workflow-runs-columns';
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
import api, { WorkflowRunStatus, queries } from '@/lib/api';
import { Loading } from '@/components/ui/loading.tsx';
import { TenantContextType } from '@/lib/outlet';
import { useOutletContext, useSearchParams } from 'react-router-dom';
import {
  FilterOption,
  ToolbarFilters,
  ToolbarType,
} from '@/components/molecules/data-table/data-table-toolbar';
import { Button } from '@/components/ui/button';
import { ArrowPathIcon, XMarkIcon } from '@heroicons/react/24/outline';
import { WorkflowRunsMetricsView } from './workflow-runs-metrics';
import queryClient from '@/query-client';
import { useApiError } from '@/lib/hooks';

export interface WorkflowRunsTableProps {
  workflowId?: string;
  parentWorkflowRunId?: string;
  parentStepRunId?: string;
  initColumnVisibility?: VisibilityState;
  filterVisibility?: { [key: string]: boolean };
  refetchInterval?: number;
}

export function WorkflowRunsTable({
  workflowId,
  initColumnVisibility = {},
  filterVisibility = {},
  parentWorkflowRunId,
  parentStepRunId,
  refetchInterval = 5000,
}: WorkflowRunsTableProps) {
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
        'sort',
        sorting.map((s) => `${s.id}:${s.desc ? 'desc' : 'asc'}`).join(','),
      );
    } else {
      newSearchParams.delete('sort');
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

    return filter?.value as Array<WorkflowRunStatus>;
  }, [columnFilters]);

  const AdditionalMetadataFilter = useMemo(() => {
    const filter = columnFilters.find((filter) => filter.id === 'Metadata');

    if (!filter) {
      return;
    }

    return filter?.value as Array<string>;
  }, [columnFilters]);

  const listWorkflowRunsQuery = useQuery({
    ...queries.workflowRuns.list(tenant.metadata.id, {
      offset,
      limit: pageSize,
      statuses,
      workflowId: workflow,
      parentWorkflowRunId,
      parentStepRunId,
      additionalMetadata: AdditionalMetadataFilter,
    }),
    refetchInterval,
  });

  const metricsQuery = useQuery({
    ...queries.workflowRuns.metrics(tenant.metadata.id, {
      workflowId: workflow,
      parentWorkflowRunId,
      parentStepRunId,
      additionalMetadata: AdditionalMetadataFilter,
    }),
    refetchInterval,
  });

  const {
    data: workflowKeys,
    isLoading: workflowKeysIsLoading,
    error: workflowKeysError,
  } = useQuery({
    ...queries.workflows.list(tenant.metadata.id),
  });

  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});

  const selectedRuns = useMemo(() => {
    return Object.entries(rowSelection)
      .filter(([, selected]) => !!selected)
      .map(([id]) => (listWorkflowRunsQuery.data?.rows || [])[Number(id)]);
  }, [listWorkflowRunsQuery.data?.rows, rowSelection]);

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
        value: WorkflowRunStatus.SUCCEEDED,
        label: 'Succeeded',
      },
      {
        value: WorkflowRunStatus.FAILED,
        label: 'Failed',
      },
      {
        value: WorkflowRunStatus.RUNNING,
        label: 'Running',
      },
      {
        value: WorkflowRunStatus.QUEUED,
        label: 'Queued',
      },
      {
        value: WorkflowRunStatus.PENDING,
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
    metricsQuery.refetch();
  };

  const actions = [
    <Button
      disabled={!Object.values(rowSelection).some((selected) => !!selected)}
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
      Cancel Selected Runs
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

  if (listWorkflowRunsQuery.isLoading) {
    return <Loading />;
  }

  return (
    <>
      {metricsQuery.data && (
        <div className="mb-4">
          <WorkflowRunsMetricsView metrics={metricsQuery.data} />
        </div>
      )}
      <DataTable
        error={workflowKeysError}
        isLoading={workflowKeysIsLoading}
        columns={columns}
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
      />
    </>
  );
}
