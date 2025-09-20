import { useState } from 'react';
import { DataTable } from '@/components/v1/molecules/data-table/data-table.tsx';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { SortingState, VisibilityState } from '@tanstack/react-table';
import { IntroDocsEmptyState } from '@/pages/onboarding/intro-docs-empty-state';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { columns, WorkflowColumn } from './components/workflow-columns';
import { useWorkflows } from './hooks/use-workflows';

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
    pagination,
    setPagination,
    setPageSize,
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
      filters={[]}
      emptyState={
        <IntroDocsEmptyState
          link="/home/your-first-task"
          title="No Registered Workflows"
          linkPreambleText="To learn more about workflows function in Hatchet,"
          linkText="check out our documentation."
        />
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
    />
  );
}
