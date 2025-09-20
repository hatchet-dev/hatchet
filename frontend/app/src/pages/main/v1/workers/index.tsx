import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { queries } from '@/lib/api';
import { DataTable } from '@/components/v1/molecules/data-table/data-table.tsx';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { ColumnFiltersState, VisibilityState } from '@tanstack/react-table';
import { IntroDocsEmptyState } from '@/pages/onboarding/intro-docs-empty-state';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { columns, WorkerColumn } from './components/worker-columns';
import { ToolbarType } from '@/components/v1/molecules/data-table/data-table-toolbar';
import { RefetchIntervalDropdown } from '@/components/refetch-interval-dropdown';

export default function Workers() {
  const { tenantId } = useCurrentTenantId();
  const { currentInterval } = useRefetchInterval();

  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([
    {
      id: 'status',
      value: ['ACTIVE', 'PAUSED'],
    },
  ]);

  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});

  const listWorkersQuery = useQuery({
    ...queries.workers.list(tenantId),
    refetchInterval: currentInterval,
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

  const actions = [
    <RefetchIntervalDropdown
      key="refetch-interval"
      isRefetching={listWorkersQuery.isRefetching}
      onRefetch={listWorkersQuery.refetch}
    />,
  ];

  return (
    <DataTable
      columns={columns(tenantId)}
      data={data}
      pageCount={1}
      rightActions={actions}
      filters={[
        {
          columnId: 'status',
          title: 'Status',
          type: ToolbarType.Checkbox,
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
      columnVisibility={columnVisibility}
      setColumnVisibility={setColumnVisibility}
      showColumnToggle={true}
      columnKeyToName={WorkerColumn}
    />
  );
}
