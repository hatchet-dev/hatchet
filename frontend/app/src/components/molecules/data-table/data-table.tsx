import * as React from "react";
import {
  ColumnDef,
  ColumnFiltersState,
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
} from "@tanstack/react-table";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

import { DataTablePagination } from "./data-table-pagination";
import { DataTableToolbar, ToolbarFilters } from "./data-table-toolbar";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

export interface IDGetter {
  metadata: {
    id: string;
  };
  getRow?: () => JSX.Element;
  onClick?: () => void;
  isExpandable?: boolean;
}

interface DataTableProps<TData extends IDGetter, TValue> {
  columns: ColumnDef<TData, TValue>[];
  data: TData[];
  filters: ToolbarFilters;
  actions?: JSX.Element[];
  sorting?: SortingState;
  setSorting?: OnChangeFn<SortingState>;
  columnFilters?: ColumnFiltersState;
  setColumnFilters?: OnChangeFn<ColumnFiltersState>;
  pagination?: PaginationState;
  setPagination?: OnChangeFn<PaginationState>;
  pageCount?: number;
  onSetPageSize?: (pageSize: number) => void;
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
        parent?: Row<TData> | undefined
      ) => string)
    | undefined;
}

export function DataTable<TData extends IDGetter, TValue>({
  columns,
  data,
  filters,
  actions = [],
  sorting,
  setSorting,
  columnFilters,
  setColumnFilters,
  pagination,
  setPagination,
  pageCount,
  onSetPageSize,
  columnVisibility,
  setColumnVisibility,
  rowSelection,
  setRowSelection,
  isLoading,
  getRowId,
}: DataTableProps<TData, TValue>) {
  const tableData = React.useMemo(
    () => (isLoading ? Array(10).fill({}) : data),
    [isLoading, data]
  );

  const tableColumns = React.useMemo(
    () =>
      isLoading
        ? columns.map((column) => ({
            ...column,
            cell: () => <Skeleton className="h-4 w-[100px]" />,
          }))
        : columns,
    [isLoading, columns]
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
    manualSorting: true,
    manualFiltering: true,
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
        data-state={row.getIsSelected() && "selected"}
        className={cn(
          row.original.isExpandable && "cursor-pointer hover:bg-muted"
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

  return (
    <div className="space-y-4">
      {filters && filters.length > 0 && (
        <DataTableToolbar table={table} filters={filters} actions={actions} />
      )}
      <div className="rounded-md border">
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
                            header.getContext()
                          )}
                    </TableHead>
                  );
                })}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {table.getRowModel().rows?.length ? (
              table.getRowModel().rows.map((row) => getTableRow(row))
            ) : (
              <TableRow>
                <TableCell
                  colSpan={columns.length}
                  className="h-24 text-center"
                >
                  No results.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
      {pagination && (
        <DataTablePagination table={table} onSetPageSize={onSetPageSize} />
      )}
    </div>
  );
}
