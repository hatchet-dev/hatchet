import { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import { Worker } from '@/lib/api';
import { Link } from 'react-router-dom';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { SdkInfo } from './sdk-info';

import { Badge, BadgeProps } from '@/components/ui/badge';
import { cn } from '@/lib/utils';

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

export const WorkerStatusConfigs: Record<string, StatusConfig> = {
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

export function WorkerStatusBadge({
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
        'text-xs font-medium rounded-md border-transparent',
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
    accessorKey: 'status',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Status" />
    ),
    cell: ({ row }) => (
      <Link to={`/tenants/${tenantId}/workers/${row.original.metadata.id}`}>
        <WorkerStatusBadge status={row.original.status} />
      </Link>
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
      <Link to={`/tenants/${tenantId}/workers/${row.original.metadata.id}`}>
        <div className="cursor-pointer hover:underline min-w-fit whitespace-nowrap">
          {row.original.webhookUrl || row.original.name}
        </div>
      </Link>
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: 'type',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Type" />
    ),
    cell: ({ row }) => (
      <div className="cursor-pointer hover:underline min-w-fit whitespace-nowrap">
        {row.original.type.toLocaleLowerCase()}
      </div>
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: 'startedAt',
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title="Started at"
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
    accessorKey: 'slots',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Available Slots" />
    ),
    cell: ({ row }) => (
      <div>
        {row.original.availableRuns} / {row.original.maxRuns}
      </div>
    ),
    enableSorting: false,
    enableHiding: true,
  },
  {
    accessorKey: 'lastHeartbeatAt',
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title="Last seen"
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
    accessorKey: 'runtime',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="SDK Version" />
    ),
    cell: ({ row }) => <SdkInfo runtimeInfo={row.original.runtimeInfo} />,
    enableSorting: false,
    enableHiding: true,
  },
];
