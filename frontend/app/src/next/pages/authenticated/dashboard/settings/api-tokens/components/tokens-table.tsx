import { useState } from 'react';
import {
  flexRender,
  getCoreRowModel,
  useReactTable,
  ColumnDef,
  SortingState,
  Column,
  Table,
} from '@tanstack/react-table';
import { APIToken } from '@/lib/api';
import {
  Table as UITable,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/next/components/ui/table';
import { Button } from '@/next/components/ui/button';
import useApiTokens, { useApiTokensContext } from '@/next/hooks/use-api-tokens';
import { Skeleton } from '@/next/components/ui/skeleton';
import { apiTokens } from '@/next/lib/can/features/api-tokens.permissions';
import useCan from '@/next/hooks/use-can';
import {
  ArrowDownIcon,
  ArrowUpIcon,
  ChevronsUpDownIcon,
  EyeOffIcon,
  SlidersHorizontalIcon,
} from 'lucide-react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/next/components/ui/dropdown-menu';
import { cn } from '@/next/lib/utils';
import { RevokeTokenForm } from '@/next/pages/authenticated/dashboard/settings/api-tokens/components/revoke-token-form';
import {
  Pagination,
  PageSizeSelector,
  PageSelector,
} from '@/next/components/ui/pagination';

// Create a DataTableColumnHeader component
interface DataTableColumnHeaderProps<TData>
  extends React.HTMLAttributes<HTMLDivElement> {
  column: Column<TData, unknown>;
  title: string;
}

function DataTableColumnHeader<TData>({
  column,
  title,
  className,
}: DataTableColumnHeaderProps<TData>) {
  if (!column.getCanSort()) {
    return <div className={cn(className)}>{title}</div>;
  }

  return (
    <div className={cn('flex items-center justify-start', className)}>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="sm"
            className="h-8 px-2 data-[state=open]:bg-accent"
          >
            <span className="font-medium">{title}</span>
            <div className="ml-2">
              {column.getIsSorted() === 'desc' ? (
                <ArrowDownIcon className="h-4 w-4" />
              ) : column.getIsSorted() === 'asc' ? (
                <ArrowUpIcon className="h-4 w-4" />
              ) : (
                <ChevronsUpDownIcon className="h-4 w-4" />
              )}
            </div>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start">
          <DropdownMenuItem onClick={() => column.toggleSorting(false)}>
            <ArrowUpIcon className="mr-2 h-3.5 w-3.5 text-muted-foreground/70" />
            Asc
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => column.toggleSorting(true)}>
            <ArrowDownIcon className="mr-2 h-3.5 w-3.5 text-muted-foreground/70" />
            Desc
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={() => column.toggleVisibility(false)}>
            <EyeOffIcon className="mr-2 h-3.5 w-3.5 text-muted-foreground/70" />
            Hide
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

// Create a DataTableViewOptions component
interface DataTableViewOptionsProps<TData> {
  table: Table<TData>;
}

function DataTableViewOptions<TData>({
  table,
}: DataTableViewOptionsProps<TData>) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm" className="h-8 w-8 p-0">
          <SlidersHorizontalIcon className="h-4 w-4" />
          <span className="sr-only">View options</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-[150px]">
        <DropdownMenuItem
          onClick={() => table.resetColumnVisibility()}
          className="justify-between"
        >
          Reset
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        {table
          .getAllColumns()
          .filter((column) => column.getCanHide())
          .map((column) => {
            return (
              <DropdownMenuItem
                key={column.id}
                className="capitalize"
                onClick={() => column.toggleVisibility(!column.getIsVisible())}
              >
                <div className="mr-1">{column.getIsVisible() ? '✓' : ''}</div>
                {column.id}
              </DropdownMenuItem>
            );
          })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

interface TokensTableProps {
  emptyState?: React.ReactNode;
}

export function TokensTable({ emptyState }: TokensTableProps) {
  const { data, isLoading } = useApiTokens();
  const [revokeToken, setRevokeToken] = useState<APIToken | null>(null);

  const { filters, setFilters } = useApiTokensContext();
  const [sorting, setSorting] = useState<SortingState>([]);
  const [pagination, setPagination] = useState({
    pageIndex: 0,
    pageSize: 10,
  });
  const { can } = useCan();

  // Define columns
  const columns: ColumnDef<APIToken>[] = [
    {
      accessorKey: 'name',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Name" />
      ),
      cell: ({ row }) => (
        <div className="font-medium">{row.getValue('name')}</div>
      ),
      enableSorting: true,
      enableHiding: false,
    },
    {
      accessorKey: 'createdAt',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Created" />
      ),
      cell: ({ row }) => {
        const createdAt = new Date(row.original.metadata.createdAt);
        return <div>{createdAt.toLocaleDateString()}</div>;
      },
      enableSorting: true,
      enableHiding: true,
    },
    {
      accessorKey: 'expiresAt',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Expires" />
      ),
      cell: ({ row }) => {
        const expiresAt = new Date(row.original.expiresAt);
        return <div>{expiresAt.toLocaleDateString()}</div>;
      },
      enableSorting: true,
      enableHiding: true,
    },
    {
      id: 'actions',
      header: () => <div className="sr-only">Actions</div>,
      cell: ({ row }) =>
        can(apiTokens.manage()) && (
          <div className="text-right">
            <Button
              variant="ghost"
              onClick={() => setRevokeToken(row.original)}
              className="text-red-500 hover:text-red-700"
            >
              Revoke
            </Button>
          </div>
        ),
      enableSorting: false,
      enableHiding: false,
    },
  ];

  // Create table instance
  const table = useReactTable<APIToken>({
    data: data ?? [],
    columns,
    getCoreRowModel: getCoreRowModel(),
    state: {
      sorting,
      pagination,
    },
    enableSorting: true,
    enableMultiSort: false,
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
    onPaginationChange: setPagination,
    manualSorting: true, // Keep as true since we're handling sorting through API
    manualPagination: true,
    pageCount: Math.ceil((data?.length ?? 0) / pagination.pageSize),
  });

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="text-sm text-muted-foreground">
          {data?.length} token(s) found
        </div>
        <DataTableViewOptions table={table} />
      </div>
      <div className="rounded-md border">
        <UITable>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <TableHead key={header.id}>
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
                <TableRow
                  key={row.id}
                  data-state={row.getIsSelected() && 'selected'}
                >
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
                  {emptyState || 'No API tokens found.'}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </UITable>
      </div>

      <Pagination className="p-2 justify-between flex flex-row">
        <PageSizeSelector />
        <PageSelector variant="dropdown" />
      </Pagination>

      {revokeToken && (
        <RevokeTokenForm
          apiToken={revokeToken}
          close={() => setRevokeToken(null)}
        />
      )}
    </div>
  );
}
