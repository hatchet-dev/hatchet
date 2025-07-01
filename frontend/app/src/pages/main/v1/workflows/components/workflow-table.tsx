import { useEffect, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { queries } from '@/lib/api';
import { useSearchParams } from 'react-router-dom';
import { DataTable } from '@/components/v1/molecules/data-table/data-table.tsx';
import { columns } from './workflow-columns';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { Button } from '@/components/v1/ui/button';
import { ArrowPathIcon } from '@heroicons/react/24/outline';
import {
  PaginationState,
  SortingState,
  VisibilityState,
} from '@tanstack/react-table';
import { IntroDocsEmptyState } from '@/pages/onboarding/intro-docs-empty-state';
import { useCurrentTenantId } from '@/hooks/use-tenant';

export function WorkflowTable() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { tenantId } = useCurrentTenantId();

  const [sorting, setSorting] = useState<SortingState>([
    {
      id: 'lastRun',
      desc: true,
    },
  ]);
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});
  const [rotate, setRotate] = useState(false);

  const [pagination, setPagination] = useState<PaginationState>(() => {
    const pageIndex = Number(searchParams.get('pageIndex')) || 0;
    const pageSize = Number(searchParams.get('pageSize')) || 50;

    return { pageIndex, pageSize };
  });
  const [pageSize, setPageSize] = useState<number>(
    Number(searchParams.get('pageSize')) || 50,
  );

  const [pageIndex, setPageIndex] = useState<number>(
    Number(searchParams.get('pageIndex')) || 0,
  );

  const listWorkflowQuery = useQuery({
    ...queries.workflows.list(tenantId, {
      limit: pagination.pageSize,
      offset: pagination.pageIndex * pageSize,
    }),
    refetchInterval: 5000,
  });

  useEffect(() => {
    const newSearchParams = new URLSearchParams(searchParams);
    newSearchParams.set('pageIndex', pagination.pageIndex.toString());
    newSearchParams.set('pageSize', pagination.pageSize.toString());
    setSearchParams(newSearchParams);
  }, [pagination, setSearchParams, searchParams]);

  const data = useMemo(() => {
    const data = listWorkflowQuery.data?.rows || [];
    setPageIndex(listWorkflowQuery.data?.pagination?.num_pages || 0);

    return data;
  }, [
    listWorkflowQuery.data?.rows,
    listWorkflowQuery.data?.pagination?.num_pages,
  ]);

  if (listWorkflowQuery.isLoading) {
    return <Loading />;
  }

  return (
    <div className="h-full">
      <DataTable
        columns={columns(tenantId)}
        data={data}
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
        pageCount={pageIndex}
        sorting={sorting}
        setSorting={setSorting}
        isLoading={listWorkflowQuery.isLoading}
        manualSorting={false}
        manualFiltering={false}
      />
    </div>
  );
}
