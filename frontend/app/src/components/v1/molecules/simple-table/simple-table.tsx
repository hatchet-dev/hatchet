import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '../../ui/table';
import { cn } from '@/lib/utils';
import React from 'react';

type SimpleTableColumn<T> = {
  columnLabel: string;
  cellRenderer: (row: T) => React.ReactNode;
  align?: 'left' | 'right';
};

type SimpleTableProps<T> = {
  columns: SimpleTableColumn<T>[];
  data: T[];
  rowKey: (row: T, index: number) => string;
};

export function SimpleTable<T>({ columns, data, rowKey }: SimpleTableProps<T>) {
  return (
    <div className="overflow-auto rounded-md border bg-background">
      <Table>
        <TableHeader>
          <TableRow>
            {columns.map(({ columnLabel, align }) => (
              <TableHead
                key={columnLabel}
                className={cn('pr-8', align === 'right' && 'text-right')}
              >
                {columnLabel}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.map((row, i) => (
            <TableRow key={rowKey(row, i)}>
              {columns.map(({ columnLabel, cellRenderer, align }) => (
                <TableCell
                  key={columnLabel}
                  className={cn('pr-8', align === 'right' && 'text-right')}
                >
                  {cellRenderer(row)}
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
