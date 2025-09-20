import {
  FilterColumn,
  filterColumns,
  isDeclarativeKey,
  scopeKey,
  workflowIdKey,
} from './components/filter-columns';
import { FilterCreateButton } from './components/filter-create-form';
import { useState } from 'react';
import { VisibilityState } from '@tanstack/react-table';
import { ToolbarType } from '@/components/v1/molecules/data-table/data-table-toolbar';
import { Button } from '@/components/v1/ui/button';
import { ArrowPathIcon } from '@heroicons/react/24/outline';
import { RefetchIntervalDropdown } from '@/components/refetch-interval-dropdown';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { useFilters } from './hooks/use-filters';
import { V1Filter } from '@/lib/api';
import { useSidePanel } from '@/hooks/use-side-panel';
import { DocsButton } from '@/components/v1/docs/docs-button';
import { docsPages } from '@/lib/generated/docs';

export default function Filters() {
  const [rotate, setRotate] = useState(false);
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
    <Button
      key="refresh"
      className="h-8 px-2 lg:px-3"
      size="sm"
      onClick={() => {
        refetch();
        setRotate(!rotate);
      }}
      variant={'outline'}
      aria-label="Refresh filters list"
    >
      <ArrowPathIcon
        className={`h-4 w-4 transition-transform ${rotate ? 'rotate-180' : ''}`}
      />
    </Button>,
    <RefetchIntervalDropdown
      key="refetch-interval"
      isRefetching={isRefetching}
      onRefetch={refetch}
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
        <div className="flex flex-col items-center justify-center p-8 gap-3 text-gray-400">
          <p className="text-base font-medium">No filters found</p>
          <DocsButton
            doc={docsPages.home['run-on-event']}
            scrollTo="event-filtering"
            size="full"
            variant="outline"
            label="Learn about event filters"
          />
        </div>
      }
      columnKeyToName={FilterColumn}
      showSelectedRows={false}
    />
  );
}
