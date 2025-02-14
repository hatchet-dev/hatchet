import * as React from 'react';
import {
  ColumnDef,
  ColumnFiltersState,
  ExpandedState,
  OnChangeFn,
  PaginationState,
  Row,
  RowSelectionState,
  SortingState,
  VisibilityState,
  flexRender,
  getCoreRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table';

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';

import { DataTablePagination } from './data-table-pagination';
import { DataTableToolbar, ToolbarFilters } from './data-table-toolbar';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';

export interface IDGetter<T> {
  metadata: {
    id: string;
  };
  subRows?: T[];
  getRow?: () => JSX.Element;
  onClick?: () => void;
  isExpandable?: boolean;
}

interface DataTableProps<TData extends IDGetter<TData>, TValue> {
  columns: ColumnDef<TData, TValue>[];
  data: TData[];
  error?: Error | null;
  filters: ToolbarFilters;
  actions?: JSX.Element[];
  sorting?: SortingState;
  setSorting?: OnChangeFn<SortingState>;
  setSearch?: (search: string) => void;
  search?: string;
  columnFilters?: ColumnFiltersState;
  setColumnFilters?: OnChangeFn<ColumnFiltersState>;
  pagination?: PaginationState;
  setPagination?: OnChangeFn<PaginationState>;
  showSelectedRows?: boolean;
  pageCount?: number;
  onSetPageSize?: (pageSize: number) => void;
  showColumnToggle?: boolean;
  columnVisibility?: VisibilityState;
  setColumnVisibility?: OnChangeFn<VisibilityState>;
  rowSelection?: RowSelectionState;
  setRowSelection?: OnChangeFn<RowSelectionState>;
  isLoading?: boolean;
  enableRowSelection?: boolean;
  getRowId?:
    | ((
        originalRow: TData,
        index: number,
        parent?: Row<TData> | undefined,
      ) => string)
    | undefined;
  manualSorting?: boolean;
  manualFiltering?: boolean;
  getSubRows?: (row: TData) => TData[];
}

interface ExtraDataTableProps {
  emptyState?: JSX.Element;
  card?: {
    containerStyle?: string;
    component: React.FC<any> | ((data: any) => JSX.Element);
  };
}

export function DataTable<TData extends IDGetter<TData>, TValue>({
  columns,
  error,
  data,
  filters,
  actions = [],
  sorting,
  setSorting,
  setSearch,
  search,
  columnFilters,
  setColumnFilters,
  pagination,
  setPagination,
  pageCount,
  onSetPageSize,
  showSelectedRows = true,
  showColumnToggle,
  columnVisibility,
  setColumnVisibility,
  rowSelection,
  setRowSelection,
  isLoading,
  getRowId,
  emptyState,
  card,
  manualSorting = true,
  manualFiltering = true,
  getSubRows,
}: DataTableProps<TData, TValue> & ExtraDataTableProps) {
  const [expanded, setExpanded] = React.useState<ExpandedState>({});

  const loadingNoData = isLoading && !data.length;

  const tableData = React.useMemo(
    () => (loadingNoData ? Array(10).fill({ metadata: {} }) : data),
    [loadingNoData, data],
  );

  const tableColumns = React.useMemo(
    () =>
      loadingNoData
        ? columns.map((column) => ({
            ...column,
            cell: () => <Skeleton className="h-4 w-[100px]" />,
          }))
        : columns,
    [loadingNoData, columns],
  );

  const table = useReactTable({
    data: tableData,
    columns: tableColumns,
    state: {
      sorting,
      columnVisibility,
      rowSelection: rowSelection || {},
      columnFilters,
      pagination,
      expanded,
    },
    pageCount,
    enableRowSelection: !!rowSelection,
    onRowSelectionChange: setRowSelection,
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    onPaginationChange: setPagination,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
    getSubRows: getSubRows,
    onExpandedChange: setExpanded,
    // TODO: Figure this out
    getRowCanExpand: (row) => row.subRows.length > 0,
    manualSorting,
    manualFiltering,
    manualPagination: true,
    getRowId,
  });

  const getTableRow = (row: Row<TData>) => {
    if (row.original.getRow) {
      return row.original.getRow();
    }

    return (
      <TableRow
        key={row.id}
        data-state={row.getIsSelected() && 'selected'}
        className={cn(
          row.original.isExpandable && 'cursor-pointer hover:bg-muted',
        )}
        onClick={row.original.onClick}
      >
        {row.getVisibleCells().map((cell) => (
          <TableCell key={cell.id}>
            {flexRender(cell.column.columnDef.cell, cell.getContext())}
          </TableCell>
        ))}
      </TableRow>
    );
  };

  const getTable = () => (
    <Table>
      <TableHeader>
        {table.getHeaderGroups().map((headerGroup) => (
          <TableRow key={headerGroup.id}>
            {headerGroup.headers.map((header) => {
              return (
                <TableHead key={header.id} colSpan={header.colSpan}>
                  {header.isPlaceholder
                    ? null
                    : flexRender(
                        header.column.columnDef.header,
                        header.getContext(),
                      )}
                </TableHead>
              );
            })}
          </TableRow>
        ))}
      </TableHeader>
      <TableBody>
        {error ? (
          <TableRow className="p-4 text-center text-red-500">
            <TableCell colSpan={columns.length}>
              {error.message || 'An error occurred.'}
            </TableCell>
          </TableRow>
        ) : table.getRowModel().rows?.length ? (
          table.getRowModel().rows.map((row) => (
            <React.Fragment key={row.id}>
              {getTableRow(row)}
              {row.getIsExpanded() && row.subRows.map((r) => getTableRow(r))}
            </React.Fragment>
          ))
        ) : (
          <TableRow>
            <TableCell colSpan={columns.length} className="h-24 text-center">
              {emptyState || 'No results.'}
            </TableCell>
          </TableRow>
        )}
      </TableBody>
    </Table>
  );

  const getCards = () => (
    <div
      className={
        card?.containerStyle ||
        'grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3'
      }
    >
      {error
        ? error.message || 'An error occurred.'
        : table.getRowModel().rows?.length
          ? table
              .getRowModel()
              .rows.map((row) =>
                card?.component
                  ? card?.component({ data: row.original })
                  : null,
              )
          : emptyState || 'No results.'}
    </div>
  );

  return (
    <div className="space-y-4">
      {(setSearch || actions || (filters && filters.length > 0)) && (
        <DataTableToolbar
          table={table}
          filters={filters}
          isLoading={isLoading}
          actions={actions}
          search={search}
          setSearch={setSearch}
          showColumnToggle={showColumnToggle}
        />
      )}
      <div className={`rounded-md ${!card && 'border'}`}>
        {!card ? getTable() : getCards()}
      </div>
      {pagination && (
        <DataTablePagination
          table={table}
          onSetPageSize={onSetPageSize}
          showSelectedRows={showSelectedRows}
        />
      )}
    </div>
  );
}
