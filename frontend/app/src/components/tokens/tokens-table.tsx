import { useState } from 'react';
import {
  flexRender,
  getCoreRowModel,
  useReactTable,
  ColumnDef,
  SortingState,
} from '@tanstack/react-table';
import { APIToken } from '@/lib/api';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { useApiTokensContext } from '@/hooks/use-api-tokens';
import { Skeleton } from '@/components/ui/skeleton';

interface TokensTableProps {
  data: APIToken[];
  isLoading: boolean;
  onRevokeClick: (token: APIToken) => void;
}

export function TokensTable({
  data,
  isLoading,
  onRevokeClick,
}: TokensTableProps) {
  const { filters, setFilters } = useApiTokensContext();
  const [sorting, setSorting] = useState<SortingState>([]);

  // Define columns
  const columns: ColumnDef<APIToken>[] = [
    {
      accessorKey: 'name',
      header: 'Name',
      cell: ({ row }) => <div>{row.getValue('name')}</div>,
    },
    {
      accessorKey: 'metadata.createdAt',
      header: 'Created',
      cell: ({ row }) => {
        const createdAt = new Date(row.original.metadata.createdAt);
        return <div>{createdAt.toLocaleDateString()}</div>;
      },
    },
    {
      accessorKey: 'expiresAt',
      header: 'Expires',
      cell: ({ row }) => {
        const expiresAt = new Date(row.original.expiresAt);
        return <div>{expiresAt.toLocaleDateString()}</div>;
      },
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => (
        <Button
          variant="ghost"
          onClick={() => onRevokeClick(row.original)}
          className="text-red-500 hover:text-red-700"
        >
          Revoke
        </Button>
      ),
    },
  ];

  // Create table instance
  const table = useReactTable<APIToken>({
    data: isLoading ? [] : data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    state: {
      sorting,
    },
    onSortingChange: (updaterOrValue) => {
      const newSorting =
        typeof updaterOrValue === 'function'
          ? updaterOrValue(sorting)
          : updaterOrValue;

      setSorting(newSorting);

      // Update context filters for sorting
      if (newSorting.length > 0) {
        const { id, desc } = newSorting[0];
        setFilters({
          ...filters,
          sortBy: id,
          sortDirection: desc ? 'desc' : 'asc',
        });
      } else {
        setFilters({
          ...filters,
          sortBy: undefined,
          sortDirection: undefined,
        });
      }
    },
    manualSorting: true,
  });

  // Handle search
  const handleSearch = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFilters({
      ...filters,
      search: e.target.value,
    });
  };

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <Input
          placeholder="Search tokens..."
          className="max-w-sm"
          value={filters.search || ''}
          onChange={handleSearch}
        />
      </div>
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <TableHead
                    key={header.id}
                    className="cursor-pointer"
                    onClick={header.column.getToggleSortingHandler()}
                  >
                    <div className="flex items-center">
                      {flexRender(
                        header.column.columnDef.header,
                        header.getContext(),
                      )}
                      {header.column.getIsSorted() === 'asc' && ' 🔼'}
                      {header.column.getIsSorted() === 'desc' && ' 🔽'}
                    </div>
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {isLoading ? (
              // Loading state
              Array(5)
                .fill(0)
                .map((_, i) => (
                  <TableRow key={i}>
                    {columns.map((_, j) => (
                      <TableCell key={j}>
                        <Skeleton className="h-4 w-24" />
                      </TableCell>
                    ))}
                  </TableRow>
                ))
            ) : table.getRowModel().rows.length > 0 ? (
              // Data rows
              table.getRowModel().rows.map((row) => (
                <TableRow key={row.id}>
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>
                      {flexRender(
                        cell.column.columnDef.cell,
                        cell.getContext(),
                      )}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              // Empty state
              <TableRow>
                <TableCell
                  colSpan={columns.length}
                  className="h-24 text-center"
                >
                  No API tokens found.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
