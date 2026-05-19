import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '../../ui/table';
import React from 'react';

type SimpleTableColumn<T> = {
  columnLabel: string;
  cellRenderer: (row: T) => React.ReactNode;
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
            {columns.map(({ columnLabel }) => (
              <TableHead key={columnLabel} className="pr-8">
                {columnLabel}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.map((row, i) => (
            <TableRow key={rowKey(row, i)}>
              {columns.map(({ columnLabel, cellRenderer }) => (
                <TableCell key={columnLabel} className="pr-8">
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
