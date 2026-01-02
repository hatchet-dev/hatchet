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

type SimpleTableProps<
  T extends {
    metadata: {
      id: string;
    };
  },
> = {
  columns: SimpleTableColumn<T>[];
  data: T[];
};

export function SimpleTable<
  T extends {
    metadata: {
      id: string;
    };
  },
>({ columns, data }: SimpleTableProps<T>) {
  return (
    <div className="overflow-hidden rounded-md border bg-background">
      <Table>
        <TableHeader>
          {columns.map(({ columnLabel }) => (
            <TableHead key={columnLabel}>{columnLabel}</TableHead>
          ))}
        </TableHeader>
        <TableBody>
          {data.map((row) => (
            <TableRow key={row.metadata.id}>
              {columns.map(({ columnLabel, cellRenderer }) => (
                <TableCell key={columnLabel}>{cellRenderer(row)}</TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
