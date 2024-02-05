import { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '../../../../components/molecules/data-table/data-table-column-header';
import { WorkflowRun } from '@/lib/api';
import { relativeDate } from '@/lib/utils';
import { Link } from 'react-router-dom';
import { RunStatus } from './run-statuses';

export const columns: ColumnDef<WorkflowRun>[] = [
  {
    accessorKey: 'id',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Run Id" />
    ),
    cell: ({ row }) => (
      <Link to={'/workflow-runs/' + row.original.metadata.id}>
        <div className="cursor-pointer hover:underline min-w-fit whitespace-nowrap">
          {row.original.displayName || row.original.metadata.id}
        </div>
      </Link>
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: 'status',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Status" />
    ),
    cell: ({ row }) => <RunStatus status={row.original.status} />,
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: 'Workflow',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Workflow" />
    ),
    cell: ({ row }) => {
      const workflowName =
        row.original.workflowVersion?.workflow?.name || 'N/A';

      return <div className="min-w-fit whitespace-nowrap">{workflowName}</div>;
    },
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: 'Triggered by',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Triggered by" />
    ),
    cell: ({ row }) => {
      const eventKey = row.original.triggeredBy?.event?.key || 'N/A';

      return <div>{eventKey}</div>;
    },
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: 'Started at',
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
          {relativeDate(row.original.startedAt)}
        </div>
      );
    },
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: 'Finished at',
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title="Finished at"
        className="whitespace-nowrap"
      />
    ),
    cell: ({ row }) => {
      const finishedAt = row.original.finishedAt
        ? relativeDate(row.original.finishedAt)
        : 'N/A';

      return <div className="whitespace-nowrap">{finishedAt}</div>;
    },
    enableSorting: false,
    enableHiding: false,
  },
  // {
  //   id: "actions",
  //   cell: ({ row }) => <DataTableRowActions row={row} labels={[]} />,
  // },
];
