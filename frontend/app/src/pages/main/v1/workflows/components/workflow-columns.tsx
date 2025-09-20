import { ColumnDef } from '@tanstack/react-table';
import { Workflow } from '@/lib/api';
import { Link } from 'react-router-dom';
import { ChevronRightIcon } from '@radix-ui/react-icons';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Badge } from '@/components/v1/ui/badge';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';

export const WorkflowColumn = {
  status: 'Status',
  name: 'Name',
  createdAt: 'Created at',
} as const;

export type WorkflowColumnKeys = keyof typeof WorkflowColumn;

export const statusKey: WorkflowColumnKeys = 'status';
export const nameKey: WorkflowColumnKeys = 'name';
export const createdAtKey: WorkflowColumnKeys = 'createdAt';

export const columns: (tenantId: string) => ColumnDef<Workflow>[] = (
  tenantId,
) => [
  {
    accessorKey: statusKey,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={WorkflowColumn.status} />
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
    accessorKey: nameKey,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={WorkflowColumn.name} />
    ),
    cell: ({ row }) => (
      <Link to={`/tenants/${tenantId}/workflows/${row.original.metadata.id}`}>
        <div className="cursor-pointer hover:underline min-w-fit whitespace-nowrap text-md p-2">
          {row.original.name}
        </div>
      </Link>
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: createdAtKey,
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={WorkflowColumn.createdAt}
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
    enableSorting: false,
    enableHiding: true,
  },
  {
    header: () => <></>,
    accessorKey: 'chevron',
    cell: ({ row }) => {
      return (
        <div className="flex gap-2 justify-end">
          <Link
            to={`/tenants/${tenantId}/workflows/${row.original.metadata.id}`}
          >
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
