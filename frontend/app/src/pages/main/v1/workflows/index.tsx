import { useState } from 'react';
import { DataTable } from '@/components/v1/molecules/data-table/data-table.tsx';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { SortingState, VisibilityState } from '@tanstack/react-table';
import { IntroDocsEmptyState } from '@/pages/onboarding/intro-docs-empty-state';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { columns, WorkflowColumn } from './components/workflow-columns';
import { useWorkflows } from './hooks/use-workflows';
import { DocsButton } from '@/components/v1/docs/docs-button';
import { docsPages } from '@/lib/generated/docs';

export default function WorkflowTable() {
  const { tenantId } = useCurrentTenantId();

  const [sorting, setSorting] = useState<SortingState>([
    {
      id: 'createdAt',
      desc: true,
    },
  ]);
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});

  const {
    workflows,
    numWorkflows,
    isLoading,
    isRefetching,
    pagination,
    setPagination,
    setPageSize,
    refetch,
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
        <div className="w-full h-full flex flex-col gap-y-4 text-foreground py-8 justify-center items-center">
          <p className="text-lg font-semibold">No workflows found</p>
          <div className="w-fit">
            <DocsButton
              doc={docsPages.home['your-first-task']}
              size="full"
              variant="outline"
              label="Learn about creating workflows and tasks"
            />
          </div>
        </div>
      }
      columnVisibility={columnVisibility}
      setColumnVisibility={setColumnVisibility}
      pagination={pagination}
      setPagination={setPagination}
      onSetPageSize={setPageSize}
      showSelectedRows={false}
      pageCount={numWorkflows}
      sorting={sorting}
      setSorting={setSorting}
      isLoading={isLoading}
      manualSorting={false}
      manualFiltering={false}
      showColumnToggle={true}
      columnKeyToName={WorkflowColumn}
      refetchProps={{
        isRefetching,
        onRefetch: refetch,
      }}
    />
  );
}
