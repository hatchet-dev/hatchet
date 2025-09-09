import {
  columns,
  FilterColumn,
  scopeKey,
  workflowIdKey,
} from './components/filter-columns';
import { useMemo, useState } from 'react';
import {
  ColumnFiltersState,
  RowSelectionState,
  VisibilityState,
} from '@tanstack/react-table';
import { useQuery } from '@tanstack/react-query';
import { queries } from '@/lib/api';
import {
  FilterOption,
  ToolbarType,
} from '@/components/v1/molecules/data-table/data-table-toolbar';
import { Button } from '@/components/v1/ui/button';
import { ArrowPathIcon } from '@heroicons/react/24/outline';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { useFilters } from './hooks/use-filters';

export default function Filters() {
  const { tenantId } = useCurrentTenantId();
  const [rotate, setRotate] = useState(false);

  const {
    pagination,
    setPagination,
    setPageSize,
    refetch,
    filters,
    error,
    isLoading,
    columnFilters,
    setColumnFilters,
  } = useFilters({
    key: 'filters',
  });

  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({
    EventId: false,
    Payload: false,
    scope: false,
  });

  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});


  const {
    data: workflowKeys,
    isLoading: workflowKeysIsLoading,
    error: workflowKeysError,
  } = useQuery({
    ...queries.workflows.list(tenantId, { limit: 200 }),
  });

  const workflowKeyFilters = useMemo((): FilterOption[] => {
    return (
      workflowKeys?.rows?.map((key) => ({
        value: key.metadata.id,
        label: key.name,
      })) || []
    );
  }, [workflowKeys]);

  const workflowIdToName = useMemo(
    () =>
      workflowKeyFilters.reduce(
        (acc, curr) => {
          acc[curr.value] = curr.label;
          return acc;
        },
        {} as Record<string, string>,
      ),
    [workflowKeyFilters],
  );

  const tableColumns = columns(workflowIdToName);

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

  return (
    <DataTable
      error={error || workflowKeysError}
      isLoading={isLoading || workflowKeysIsLoading}
      columns={tableColumns}
      data={filters}
      filters={[
        {
          columnId: workflowIdKey,
          title: FilterColumn.workflowId,
          options: workflowKeyFilters,
        },
        {
          columnId: scopeKey,
          title: FilterColumn.scope,
          type: ToolbarType.Array,
        },
      ]}
      showColumnToggle={true}
      columnVisibility={columnVisibility}
      setColumnVisibility={setColumnVisibility}
      actions={actions}
      columnFilters={columnFilters}
      setColumnFilters={setColumnFilters}
      pagination={pagination}
      setPagination={setPagination}
      onSetPageSize={setPageSize}
      pageCount={filters.length}
      rowSelection={rowSelection}
      setRowSelection={setRowSelection}
      getRowId={(row) => row.metadata.id}
    />
  );
}
