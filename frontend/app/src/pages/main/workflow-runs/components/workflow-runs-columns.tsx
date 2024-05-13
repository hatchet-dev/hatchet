import { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '../../../../components/molecules/data-table/data-table-column-header';
import { WorkflowRun } from '@/lib/api';
import { Link } from 'react-router-dom';
import { RunStatus } from './run-statuses';
import { AdditionalMetadata } from '../../events/components/additional-metadata';
import RelativeDate from '@/components/molecules/relative-date';
import { Checkbox } from '@/components/ui/checkbox';

export const columns: ColumnDef<WorkflowRun>[] = [
  {
    id: 'select',
    header: ({ table }) => (
      <Checkbox
        checked={
          table.getIsAllPageRowsSelected() ||
          (table.getIsSomePageRowsSelected() && 'indeterminate')
        }
        onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
        aria-label="Select all"
        className="translate-y-[2px]"
      />
    ),
    cell: ({ row }) => (
      <Checkbox
        checked={row.getIsSelected()}
        onCheckedChange={(value) => row.toggleSelected(!!value)}
        aria-label="Select row"
        className="translate-y-[2px]"
      />
    ),
    enableSorting: false,
    enableHiding: false,
  },
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
      const workflow = row.original.workflowVersion?.workflow;
      const workflowName = workflow?.name;
      const workflowId = workflow?.metadata.id;

      return (
        <div className="min-w-fit whitespace-nowrap">
          {(workflow && (
            <a href={`/workflows/${workflowId}`}>{workflowName}</a>
          )) ||
            'N/A'}
        </div>
      );
    },
    enableSorting: false,
    enableHiding: true,
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
    enableHiding: true,
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
          {row.original.startedAt ? (
            <RelativeDate date={row.original.startedAt} />
          ) : (
            'N/A'
          )}
        </div>
      );
    },
    enableSorting: false,
    enableHiding: true,
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
      const finishedAt = row.original.finishedAt ? (
        <RelativeDate date={row.original.finishedAt} />
      ) : (
        'N/A'
      );

      return <div className="whitespace-nowrap">{finishedAt}</div>;
    },
    enableSorting: false,
    enableHiding: true,
  },
  {
    accessorKey: 'Metadata',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Metadata" />
    ),
    cell: ({ row }) => {
      if (!row.original.additionalMetadata) {
        return <div></div>;
      }

      return <AdditionalMetadata metadata={row.original.additionalMetadata} />;
    },
    enableSorting: false,
  },
  // {
  //   id: "actions",
  //   cell: ({ row }) => <DataTableRowActions row={row} labels={[]} />,
  // },
];
