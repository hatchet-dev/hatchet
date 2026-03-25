import { SdkInfo } from './sdk-info';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Badge, BadgeProps } from '@/components/v1/ui/badge';
import { Worker } from '@/lib/api';
import { cn } from '@/lib/utils';
import { appRoutes } from '@/router';
import { Link } from '@tanstack/react-router';
import { ColumnDef } from '@tanstack/react-table';

export const WorkerColumn = {
  status: 'Status',
  name: 'Name',
  type: 'Type',
  startedAt: 'Started at',
  slots: 'Slots',
  lastHeartbeatAt: 'Last seen',
  runtime: 'SDK Version',
} as const;

type WorkerColumnKeys = keyof typeof WorkerColumn;

export const statusKey: WorkerColumnKeys = 'status';
const nameKey: WorkerColumnKeys = 'name';
const startedAtKey: WorkerColumnKeys = 'startedAt';
const slotsKey: WorkerColumnKeys = 'slots';
const lastHeartbeatAtKey: WorkerColumnKeys = 'lastHeartbeatAt';
const runtimeKey: WorkerColumnKeys = 'runtime';

interface WorkerStatusBadgeProps extends BadgeProps {
  status?: string;
  count?: number;
  animated?: boolean;
  isLoading?: boolean;
}

type StatusConfig = {
  colors: string;
  primary: string;
  primaryOKLCH: string;
  label: string;
};

const WorkerStatusConfigs: Record<string, StatusConfig> = {
  ACTIVE: {
    colors:
      'text-green-800 dark:text-green-300 bg-green-500/20 ring-green-500/30',
    primary: 'text-green-500 bg-green-500',
    primaryOKLCH: 'oklch(0.723 0.219 149.579)',
    label: 'Active',
  },
  INACTIVE: {
    colors: 'text-red-800 dark:text-red-300 bg-red-500/20 ring-red-500',
    primary: 'text-red-500 bg-red-500',
    primaryOKLCH: 'oklch(0.637 0.237 25.331)',
    label: 'Inactive',
  },
  PAUSED: {
    colors:
      'text-yellow-800 dark:text-yellow-300 bg-yellow-500/20 ring-yellow-500/30',
    primary: 'text-yellow-500 bg-yellow-500',
    primaryOKLCH: 'oklch(0.795 0.184 86.047)',
    label: 'Paused',
  },
};

function WorkerStatusBadge({
  status,
  count,
  variant,
  animated,
  isLoading,
  className,
  ...props
}: WorkerStatusBadgeProps) {
  const config = !status
    ? {
        colors:
          'text-gray-800 dark:text-gray-300 bg-gray-500/20 ring-gray-500/30',
        primary: 'text-gray-500 bg-gray-500',
        primaryOKLCH: 'oklch(0.551 0.027 264.364)',
        label: 'Unknown',
      }
    : WorkerStatusConfigs[status] || {
        colors:
          'text-gray-800 dark:text-gray-300 bg-gray-500/20 ring-gray-500/30',
        primary: 'text-gray-500 bg-gray-500',
        primaryOKLCH: 'oklch(0.551 0.027 264.364)',
        label: status,
      };

  const isDisabled = count === 0;
  const finalConfig = isDisabled
    ? {
        colors: 'text-red-800 dark:text-red-300 bg-red-500/20 ring-red-500',
        primary: 'text-red-500 bg-red-500',
      }
    : config;

  const content = (
    <>
      {count !== undefined && `${count} `}
      {config.label}
    </>
  );

  return (
    <Badge
      className={cn(
        'px-3 py-1',
        finalConfig.colors,
        'rounded-md border-transparent text-xs font-medium',
        className,
      )}
      variant={variant}
      {...props}
    >
      {content}
    </Badge>
  );
}

export const columns: (tenantId: string) => ColumnDef<Worker>[] = (
  tenantId,
) => [
  {
    accessorKey: nameKey,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={WorkerColumn.name} />
    ),
    cell: ({ row }) => (
      <Link
        to={appRoutes.tenantWorkerRoute.to}
        params={{ tenant: tenantId, worker: row.original.metadata.id }}
      >
        <div className="min-w-fit cursor-pointer whitespace-nowrap hover:underline">
          {row.original.webhookUrl || row.original.name}
        </div>
      </Link>
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: statusKey,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={WorkerColumn.status} />
    ),
    cell: ({ row }) => (
      <Link
        to={appRoutes.tenantWorkerRoute.to}
        params={{ tenant: tenantId, worker: row.original.metadata.id }}
      >
        <WorkerStatusBadge status={row.original.status} />
      </Link>
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: startedAtKey,
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={WorkerColumn.startedAt}
        className="whitespace-nowrap"
      />
    ),
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
    accessorKey: slotsKey,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={WorkerColumn.slots} />
    ),
    cell: ({ row }) => {
      const slotConfig = row.original.slotConfig || {};
      const entries = Object.entries(slotConfig).sort(([a], [b]) =>
        a.localeCompare(b),
      );

      if (entries.length === 0) {
        return <div className="text-xs text-muted-foreground">No slots</div>;
      }

      return (
        <div className="space-y-1">
          {entries.map(([slotType, capacity]) => {
            const available = capacity?.available;
            const limit = capacity?.limit;
            const label =
              available !== undefined ? `${available} / ${limit}` : `${limit}`;

            return (
              <div key={slotType} className="text-xs text-muted-foreground">
                <span className="font-medium text-foreground">{slotType}</span>:{' '}
                {label}
              </div>
            );
          })}
        </div>
      );
    },
    enableSorting: false,
    enableHiding: true,
  },
  {
    accessorKey: lastHeartbeatAtKey,
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={WorkerColumn.lastHeartbeatAt}
        className="whitespace-nowrap"
      />
    ),
    cell: ({ row }) => {
      return (
        <div className="whitespace-nowrap">
          {row.original.lastHeartbeatAt && (
            <RelativeDate date={row.original.lastHeartbeatAt} />
          )}
        </div>
      );
    },
    enableSorting: false,
    enableHiding: true,
  },
  {
    accessorKey: runtimeKey,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={WorkerColumn.runtime} />
    ),
    cell: ({ row }) => <SdkInfo runtimeInfo={row.original.runtimeInfo} />,
    enableSorting: false,
    enableHiding: true,
  },
];
