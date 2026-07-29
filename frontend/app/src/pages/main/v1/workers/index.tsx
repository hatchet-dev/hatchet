import {
  columns,
  labelsKey,
  statusKey,
  WorkerColumn,
} from './components/worker-columns';
import { ToolbarType } from '@/components/v1/molecules/data-table/data-table-toolbar';
import { DataTable } from '@/components/v1/molecules/data-table/data-table.tsx';
import { EmptyState } from '@/components/v1/molecules/empty-state/empty-state';
import { WorkflowsGuard } from '@/components/v1/molecules/empty-state/workflows-guard';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { useLocalStorageState } from '@/hooks/use-local-storage-state';
import { usePagination } from '@/hooks/use-pagination';
import { useZodColumnFilters } from '@/hooks/use-zod-column-filters';
import { queries } from '@/lib/api';
import { WorkerStatus } from '@/lib/api/generated/data-contracts';
import { docsPages } from '@/lib/generated/docs';
import { appRoutes } from '@/router';
import { useQuery } from '@tanstack/react-query';
import { useParams } from '@tanstack/react-router';
import { VisibilityState } from '@tanstack/react-table';
import { useMemo, useState, useCallback } from 'react';
import { z } from 'zod';

const workersQuerySchema = z
  .object({
    s: z.array(z.enum(['ACTIVE', 'INACTIVE', 'PAUSED'])).optional(), // status
    l: z.array(z.string()).optional(), // labels
  })
  .default({})
  .transform((data) => ({
    s: data.s ?? ['ACTIVE', 'PAUSED'],
    l: data.l ?? [],
  }));

export default function Workers() {
  return (
    <WorkflowsGuard
      title="No workers found"
      description="Deploy workers on Kubernetes, Porter, Railway, Render, ECS, or any container platform. They automatically connect to Hatchet and can scale up or down based on workload."
      docs={{
        href: docsPages.v1.workers.href,
        description: 'Learn about workers',
      }}
    >
      <WorkersTable />
    </WorkflowsGuard>
  );
}

function WorkersTable() {
  const { tenant: tenantId } = useParams({ from: appRoutes.tenantRoute.to });
  const { refetchInterval } = useRefetchInterval();

  const paramKey = 'workers-table';
  const [openLabelsPopover, setOpenLabelsPopover] = useState<string | null>(
    null,
  );

  const {
    state: { s: statuses, l: labels },
    columnFilters,
    setColumnFilters,
    resetFilters,
  } = useZodColumnFilters(workersQuerySchema, paramKey, {
    s: statusKey,
    l: labelsKey,
  });

  const { pagination, setPagination, limit, offset, setPageSize } =
    usePagination({
      key: paramKey,
      resetPageOnChange: [statuses, labels],
    });

  const [columnVisibility, setColumnVisibility] =
    useLocalStorageState<VisibilityState>('hatchet:columns:workers', {});

  const handleSetOpenLabelsPopover = useCallback(
    (id: string | null) => setOpenLabelsPopover(id),
    [],
  );

  const tableColumns = useMemo(
    () => columns(tenantId, openLabelsPopover, handleSetOpenLabelsPopover),
    [tenantId, openLabelsPopover, handleSetOpenLabelsPopover],
  );

  const listWorkersQuery = useQuery({
    ...queries.workers.list(tenantId, {
      offset,
      limit,
      statuses: statuses as WorkerStatus[],
      labels: labels.length > 0 ? labels : undefined,
    }),
    refetchInterval,
  });

  const rows = listWorkersQuery.data?.rows ?? [];
  const pageCount =
    listWorkersQuery.data?.pagination?.num_pages ??
    Math.ceil(rows.length / limit);

  if (listWorkersQuery.isLoading) {
    return <Loading />;
  }

  return (
    <DataTable
      columns={tableColumns}
      data={rows}
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
        {
          columnId: labelsKey,
          title: 'Labels',
          type: ToolbarType.KeyValue,
        },
      ]}
      emptyState={
        <EmptyState
          filterHint="Try changing your filters."
          title="No workers found"
          description="Workers are persistent processes that pull and execute your tasks. Connect a worker to start running workflows."
          docPage={docsPages.v1.workers}
          docLabel="Learn about workers"
        />
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
      pageCount={pageCount}
    />
  );
}
