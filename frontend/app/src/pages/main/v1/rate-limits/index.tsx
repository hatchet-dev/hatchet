import {
  columns,
  keyKey,
  RateLimitColumn,
} from './components/rate-limit-columns';
import { useRateLimits } from './hooks/use-rate-limits';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { ToolbarType } from '@/components/v1/molecules/data-table/data-table-toolbar';
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
    />
  );
}
