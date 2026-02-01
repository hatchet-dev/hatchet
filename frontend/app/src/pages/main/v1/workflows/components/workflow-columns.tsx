import { ActivityMiniChart } from '@/components/v1/molecules/charts/activity-mini-chart';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Badge } from '@/components/v1/ui/badge';
import { Workflow } from '@/lib/api';
import { appRoutes } from '@/router';
import { ChevronRightIcon } from '@radix-ui/react-icons';
import { Link } from '@tanstack/react-router';
import { ColumnDef } from '@tanstack/react-table';

export const WorkflowColumn = {
  status: 'Status',
  name: 'Name',
  activity: 'Activity',
  createdAt: 'Created at',
} as const;

type WorkflowColumnKeys = keyof typeof WorkflowColumn;

const statusKey: WorkflowColumnKeys = 'status';
export const nameKey: WorkflowColumnKeys = 'name';
const activityKey: WorkflowColumnKeys = 'activity';
const createdAtKey: WorkflowColumnKeys = 'createdAt';

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
      <Link
        to={appRoutes.tenantWorkflowRoute.to}
        params={{ tenant: tenantId, workflow: row.original.metadata.id }}
      >
        <div className="text-md min-w-fit cursor-pointer whitespace-nowrap p-2 hover:underline">
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
    accessorKey: activityKey,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={WorkflowColumn.activity} />
    ),
    cell: ({ row }) => (
      <div className="flex flex-row items-center">
        <ActivityMiniChart workflowId={row.original.metadata.id} />
      </div>
    ),
    enableSorting: false,
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
            params={{ tenant: tenantId, workflow: row.original.metadata.id }}
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
