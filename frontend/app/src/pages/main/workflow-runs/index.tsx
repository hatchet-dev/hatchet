import { DataTable } from '@/components/molecules/data-table/data-table.tsx';
import { columns } from './components/workflow-runs-columns';
import { Separator } from '@/components/ui/separator';
import { useMemo, useState } from 'react';
import {
  ColumnFiltersState,
  PaginationState,
  SortingState,
} from '@tanstack/react-table';
import { useQuery } from '@tanstack/react-query';
import invariant from 'tiny-invariant';
import { WorkflowRunStatus, queries } from '@/lib/api';
import { Loading } from '@/components/ui/loading.tsx';
import { TenantContextType } from '@/lib/outlet';
import { useOutletContext } from 'react-router-dom';
import { FilterOption } from '@/components/molecules/data-table/data-table-toolbar';

export default function WorkflowRuns() {
  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <h2 className="text-2xl font-bold leading-tight text-foreground">
          Workflow Runs
        </h2>
        <Separator className="my-4" />
        <WorkflowRunsTable />
      </div>
    </div>
  );
}

function WorkflowRunsTable() {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const [sorting, setSorting] = useState<SortingState>([]);
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 50,
  });
  const [pageSize, setPageSize] = useState<number>(50);

  const offset = useMemo(() => {
    if (!pagination) {
      return;
    }

    return pagination.pageIndex * pagination.pageSize;
  }, [pagination]);

  const workflows = useMemo(() => {
    const filter = columnFilters.find((filter) => filter.id === 'Workflow');

    if (!filter) {
      return;
    }

    return filter?.value as Array<string>;
  }, [columnFilters]);

  const statuses = useMemo(() => {
    const filter = columnFilters.find((filter) => filter.id === 'status');

    if (!filter) {
      return;
    }

    return filter?.value as Array<WorkflowRunStatus>;
  }, [columnFilters]);

  const listWorkflowRunsQuery = useQuery({
    ...queries.workflowRuns.list(tenant.metadata.id, {
      offset,
      limit: pageSize,
      statuses,
      workflowId: workflows ? workflows[0] : undefined,
    }),
  });

  const {
    data: workflowKeys,
    isLoading: workflowKeysIsLoading,
    error: workflowKeysError,
  } = useQuery({
    ...queries.workflows.list(tenant.metadata.id),
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

  if (listWorkflowRunsQuery.isLoading) {
    return <Loading />;
  }

  return (
    <DataTable
      error={workflowKeysError}
      isLoading={workflowKeysIsLoading}
      columns={columns}
      data={listWorkflowRunsQuery.data?.rows || []}
      filters={[
        {
          columnId: 'Workflow',
          title: 'Workflow',
          options: workflowKeyFilters,
          type: 'radio',
        },
        {
          columnId: 'status',
          title: 'Status',
          options: workflowRunStatusFilters,
        },
      ]}
      sorting={sorting}
      setSorting={setSorting}
      columnFilters={columnFilters}
      setColumnFilters={setColumnFilters}
      pagination={pagination}
      setPagination={setPagination}
      onSetPageSize={setPageSize}
      pageCount={listWorkflowRunsQuery.data?.pagination?.num_pages || 0}
    />
  );
}
