import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { queries } from '@/lib/api';
import { DataTable } from '@/components/v1/molecules/data-table/data-table.tsx';
import { columns } from './worker-columns';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { ColumnFiltersState } from '@tanstack/react-table';
import { IntroDocsEmptyState } from '@/pages/onboarding/intro-docs-empty-state';
import { useCurrentTenantId } from '@/hooks/use-tenant';

export function WorkersTable() {
  const { tenantId } = useCurrentTenantId();

  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([
    {
      id: 'status',
      value: ['ACTIVE', 'PAUSED'],
    },
  ]);

  const listWorkersQuery = useQuery({
    ...queries.workers.list(tenantId),
    refetchInterval: 3000,
  });

  const data = useMemo(() => {
    let rows = listWorkersQuery.data?.rows || [];

    columnFilters.map((filter) => {
      if (filter.id === 'status') {
        rows = rows.filter((row) =>
          (filter.value as any[]).includes(row.status),
        );
      }
    });

    return rows.sort(
      (a, b) =>
        new Date(b.metadata?.createdAt).getTime() -
        new Date(a.metadata?.createdAt).getTime(),
    );
  }, [listWorkersQuery.data?.rows, columnFilters]);

  if (listWorkersQuery.isLoading) {
    return <Loading />;
  }

  const emptyState = (
    <IntroDocsEmptyState
      link="/home/workers"
      title="No Workers Found"
      linkPreambleText="To learn more about how workers function in Hatchet,"
      linkText="check out our documentation."
    />
  );

  return (
    <DataTable
      columns={columns(tenantId)}
      data={data}
      pageCount={1}
      filters={[
        {
          columnId: 'status',
          title: 'Status',
          options: [
            { value: 'ACTIVE', label: 'Active' },
            { value: 'PAUSED', label: 'Paused' },
            { value: 'INACTIVE', label: 'Inactive' },
          ],
        },
      ]}
      emptyState={emptyState}
      columnFilters={columnFilters}
      setColumnFilters={setColumnFilters}
    />
  );
}
