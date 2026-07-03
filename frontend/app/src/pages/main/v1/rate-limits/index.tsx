import {
  columns,
  keyKey,
  RateLimitColumn,
} from './components/rate-limit-columns';
import { useRateLimits } from './hooks/use-rate-limits';
import { RateLimitWithMetadata } from './hooks/use-rate-limits';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { ToolbarType } from '@/components/v1/molecules/data-table/data-table-toolbar';
import { EmptyState } from '@/components/v1/molecules/empty-state/empty-state';
import { WorkflowsGuard } from '@/components/v1/molecules/empty-state/workflows-guard';
import { useLocalStorageState } from '@/hooks/use-local-storage-state';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import api from '@/lib/api';
import { docsPages } from '@/lib/generated/docs';
import { useApiError } from '@/lib/hooks';
import { useMutation } from '@tanstack/react-query';
import { VisibilityState } from '@tanstack/react-table';
import { useMemo } from 'react';

export default function RateLimits() {
  return (
    <WorkflowsGuard
      title="No rate limits found"
      description="Rate limits cap how many times a task can run within a time window to prevent resource exhaustion."
      docs={{
        href: docsPages.v1['rate-limits'].href,
        description: 'Learn about rate limits',
      }}
    >
      <RateLimitsTable />
    </WorkflowsGuard>
  );
}

function RateLimitsTable() {
  const [columnVisibility, setColumnVisibility] =
    useLocalStorageState<VisibilityState>('hatchet:columns:rate-limits', {});
  const { tenantId } = useCurrentTenantId();
  const { handleApiError } = useApiError({});

  const {
    data,
    error,
    isLoading,
    columnFilters,
    setColumnFilters,
    pagination,
    setPageSize,
    setPagination,
    numPages,
    isRefetching,
    refetch,
    resetFilters,
  } = useRateLimits({ key: 'rate-limits-table' });

  const deleteMutation = useMutation({
    mutationKey: ['rate-limit:delete', tenantId],
    mutationFn: async (row: RateLimitWithMetadata) => {
      await api.rateLimitDelete(tenantId, { key: row.key });
    },
    onSuccess: () => refetch(),
    onError: handleApiError,
  });

  const tableColumns = useMemo(
    () => columns({ onDeleteClick: (row) => deleteMutation.mutate(row) }),
    [deleteMutation],
  );

  return (
    <DataTable
      error={error}
      isLoading={isLoading}
      columns={tableColumns}
      data={data}
      filters={[
        {
          columnId: keyKey,
          title: RateLimitColumn.key,
          type: ToolbarType.Search,
        },
      ]}
      showColumnToggle={true}
      columnVisibility={columnVisibility}
      setColumnVisibility={setColumnVisibility}
      columnFilters={columnFilters}
      setColumnFilters={setColumnFilters}
      pagination={pagination}
      setPagination={setPagination}
      onSetPageSize={setPageSize}
      pageCount={numPages}
      getRowId={(row) => row.metadata.id}
      columnKeyToName={RateLimitColumn}
      refetchProps={{
        isRefetching,
        onRefetch: refetch,
      }}
      onResetFilters={resetFilters}
      emptyState={
        <EmptyState
          title="No rate limits found"
          description="Rate limits cap how many times a task can run within a time window to prevent resource exhaustion."
          docPage={docsPages.v1['rate-limits']}
          docLabel="Learn about rate limits"
        />
      }
    />
  );
}
