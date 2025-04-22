'use client';

import { useState, useMemo, useCallback, useEffect } from 'react';
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
import { useNavigate } from 'react-router-dom';
import { ROUTES } from '@/next/lib/routes';
import { V1WorkflowType } from '@/lib/api';

const styles = {
  status: 'p-0 w-[40px]',
  runId: 'border-r border-border',
};

interface DataTableProps<TData, TValue> {
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
}

const getTableRow = <TData,>(
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
      className="group cursor-pointer"
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

export function DataTable<TData, TValue>({
  columns,
  data,
  emptyState,
  isLoading,
  selectedTaskId,
  onRowClick = () => {},
  onSelectionChange,
  rowSelection = {},
  setRowSelection,
  selectAll = false,
  getSubRows,
}: DataTableProps<TData, TValue>) {
  const navigate = useNavigate();
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});

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

      onRowClick(row.original);
    },
    [onRowClick],
  );

  const handleDoubleClick = useCallback(
    (row: Row<TData>) => {
      // TODO: Fix type
      const task = row.original as any;
      if (task.type !== V1WorkflowType.TASK) {
        navigate(ROUTES.runs.detail(task.taskExternalId || ''));
      }
    },
    [navigate],
  );

  if (isLoading) {
    return (
      <TableRow>
        <TableCell colSpan={columns.length} className="h-24 text-center">
          Loading...
        </TableCell>
      </TableRow>
    );
  }
  if (!table.getRowModel().rows?.length) {
    return (
      <TableRow>
        <TableCell colSpan={columns.length} className="h-24 text-center">
          {emptyState || 'No results found.'}
        </TableCell>
      </TableRow>
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
          {table.getRowModel().rows.map((row) => {
            const isSelected = row.getIsSelected();
            const isTaskSelected =
              selectedTaskId === (row.original as any).taskExternalId;

            return (
              <>
                {getTableRow(
                  row,
                  isSelected,
                  isTaskSelected,
                  (e: React.MouseEvent) => handleClick(row, e, isSelected),
                  () => handleDoubleClick(row),
                )}
                {row.getIsExpanded() &&
                  row.subRows.map((r) =>
                    getTableRow(
                      r,
                      isSelected,
                      isTaskSelected,
                      (e: React.MouseEvent) => handleClick(r, e, isSelected),
                      () => handleDoubleClick(r),
                    ),
                  )}
              </>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
