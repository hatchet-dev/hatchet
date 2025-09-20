import {
  columns,
  keyKey,
  RateLimitColumn,
} from './components/rate-limit-columns';
import { useState } from 'react';
import { VisibilityState } from '@tanstack/react-table';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { useRateLimits } from './hooks/use-rate-limits';
import { ToolbarType } from '@/components/v1/molecules/data-table/data-table-toolbar';

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
      isRefetching={isRefetching}
      onRefetch={refetch}
    />
  );
}
