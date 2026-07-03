import {
  FilterColumn,
  filterColumns,
  isDeclarativeKey,
  scopeKey,
  workflowIdKey,
} from './components/filter-columns';
import { FilterCreateButton } from './components/filter-create-form';
import { useFilters } from './hooks/use-filters';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { ToolbarType } from '@/components/v1/molecules/data-table/data-table-toolbar';
import { EmptyState } from '@/components/v1/molecules/empty-state/empty-state';
import { WorkflowsGuard } from '@/components/v1/molecules/empty-state/workflows-guard';
import { useLocalStorageState } from '@/hooks/use-local-storage-state';
import { useSidePanel } from '@/hooks/use-side-panel';
import { V1Filter } from '@/lib/api';
import { docsPages } from '@/lib/generated/docs';
import { VisibilityState } from '@tanstack/react-table';

export default function FiltersPage() {
  return (
    <WorkflowsGuard
      title="No filters found"
      description="Event filters route incoming events to specific workflows based on payload conditions."
      docs={{
        href: `${docsPages.v1.events.href}#event-filters`,
        description: 'Learn about event filters',
      }}
    >
      <Filters />
    </WorkflowsGuard>
  );
}

function Filters() {
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

  const [columnVisibility, setColumnVisibility] =
    useLocalStorageState<VisibilityState>('hatchet:columns:filters', {
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
        <EmptyState
          title="No filters found"
          description="Event filters route incoming events to specific workflows based on payload conditions."
          docPage={{ href: `${docsPages.v1.events.href}#event-filters` }}
          docLabel="Learn about event filters"
        />
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
