import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { WorkflowTriggers } from '@/components/v1/molecules/workflow-triggers/workflow-triggers';
import { Badge } from '@/components/v1/ui/badge';
import { Workflow, WorkflowTriggerType } from '@/lib/api';
import { appRoutes } from '@/router';
import { ChevronRightIcon } from '@radix-ui/react-icons';
import { Link } from '@tanstack/react-router';
import { ColumnDef } from '@tanstack/react-table';

export const WorkflowColumn = {
  status: 'Status',
  name: 'Name',
  triggers: 'Triggers',
  lastDeployed: 'Last deployed',
  createdAt: 'Created at',
} as const;

type WorkflowColumnKeys = keyof typeof WorkflowColumn;

export const statusKey: WorkflowColumnKeys = 'status';
export const nameKey: WorkflowColumnKeys = 'name';
export const triggersKey: WorkflowColumnKeys = 'triggers';
const lastDeployedKey: WorkflowColumnKeys = 'lastDeployed';
const createdAtKey: WorkflowColumnKeys = 'createdAt';

export const workflowStatusFilters = [
  { value: 'ACTIVE', label: 'Active' },
  { value: 'PAUSED', label: 'Paused' },
];

export const workflowTriggerTypeFilters = [
  { value: WorkflowTriggerType.EVENT, label: 'Event' },
  { value: WorkflowTriggerType.CRON, label: 'Cron' },
  { value: WorkflowTriggerType.NONE, label: 'None' },
];

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
        <div className="min-w-fit cursor-pointer p-2">
          <div className="text-md whitespace-nowrap hover:underline">
            {row.original.name}
          </div>
          {row.original.description && (
            <div
              className="max-w-md truncate text-xs text-muted-foreground"
              title={row.original.description}
            >
              {row.original.description}
            </div>
          )}
        </div>
      </Link>
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: triggersKey,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={WorkflowColumn.triggers} />
    ),
    cell: ({ row }) => <WorkflowTriggers workflow={row.original} />,
    enableSorting: false,
    enableHiding: true,
  },
  {
    accessorKey: lastDeployedKey,
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={WorkflowColumn.lastDeployed}
        className="whitespace-nowrap"
      />
    ),
    cell: ({ row }) => {
      const latestVersion = row.original.versions?.[0];

      if (!latestVersion) {
        return <span className="text-muted-foreground">—</span>;
      }

      return (
        <div className="whitespace-nowrap">
          <RelativeDate date={latestVersion.metadata.createdAt} />
          {latestVersion.version && (
            <div className="font-mono text-xs text-muted-foreground">
              {latestVersion.version}
            </div>
          )}
        </div>
      );
    },
    enableSorting: false,
    enableHiding: true,
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
