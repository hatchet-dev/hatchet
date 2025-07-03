import { useEffect, useMemo, useState } from 'react';
import {
  ColumnFiltersState,
  PaginationState,
  RowSelectionState,
  SortingState,
  VisibilityState,
} from '@tanstack/react-table';
import { useQuery } from '@tanstack/react-query';
import {
  CronWorkflows,
  CronWorkflowsOrderByField,
  WorkflowRunOrderByDirection,
  queries,
} from '@/lib/api';
import { useSearchParams } from 'react-router-dom';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { columns } from './recurring-columns';
import { Button } from '@/components/v1/ui/button';
import { ArrowPathIcon } from '@heroicons/react/24/outline';
import { DeleteCron } from './delete-cron';
import {
  FilterOption,
  ToolbarFilters,
  ToolbarType,
} from '@/components/v1/molecules/data-table/data-table-toolbar';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { TriggerWorkflowForm } from '../../workflows/$workflow/components/trigger-workflow-form';

export function CronsTable() {
  const { tenantId } = useCurrentTenantId();
  const [searchParams, setSearchParams] = useSearchParams();
  const [triggerWorkflow, setTriggerWorkflow] = useState(false);
  const [selectedJobId, setSelectedJobId] = useState<string | null>(null);

  const [sorting, setSorting] = useState<SortingState>(() => {
    const sortParam = searchParams.get('sort');
    if (sortParam) {
      const [id, desc] = sortParam.split(':');
      return [{ id, desc: desc === 'desc' }];
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

  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});

  const [pagination, setPagination] = useState<PaginationState>(() => {
    const pageIndex = Number(searchParams.get('pageIndex')) || 0;
    const pageSize = Number(searchParams.get('pageSize')) || 50;
    return { pageIndex, pageSize };
  });

  const [pageSize, setPageSize] = useState<number>(
    Number(searchParams.get('pageSize')) || 50,
  );
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});

  useEffect(() => {
    const newSearchParams = new URLSearchParams(searchParams);

    newSearchParams.set(
      'sort',
      sorting.map((s) => `${s.id}:${s.desc ? 'desc' : 'asc'}`).join(','),
    );
    newSearchParams.set('filters', JSON.stringify(columnFilters));
    newSearchParams.set('pageIndex', pagination.pageIndex.toString());
    newSearchParams.set('pageSize', pagination.pageSize.toString());
    setSearchParams(newSearchParams);
  }, [sorting, columnFilters, pagination, setSearchParams, searchParams]);

  const workflow = useMemo<string | undefined>(() => {
    const filter = columnFilters.find((filter) => filter.id === 'Workflow');

    if (!filter) {
      return;
    }

    const vals = filter?.value as Array<string>;
    return vals[0];
  }, [columnFilters]);

  const orderByDirection = useMemo<
    WorkflowRunOrderByDirection | undefined
  >(() => {
    if (!sorting.length) {
      return;
    }

    return sorting[0]?.desc
      ? WorkflowRunOrderByDirection.DESC
      : WorkflowRunOrderByDirection.ASC;
  }, [sorting]);

  const orderByField = useMemo<CronWorkflowsOrderByField | undefined>(() => {
    if (!sorting.length) {
      return;
    }

    switch (sorting[0]?.id) {
      case 'createdAt':
        return CronWorkflowsOrderByField.CreatedAt;
      case 'name':
        return CronWorkflowsOrderByField.Name;
      default:
        return CronWorkflowsOrderByField.CreatedAt;
    }
  }, [sorting]);

  const offset = useMemo(() => {
    if (!pagination) {
      return;
    }

    return pagination.pageIndex * pagination.pageSize;
  }, [pagination]);

  const {
    data,
    isLoading: queryIsLoading,
    error: queryError,
    refetch,
  } = useQuery({
    ...queries.cronJobs.list(tenantId, {
      orderByField,
      orderByDirection,
      offset,
      limit: pageSize,
      workflowId: workflow,
      additionalMetadata: columnFilters.find(
        (filter) => filter.id === 'Metadata',
      )?.value as string[] | undefined,
    }),
    refetchInterval: selectedJobId ? false : 2000,
  });

  const [showDeleteCron, setShowDeleteCron] = useState<
    CronWorkflows | undefined
  >();

  const handleDeleteClick = (cron: CronWorkflows) => {
    setShowDeleteCron(cron);
  };

  const handleConfirmDelete = () => {
    if (showDeleteCron) {
      setShowDeleteCron(undefined);
      refetch();
    }
  };

  const { data: workflowKeys } = useQuery({
    ...queries.workflows.list(tenantId, { limit: 200 }),
    refetchInterval: selectedJobId ? false : 2000,
  });

  const workflowKeyFilters = useMemo((): FilterOption[] => {
    return (
      workflowKeys?.rows?.map((key) => ({
        value: key.metadata.id,
        label: key.name,
      })) || []
    );
  }, [workflowKeys]);

  const filters: ToolbarFilters = [
    {
      columnId: 'Workflow',
      title: 'Workflow',
      options: workflowKeyFilters,
      type: ToolbarType.Radio,
    },
    {
      columnId: 'Metadata',
      title: 'Metadata',
      type: ToolbarType.KeyValue,
    },
  ];

  const actions = [
    <Button
      key="create-cron"
      onClick={() => setTriggerWorkflow(true)}
      className="h-8 border"
    >
      Create Cron Job
    </Button>,
    <Button
      key="refresh"
      className="h-8 px-2 lg:px-3"
      size="sm"
      onClick={() => {
        refetch();
      }}
      variant={'outline'}
      aria-label="Refresh crons list"
    >
      <ArrowPathIcon className={`h-4 w-4`} />
    </Button>,
  ];

  return (
    <>
      {showDeleteCron && (
        <DeleteCron
          cron={showDeleteCron}
          setShowCronRevoke={setShowDeleteCron}
          onSuccess={handleConfirmDelete}
        />
      )}
      <TriggerWorkflowForm
        defaultTimingOption="cron"
        defaultWorkflow={undefined}
        show={triggerWorkflow}
        onClose={() => setTriggerWorkflow(false)}
      />

      <DataTable
        error={queryError}
        isLoading={queryIsLoading}
        columns={columns({
          tenantId,
          onDeleteClick: handleDeleteClick,
          selectedJobId,
          setSelectedJobId,
        })}
        data={data?.rows || []}
        filters={filters}
        showColumnToggle={true}
        columnVisibility={columnVisibility}
        setColumnVisibility={setColumnVisibility}
        sorting={sorting}
        setSorting={setSorting}
        columnFilters={columnFilters}
        setColumnFilters={setColumnFilters}
        pagination={pagination}
        setPagination={setPagination}
        onSetPageSize={setPageSize}
        pageCount={data?.pagination?.num_pages || 0}
        rowSelection={rowSelection}
        setRowSelection={setRowSelection}
        actions={actions}
        getRowId={(row) => row.metadata.id}
      />
    </>
  );
}
