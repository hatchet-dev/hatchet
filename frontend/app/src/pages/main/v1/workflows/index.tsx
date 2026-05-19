import {
  columns,
  nameKey,
  WorkflowColumn,
} from './components/workflow-columns';
import { useWorkflows } from './hooks/use-workflows';
import { DocsButton } from '@/components/v1/docs/docs-button';
import { ToolbarType } from '@/components/v1/molecules/data-table/data-table-toolbar';
import { DataTable } from '@/components/v1/molecules/data-table/data-table.tsx';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { useLocalStorageState } from '@/hooks/use-local-storage-state';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { docsPages } from '@/lib/generated/docs';
import { VisibilityState } from '@tanstack/react-table';

export default function WorkflowTable() {
  const { tenantId } = useCurrentTenantId();

  const [columnVisibility, setColumnVisibility] =
    useLocalStorageState<VisibilityState>('hatchet:columns:workflows', {});

  const {
    workflows,
    numWorkflows,
    isLoading,
    isRefetching,
    pagination,
    setPagination,
    setPageSize,
    refetch,
    columnFilters,
    setColumnFilters,
    resetFilters,
  } = useWorkflows({
    key: 'workflows-table',
  });

  if (isLoading) {
    return <Loading />;
  }

  return (
    <DataTable
      columns={columns(tenantId)}
      data={workflows}
      emptyState={
        <div className="flex h-full w-full flex-col items-center justify-center gap-y-4 py-8 text-foreground">
          <p className="text-lg font-semibold">No workflows found</p>
          <div className="w-fit">
            <DocsButton
              doc={docsPages.v1.quickstart}
              label="Learn about creating workflows and tasks"
            />
          </div>
        </div>
      }
      filters={[
        {
          columnId: nameKey,
          title: WorkflowColumn.name,
          type: ToolbarType.Search,
        },
      ]}
      columnVisibility={columnVisibility}
      setColumnVisibility={setColumnVisibility}
      pagination={pagination}
      setPagination={setPagination}
      onSetPageSize={setPageSize}
      showSelectedRows={false}
      pageCount={numWorkflows}
      isLoading={isLoading}
      showColumnToggle={true}
      columnKeyToName={WorkflowColumn}
      refetchProps={{
        isRefetching,
        onRefetch: refetch,
      }}
      columnFilters={columnFilters}
      setColumnFilters={setColumnFilters}
      onResetFilters={resetFilters}
    />
  );
}
