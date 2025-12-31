import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { ManagedWorker } from '@/lib/api/generated/cloud/data-contracts';
import { appRoutes } from '@/router';
import { ChevronRightIcon } from '@radix-ui/react-icons';
import { Link } from '@tanstack/react-router';
import { ColumnDef } from '@tanstack/react-table';

export const columns: (tenantId: string) => ColumnDef<ManagedWorker>[] = (
  tenantId,
) => [
  {
    accessorKey: 'name',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Name" />
    ),
    cell: ({ row }) => (
      <Link
        to={appRoutes.tenantWorkflowRoute.to}
        params={{
          tenant: tenantId,
          workflow: row.original.metadata.id,
        }}
      >
        <div className="text-md min-w-fit cursor-pointer whitespace-nowrap p-2 hover:underline">
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
        <div className="flex justify-end gap-2">
          <Link
            to={appRoutes.tenantWorkflowRoute.to}
            params={{
              tenant: tenantId,
              workflow: row.original.metadata.id,
            }}
          >
            <div className="text-md min-w-fit cursor-pointer whitespace-nowrap p-2 hover:underline">
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
