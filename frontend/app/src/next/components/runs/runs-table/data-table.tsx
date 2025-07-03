'use client';

import React, { useState, useMemo, useCallback, useEffect } from 'react';
import {
  ColumnDef,
  ColumnFiltersState,
  VisibilityState,
  RowSelectionState,
  OnChangeFn,
  flexRender,
  getCoreRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getSortedRowModel,
  useReactTable,
  Row,
  ExpandedState,
} from '@tanstack/react-table';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/next/components/ui/table';
import { cn } from '@/next/lib/utils';

const styles = {
  status: 'p-0 w-[40px]',
  runId: 'border-r border-border',
};

interface IDGetter {
  metadata: {
    id: string;
  };
  isExpandable?: boolean;
}

interface DataTableProps<TData extends IDGetter, TValue> {
  columns: ColumnDef<TData, TValue>[];
  data: TData[];
  emptyState?: React.ReactNode;
  isLoading?: boolean;
  selectedTaskId?: string;
  onRowClick?: (row: TData) => void;
  onSelectionChange?: (selectedRows: TData[]) => void;
  rowSelection?: RowSelectionState;
  setRowSelection?: OnChangeFn<RowSelectionState>;
  selectAll?: boolean;
  getSubRows?: (originalRow: TData, index: number) => TData[];
  onDoubleClick?: (row: TData) => void;
}

const nullCallback = () => {};
const defaultRowSelection = {};

export function DataTable<TData extends IDGetter, TValue>({
  columns,
  data,
  emptyState,
  isLoading,
  selectedTaskId,
  onRowClick = nullCallback,
  onSelectionChange,
  rowSelection = defaultRowSelection,
  setRowSelection,
  selectAll = false,
  getSubRows,
  onDoubleClick = nullCallback,
}: DataTableProps<TData, TValue>) {
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});
  const [expanded, setExpanded] = useState<ExpandedState>({});

  const memoizedRowSelection = useMemo(() => {
    if (selectAll) {
      return data.reduce((acc, _, index) => ({ ...acc, [index]: true }), {});
    }
    return rowSelection;
  }, [selectAll, data, rowSelection]);

  const table = useReactTable({
    data,
    columns,
    state: {
      columnFilters,
      columnVisibility,
      rowSelection: memoizedRowSelection,
      expanded,
    },
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    onRowSelectionChange: setRowSelection,
    enableRowSelection: true,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
    getRowId: (row) => {
      const typedRow = row as { taskExternalId?: string; id?: string };
      return typedRow.taskExternalId || typedRow.id || String(Math.random());
    },
    getSubRows,
    getRowCanExpand: (row) => row.subRows.length > 0,
    onExpandedChange: setExpanded,
  });

  // Notify parent component of selection changes
  useEffect(() => {
    if (onSelectionChange) {
      const selectedRows = table
        .getSelectedRowModel()
        .rows.map((row) => row.original);
      onSelectionChange(selectedRows);
    }
  }, [onSelectionChange, table]);

  const getTableRow = (
    row: Row<TData>,
    isSelected: boolean,
    isTaskSelected: boolean,
    handleClick: (e: React.MouseEvent) => void,
    handleDoubleClick: () => void,
  ) => {
    return (
      <TableRow
        key={row.id}
        data-state={isSelected || isTaskSelected ? 'selected' : undefined}
        className={cn(
          row.original.isExpandable && 'cursor-pointer hover:bg-muted',
          'group cursor-pointer',
        )}
        onClick={handleClick}
        onDoubleClick={handleDoubleClick}
      >
        {row.getVisibleCells().map((cell) => (
          <TableCell
            key={cell.id}
            className={cn(styles[cell.column.id as keyof typeof styles])}
          >
            {flexRender(cell.column.columnDef.cell, cell.getContext())}
          </TableCell>
        ))}
      </TableRow>
    );
  };

  const handleClick = useCallback(
    (row: Row<TData>, e: React.MouseEvent, isSelected: boolean) => {
      // Prevent row click if clicking on a button or link
      if ((e.target as HTMLElement).closest('button, a')) {
        return;
      }

      // If Cmd/Ctrl is held, toggle selection instead of triggering row click
      if (e.metaKey || e.ctrlKey) {
        row.toggleSelected(!isSelected);
        return;
      }

      // Toggle row expansion if the row can be expanded
      if (row.getCanExpand()) {
        row.toggleExpanded(true);
      }

      onRowClick(row.original);
    },
    [onRowClick],
  );

  if (isLoading) {
    return (
      <div className="rounded-md border">
        <Table>
          <TableBody>
            <TableRow>
              <TableCell colSpan={columns.length} className="h-24 text-center">
                Loading...
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </div>
    );
  }

  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          {table.getHeaderGroups().map((headerGroup) => (
            <TableRow key={headerGroup.id}>
              {headerGroup.headers.map((header) => (
                <TableHead
                  key={header.id}
                  className={cn(styles[header.id as keyof typeof styles])}
                >
                  {header.isPlaceholder
                    ? null
                    : flexRender(
                        header.column.columnDef.header,
                        header.getContext(),
                      )}
                </TableHead>
              ))}
            </TableRow>
          ))}
        </TableHeader>
        <TableBody>
          {!table.getRowModel().rows?.length ? (
            <TableRow>
              <TableCell
                colSpan={table.getHeaderGroups()[0].headers.length}
                className="h-48 text-center py-8"
              >
                {emptyState || 'No results found.'}
              </TableCell>
            </TableRow>
          ) : (
            table.getRowModel().rows.map((row) => {
              const isSelected = row.getIsSelected();
              const isTaskSelected =
                selectedTaskId ===
                (
                  row.original as TData & {
                    taskExternalId?: string;
                  }
                )?.taskExternalId;

              return (
                <React.Fragment key={row.id}>
                  {getTableRow(
                    row,
                    isSelected,
                    isTaskSelected,
                    (e: React.MouseEvent) => handleClick(row, e, isSelected),
                    () => onDoubleClick(row.original),
                  )}
                  {row.getIsExpanded() &&
                    row.subRows.map((r) =>
                      getTableRow(
                        r,
                        isSelected,
                        isTaskSelected,
                        (e: React.MouseEvent) => handleClick(r, e, isSelected),
                        () => onDoubleClick(r.original),
                      ),
                    )}
                </React.Fragment>
              );
            })
          )}
        </TableBody>
      </Table>
    </div>
  );
}
