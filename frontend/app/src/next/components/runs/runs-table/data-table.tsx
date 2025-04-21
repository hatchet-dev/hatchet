'use client';

import { useState, useMemo } from 'react';
import {
  ColumnDef,
  ColumnFiltersState,
  VisibilityState,
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
}

export function DataTable<TData, TValue>({
  columns,
  data,
  emptyState,
  isLoading,
  selectedTaskId,
  rowClicked,
}: DataTableProps<TData, TValue>) {
  const navigate = useNavigate();
  // Client-side state
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});

  // Set up table
  const table = useReactTable({
    data,
    columns,
    state: {
      columnFilters,
      columnVisibility,
    },
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
  });

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
      const isSelected =
        selectedTaskId === (row.original as any).taskExternalId;

      const handleClick = () => {
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
          data-state={isSelected ? 'selected' : undefined}
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
    table.getRowModel().rows,
    isLoading,
    emptyState,
    selectedTaskId,
    columns.length,
    rowClicked,
    navigate,
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
