import { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import { Worker } from '@/lib/api';
import { Link } from 'react-router-dom';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { SdkInfo } from './sdk-info';
import { WorkerStatusBadge } from '@/next/pages/authenticated/dashboard/workers/components';

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
