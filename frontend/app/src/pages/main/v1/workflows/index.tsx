import { columns, WorkflowColumn } from './components/workflow-columns';
import { useWorkflows } from './hooks/use-workflows';
import { DataTable } from '@/components/v1/molecules/data-table/data-table.tsx';
import { EmptyState } from '@/components/v1/molecules/empty-state/empty-state';
import { WorkflowsGuard } from '@/components/v1/molecules/empty-state/workflows-guard';
import {
  SearchBarWithFilters,
  type SearchSuggestion,
} from '@/components/v1/molecules/search-bar-with-filters/search-bar-with-filters';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { useLocalStorageState } from '@/hooks/use-local-storage-state';
import { docsPages } from '@/lib/generated/docs';
import { appRoutes } from '@/router';
import { useParams } from '@tanstack/react-router';
import { VisibilityState } from '@tanstack/react-table';
import { useMemo } from 'react';

const noopAutocomplete = () => ({ suggestions: [] as SearchSuggestion[] });
const noopApplySuggestion = (query: string) => query;

export default function WorkflowsPage() {
  return (
    <WorkflowsGuard
      title="No workflows found"
      description="Workflows define sequences of tasks that execute together. Create your first workflow to start orchestrating work."
      docs={{
        href: docsPages.v1.quickstart.href,
        description: 'Learn about workflows and tasks',
      }}
    >
      <WorkflowTable />
    </WorkflowsGuard>
  );
}

function WorkflowTable() {
  const { tenant: tenantId } = useParams({ from: appRoutes.tenantRoute.to });

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
    search,
    setSearch,
  } = useWorkflows({
    key: 'workflows-table',
  });

  const autocompleteContext = useMemo(() => ({}), []);

  if (isLoading) {
    return <Loading />;
  }

  const searchBar = (
    <SearchBarWithFilters
      value={search}
      onChange={setSearch}
      onSubmit={setSearch}
      getAutocomplete={noopAutocomplete}
      applySuggestion={noopApplySuggestion}
      autocompleteContext={autocompleteContext}
      placeholder="Search workflows by name..."
    />
  );

  return (
    <DataTable
      columns={columns(tenantId)}
      data={workflows}
      emptyState={
        <EmptyState
          filterHint="Try changing your search or filters."
          title="No workflows found"
          description="Workflows define sequences of tasks that execute together."
          docPage={docsPages.v1.quickstart}
          docLabel="Learn about workflows"
        />
      }
      searchBar={searchBar}
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
