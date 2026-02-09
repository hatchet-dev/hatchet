import {
  columns,
  keyKey,
  RateLimitColumn,
} from './components/rate-limit-columns';
import { useRateLimits } from './hooks/use-rate-limits';
import { DocsButton } from '@/components/v1/docs/docs-button';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { ToolbarType } from '@/components/v1/molecules/data-table/data-table-toolbar';
import { docsPages } from '@/lib/generated/docs';
import { VisibilityState } from '@tanstack/react-table';
import { useState } from 'react';

export default function RateLimits() {
  return <RateLimitsTable />;
}

function RateLimitsTable() {
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});

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

  return (
    <DataTable
      error={error}
      isLoading={isLoading}
      columns={columns}
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
        <div className="flex h-full w-full flex-col items-center justify-center gap-y-4 py-8 text-foreground">
          <p className="text-lg font-semibold">No rate limits found</p>
          <div className="w-fit">
            <DocsButton
              doc={docsPages.home['rate-limits']}
              label="Learn about rate limits"
            />
          </div>
        </div>
      }
    />
  );
}
