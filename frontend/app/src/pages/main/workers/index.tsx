import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { queries } from '@/lib/api';
import { DataTable } from '@/components/molecules/data-table/data-table';
import { Loading } from '@/components/ui/loading';
import { VisibilityState } from '@tanstack/react-table';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { columns, statusKey, WorkerColumn } from './components/worker-columns';
import { ToolbarType } from '@/components/molecules/data-table/data-table-toolbar';
import { DocsButton } from '@/components/docs/docs-button';
import { docsPages } from '@/lib/generated/docs';
import { useZodColumnFilters } from '@/hooks/use-zod-column-filters';
import { z } from 'zod';
import { usePagination } from '@/hooks/use-pagination';

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
        <div className="w-full h-full flex flex-col gap-y-4 text-foreground py-8 justify-center items-center">
          <p className="text-lg font-semibold">No workers found</p>
          <div className="w-fit">
            <DocsButton
              doc={docsPages.home.workers}
              size="full"
              variant="outline"
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
