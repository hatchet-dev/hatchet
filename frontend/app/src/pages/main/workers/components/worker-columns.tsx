import { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '../../../../components/molecules/data-table/data-table-column-header';
import { Worker } from '@/lib/api';
import { relativeDate } from '@/lib/utils';
import { Link } from 'react-router-dom';

export const columns: ColumnDef<Worker>[] = [
  {
    accessorKey: 'name',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Name" />
    ),
    cell: ({ row }) => (
      <Link to={`/workers/${row.original.metadata.id}`}>
        <div className="cursor-pointer hover:underline min-w-fit whitespace-nowrap">
          {row.original.name}
        </div>
      </Link>
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
          {relativeDate(row.original.metadata.createdAt)}
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
          {relativeDate(row.original.lastHeartbeatAt)}
        </div>
      );
    },
    enableSorting: false,
    enableHiding: true,
  },
];
