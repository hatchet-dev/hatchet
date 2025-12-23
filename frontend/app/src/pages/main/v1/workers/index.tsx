import { columns, statusKey, WorkerColumn } from './components/worker-columns';
import { DocsButton } from '@/components/v1/docs/docs-button';
import { ToolbarType } from '@/components/v1/molecules/data-table/data-table-toolbar';
import { DataTable } from '@/components/v1/molecules/data-table/data-table.tsx';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { usePagination } from '@/hooks/use-pagination';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { useZodColumnFilters } from '@/hooks/use-zod-column-filters';
import { queries } from '@/lib/api';
import { docsPages } from '@/lib/generated/docs';
import { useQuery } from '@tanstack/react-query';
import { VisibilityState } from '@tanstack/react-table';
import { useMemo, useState } from 'react';
import { z } from 'zod';

const workersQuerySchema = z
  .object({
    s: z.array(z.enum(['ACTIVE', 'INACTIVE', 'PAUSED'])).optional(), // status
  })
  .default({})
  .transform((data) => ({
    s: data.s ?? ['ACTIVE', 'PAUSED'],
  }));

export default function Workers() {
  const { tenantId } = useCurrentTenantId();
  const { refetchInterval } = useRefetchInterval();
  const paramKey = 'workers-table';
  const { pagination, setPagination, limit, offset, setPageSize } =
    usePagination({
      key: paramKey,
    });

  const {
    state: { s: statuses },
    columnFilters,
    setColumnFilters,
    resetFilters,
  } = useZodColumnFilters(workersQuerySchema, paramKey, { s: statusKey });

  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});

  const listWorkersQuery = useQuery({
    ...queries.workers.list(tenantId),
    refetchInterval,
  });

  const data = useMemo(
    () =>
      listWorkersQuery.data?.rows
        ?.filter((w) => w.status && statuses.includes(w.status))
        ?.sort(
          (a, b) =>
            new Date(b.metadata?.createdAt).getTime() -
            new Date(a.metadata?.createdAt).getTime(),
        ) ?? [],
    [listWorkersQuery.data?.rows, statuses],
  );

  const paginatedData = useMemo(
    () => data.slice(offset, offset + limit),
    [data, limit, offset],
  );

  if (listWorkersQuery.isLoading) {
    return <Loading />;
  }

  return (
    <DataTable
      columns={columns(tenantId)}
      data={paginatedData}
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
      emptyState={
        <div className="flex h-full w-full flex-col items-center justify-center gap-y-4 py-8 text-foreground">
          <p className="text-lg font-semibold">No workers found</p>
          <div className="w-fit">
            <DocsButton
              doc={docsPages.home.workers}
              label="Learn about running workers"
            />
          </div>
        </div>
      }
      columnFilters={columnFilters}
      setColumnFilters={setColumnFilters}
      columnVisibility={columnVisibility}
      setColumnVisibility={setColumnVisibility}
      showColumnToggle={true}
      columnKeyToName={WorkerColumn}
      refetchProps={{
        isRefetching: listWorkersQuery.isRefetching,
        onRefetch: listWorkersQuery.refetch,
      }}
      onResetFilters={resetFilters}
      pagination={pagination}
      setPagination={setPagination}
      onSetPageSize={setPageSize}
      pageCount={Math.ceil(data.length / limit)}
    />
  );
}
