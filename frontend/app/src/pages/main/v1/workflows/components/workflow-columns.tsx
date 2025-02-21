import { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '../../../../components/molecules/data-table/data-table-column-header';
import { Workflow } from '@/lib/api';
import { Link } from 'react-router-dom';
import { ChevronRightIcon } from '@radix-ui/react-icons';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Badge } from '@/components/v1/ui/badge';

export const columns: ColumnDef<Workflow>[] = [
  {
    accessorKey: 'Status',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Status" />
    ),
    cell: ({ row }) => (
      <>
        {row.original.isPaused ? (
          <Badge variant="inProgress">Paused</Badge>
        ) : (
          <Badge variant="successful">Active</Badge>
        )}
      </>
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: 'name',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Name" />
    ),
    cell: ({ row }) => (
      <Link to={`/workflows/${row.original.metadata.id}`}>
        <div className="cursor-pointer hover:underline min-w-fit whitespace-nowrap text-md p-2">
          {row.original.name}
        </div>
      </Link>
    ),
    enableSorting: true,
    enableHiding: false,
  },
  {
    accessorKey: 'createdAt',
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title="Created at"
        className="whitespace-nowrap"
      />
    ),
    sortingFn: (a, b) => {
      return (
        new Date(a.original.metadata.createdAt).getTime() -
        new Date(b.original.metadata.createdAt).getTime()
      );
    },
    cell: ({ row }) => {
      return (
        <div className="whitespace-nowrap">
          <RelativeDate date={row.original.metadata.createdAt} />
        </div>
      );
    },
    enableSorting: true,
    enableHiding: true,
  },
  {
    header: () => <></>,
    accessorKey: 'chevron',
    cell: ({ row }) => {
      return (
        <div className="flex gap-2 justify-end">
          <Link to={`/workflows/${row.original.metadata.id}`}>
            <div className="cursor-pointer hover:underline min-w-fit whitespace-nowrap text-md p-2">
              <ChevronRightIcon
                className="h-5 w-5 flex-none text-gray-700 dark:text-gray-300"
                aria-hidden="true"
              />
            </div>
          </Link>
        </div>
      );
    },
    enableSorting: false,
    enableHiding: false,
  },
];
