import {
  FilterColumn,
  filterColumns,
  isDeclarativeKey,
  scopeKey,
  workflowIdKey,
} from './components/filter-columns';
import { FilterCreateButton } from './components/filter-create-form';
import { useFilters } from './hooks/use-filters';
import { DocsButton } from '@/components/v1/docs/docs-button';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { ToolbarType } from '@/components/v1/molecules/data-table/data-table-toolbar';
import { useSidePanel } from '@/hooks/use-side-panel';
import { V1Filter } from '@/lib/api';
import { docsPages } from '@/lib/generated/docs';
import { VisibilityState } from '@tanstack/react-table';
import { useState } from 'react';

export default function Filters() {
  const sidePanel = useSidePanel();

  const {
    pagination,
    setPagination,
    setPageSize,
    refetch,
    filters,
    numFilters,
    error,
    isLoading,
    isRefetching,
    columnFilters,
    setColumnFilters,
    workflowIdToName,
    workflowNameFilters,
    mutations,
  } = useFilters({
    key: 'table',
  });

  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({
    [isDeclarativeKey]: false,
  });

  const handleRowClick = (filter: V1Filter) => {
    sidePanel.open({
      type: 'filter-detail',
      content: {
        filter,
      },
    });
  };

  const tableColumns = filterColumns(workflowIdToName, handleRowClick);

  const actions = [
    <FilterCreateButton
      key="create"
      workflowNameFilters={workflowNameFilters}
      onCreate={mutations.create.perform}
      isCreating={mutations.create.isPending}
    />,
  ];

  return (
    <DataTable
      error={error}
      isLoading={isLoading}
      columns={tableColumns}
      data={filters}
      filters={[
        {
          columnId: workflowIdKey,
          title: FilterColumn.workflowId,
          options: workflowNameFilters,
          type: ToolbarType.Checkbox,
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
      rightActions={actions}
      columnFilters={columnFilters}
      setColumnFilters={setColumnFilters}
      pagination={pagination}
      setPagination={setPagination}
      onSetPageSize={setPageSize}
      pageCount={numFilters}
      getRowId={(row) => row.metadata.id}
      emptyState={
        <div className="flex h-full w-full flex-col items-center justify-center gap-y-4 py-8 text-foreground">
          <p className="text-lg font-semibold">No filters found</p>
          <div className="w-fit">
            <DocsButton
              doc={docsPages.home['run-on-event']}
              scrollTo="event-filtering"
              label="Learn about event filters"
            />
          </div>
        </div>
      }
      columnKeyToName={FilterColumn}
      showSelectedRows={false}
      refetchProps={{
        isRefetching,
        onRefetch: refetch,
      }}
    />
  );
}
