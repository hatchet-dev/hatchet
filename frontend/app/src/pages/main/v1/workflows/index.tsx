import { columns, WorkflowColumn } from './components/workflow-columns';
import { useWorkflows } from './hooks/use-workflows';
import { EmptyState } from '@/components/v1/molecules/empty-state/empty-state';
import { DataTable } from '@/components/v1/molecules/data-table/data-table.tsx';
import {
  SearchBarWithFilters,
  type SearchSuggestion,
} from '@/components/v1/molecules/search-bar-with-filters/search-bar-with-filters';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { usePylon } from '@/components/support-chat';
import { useLocalStorageState } from '@/hooks/use-local-storage-state';
import { queries } from '@/lib/api';
import { docsPages } from '@/lib/generated/docs';
import { appRoutes } from '@/router';
import { useQuery } from '@tanstack/react-query';
import { useParams } from '@tanstack/react-router';
import { VisibilityState } from '@tanstack/react-table';
import { BookOpen, Calendar, MessageCircle, Rocket } from 'lucide-react';
import { useMemo } from 'react';

const noopAutocomplete = () => ({ suggestions: [] as SearchSuggestion[] });
const noopApplySuggestion = (query: string) => query;

export default function WorkflowTable() {
  const { tenant: tenantId } = useParams({ from: appRoutes.tenantRoute.to });
  const pylon = usePylon();

  const workflowCountQuery = useQuery(
    queries.workflows.list(tenantId, { limit: 1, offset: 0 }),
  );

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

  if (isLoading || !workflowCountQuery.isSuccess) {
    return <Loading />;
  }

  const hasWorkflows = (workflowCountQuery.data?.rows?.length ?? 0) > 0;

  if (!hasWorkflows) {
    return (
      <div className="flex h-full items-center justify-center">
        <EmptyState
          title="No workflows found"
          description="Workflows define sequences of tasks that execute together. Create your first workflow to start orchestrating work."
          actions={[
            {
              icon: <Rocket className="size-4" />,
              label: 'Get started',
              description: 'Follow our onboarding guide',
              href: `/tenants/${tenantId}/overview`,
            },
            {
              icon: <BookOpen className="size-4" />,
              label: 'Read the docs',
              description: 'Learn about workflows and tasks',
              href: docsPages.v1.quickstart.href,
              external: true,
            },
            ...(pylon.enabled
              ? [
                  {
                    icon: <MessageCircle className="size-4" />,
                    label: 'Talk to us',
                    description: 'Chat with our support team',
                    onClick: pylon.show,
                  } as const,
                ]
              : [
                  {
                    icon: <MessageCircle className="size-4" />,
                    label: 'Join Discord',
                    description: 'Chat with the Hatchet community',
                    href: 'https://discord.com/invite/ZMeUafwH89',
                    external: true,
                  } as const,
                ]),
            {
              icon: <Calendar className="size-4" />,
              label: 'Book office hours',
              description: 'Schedule time with the Hatchet team',
              href: 'https://hatchet.run/office-hours',
              external: true,
            },
          ]}
        />
      </div>
    );
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
