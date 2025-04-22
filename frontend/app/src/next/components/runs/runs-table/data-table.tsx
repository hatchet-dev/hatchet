'use client';

import { useState, useMemo } from 'react';
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

interface DataTableProps<TData, TValue> {
  columns: ColumnDef<TData, TValue>[];
  data: TData[];
  emptyState?: React.ReactNode;
  isLoading?: boolean;
  selectedTaskId?: string;
  rowClicked?: (row: TData) => void;
  onSelectionChange?: (selectedRows: TData[]) => void;
  rowSelection?: RowSelectionState;
  setRowSelection?: OnChangeFn<RowSelectionState>;
  selectAll?: boolean;
}

export function DataTable<TData, TValue>({
  columns,
  data,
  emptyState,
  isLoading,
  selectedTaskId,
  rowClicked,
  onSelectionChange,
  rowSelection = {},
  setRowSelection,
  selectAll = false,
}: DataTableProps<TData, TValue>) {
  const navigate = useNavigate();
  // Client-side state
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});

  // Memoize the row selection state
  const memoizedRowSelection = useMemo(() => {
    if (selectAll) {
      return data.reduce((acc, _, index) => ({ ...acc, [index]: true }), {});
    }
    return rowSelection;
  }, [selectAll, data, rowSelection]);

  // Set up table
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
  });

  // Notify parent component of selection changes
  useMemo(() => {
    if (onSelectionChange) {
      const selectedRows = table
        .getSelectedRowModel()
        .rows.map((row) => row.original);
      onSelectionChange(selectedRows);
    }
  }, [onSelectionChange, table]);

  const styles = {
    status: 'p-0 w-[40px]',
    runId: 'border-r border-border',
  };

  const tableRows = useMemo(() => {
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

    return table.getRowModel().rows.map((row) => {
      const isSelected = row.getIsSelected();
      const isTaskSelected =
        selectedTaskId === (row.original as any).taskExternalId;

      const handleClick = (e: React.MouseEvent) => {
        // Prevent row click if clicking on a button or link
        if ((e.target as HTMLElement).closest('button, a')) {
          return;
        }

        // If Cmd/Ctrl is held, toggle selection instead of triggering row click
        if (e.metaKey || e.ctrlKey) {
          row.toggleSelected(!isSelected);
          return;
        }

        if (rowClicked) {
          rowClicked(row.original);
        }
      };

      const handleDoubleClick = () => {
        const task = row.original as any;
        if (task.type !== V1WorkflowType.TASK) {
          navigate(ROUTES.runs.detail(task.taskExternalId || ''));
        }
      };

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
    });
  }, [
    isLoading,
    table,
    columns.length,
    emptyState,
    selectedTaskId,
    rowClicked,
    navigate,
    styles,
  ]);

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
        <TableBody>{tableRows}</TableBody>
      </Table>
    </div>
  );
}
