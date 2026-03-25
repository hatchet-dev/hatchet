import { DataTablePagination } from './data-table-pagination';
import {
  DataTableToolbar,
  ShowTableActionsProps,
  ToolbarFilters,
} from './data-table-toolbar';
import { Skeleton } from '@/components/v1/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/v1/ui/table';
import { cn } from '@/lib/utils';
import { ConfirmActionModal } from '@/pages/main/v1/task-runs-v1/actions';
import { flattenDAGsKey } from '@/pages/main/v1/workflow-runs-v1/components/v1/task-runs-columns';
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
import * as React from 'react';

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
  filters?: ToolbarFilters;
  leftActions?: JSX.Element[];
  rightActions?: JSX.Element[];
  sorting?: SortingState;
  setSorting?: OnChangeFn<SortingState>;
  setSearch?: (search: string) => void;
  search?: string;
  columnFilters?: ColumnFiltersState;
  setColumnFilters?: OnChangeFn<ColumnFiltersState>;
  pagination: PaginationState;
  setPagination: OnChangeFn<PaginationState>;
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
  hiddenFilters?: string[];
  onResetFilters?: () => void;
}

type RefetchProps = {
  isRefetching: boolean;
  onRefetch: () => void;
};

interface ExtraDataTableProps {
  emptyState?: JSX.Element;
  columnKeyToName?: Record<string, string>;
  refetchProps?: RefetchProps;
  tableActions?: ShowTableActionsProps;
}

export function DataTable<TData extends IDGetter<TData>, TValue>({
  columns,
  error,
  data,
  filters = [],
  leftActions = [],
  rightActions = [],
  sorting,
  setSorting,
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
  manualSorting = true,
  manualFiltering = true,
  getSubRows,
  hiddenFilters = [],
  onResetFilters,
  columnKeyToName,
  refetchProps,
  tableActions,
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

  const hasRows = table.getRowModel().rows?.length > 0;

  if (error) {
    return (
      <div className="p-4 text-center text-red-500">
        {error.message || 'An error occurred.'}
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      <div className="shrink-0 h-10 flex flex-col size-full items-center pt-2 mb-2">
        <div className="w-full">
          {tableActions?.selectedActionType && (
            <ConfirmActionModal
              actionType={tableActions.selectedActionType}
              params={tableActions.actionModalParams}
              table={table}
              columnKeyToName={columnKeyToName}
              filters={filters}
              hiddenFilters={[flattenDAGsKey]}
              showColumnVisibility={false}
            />
          )}
          {(leftActions || rightActions || filters.length > 0) && (
            <DataTableToolbar
              table={table}
              filters={filters}
              isLoading={isLoading}
              leftActions={leftActions}
              rightActions={rightActions}
              showColumnToggle={showColumnToggle}
              hiddenFilters={hiddenFilters}
              columnKeyToName={columnKeyToName}
              refetchProps={refetchProps}
              tableActions={tableActions}
              onResetFilters={onResetFilters}
            />
          )}
        </div>
      </div>
      <div className="min-h-0 flex-1 overflow-auto relative">
        <Table className="table-auto w-full relative z-10">
          <TableHeader className="sticky top-0 z-10 bg-background">
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => {
                  return (
                    <TableHead
                      key={header.id}
                      colSpan={header.colSpan}
                      className="border-b bg-background"
                    >
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
          <TableBody className="w-full">
            {!hasRows ? (
              <TableRow className="hover:bg-transparent">
                <TableCell colSpan={columns.length} className="h-full">
                  <div className="flex h-full w-full flex-col items-center justify-center pt-8">
                    {emptyState || (
                      <p className="text-lg font-semibold">No results.</p>
                    )}
                  </div>
                </TableCell>
              </TableRow>
            ) : (
              table.getRowModel().rows.map((row) => (
                <React.Fragment key={row.id}>
                  {getTableRow(row)}
                  {row.getIsExpanded() &&
                    row.subRows.map((r) => getTableRow(r))}
                </React.Fragment>
              ))
            )}
          </TableBody>
        </Table>
      </div>
      <div className="shrink-0 h-10 flex items-center pt-2">
        <div className="w-full">
          <DataTablePagination
            table={table}
            onSetPageSize={onSetPageSize}
            showSelectedRows={showSelectedRows}
          />
        </div>
      </div>
    </div>
  );
}
