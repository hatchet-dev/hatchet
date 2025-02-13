import { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '../../../../../components/molecules/data-table/data-table-column-header';
import { Link } from 'react-router-dom';
import { V2RunStatus } from '.././run-statuses';
import {
  AdditionalMetadata,
  AdditionalMetadataClick,
} from '../../../events/components/additional-metadata';
import RelativeDate from '@/components/molecules/relative-date';
import { Checkbox } from '@/components/ui/checkbox';
import { ListableWorkflowRun } from '../workflow-runs-table';

export const columns: (
  onAdditionalMetadataClick?: (click: AdditionalMetadataClick) => void,
) => ColumnDef<ListableWorkflowRun>[] = (onAdditionalMetadataClick) => [
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
          {row.original.parent.displayName || row.original.metadata.id}
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
    cell: ({ row }) => <V2RunStatus status={row.original.parent.status} />,
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: 'Workflow',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Workflow" />
    ),
    cell: ({ row }) => {
      const workflowId = row.original?.parent.workflowId;
      const workflowName = row.original.workflowName;

      return (
        <div className="min-w-fit whitespace-nowrap">
          {(workflowId && workflowName && (
            <a href={`/workflows/${workflowId}`}>{workflowName}</a>
          )) ||
            'N/A'}
        </div>
      );
    },
    show: false,
    enableSorting: false,
    enableHiding: true,
  },
  // TODO: Add this back
  // {
  //   accessorKey: 'Triggered by',
  //   header: ({ column }) => (
  //     <DataTableColumnHeader column={column} title="Triggered by" />
  //   ),
  //   cell: () => {
  //     const eventKey = 'N/A'; // FIXME: add back event keys, crons, etc

  //     return <div>{eventKey}</div>;
  //   },
  //   enableSorting: false,
  //   enableHiding: true,
  // },
  {
    accessorKey: 'Created at',
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title="Created at"
        className="whitespace-nowrap"
      />
    ),
    cell: ({ row }) => {
      return (
        <div className="whitespace-nowrap">
          {row.original.metadata.createdAt ? (
            <RelativeDate date={row.original.metadata.createdAt} />
          ) : (
            'N/A'
          )}
        </div>
      );
    },
    enableSorting: true,
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
          {row.original.parent.startedAt ? (
            <RelativeDate date={row.original.parent.startedAt} />
          ) : (
            'N/A'
          )}
        </div>
      );
    },
    enableSorting: true,
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
      const finishedAt = row.original.parent.finishedAt ? (
        <RelativeDate date={row.original.parent.finishedAt} />
      ) : (
        'N/A'
      );

      return <div className="whitespace-nowrap">{finishedAt}</div>;
    },
    enableSorting: true,
    enableHiding: true,
  },
  {
    accessorKey: 'Duration',
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title="Duration (ms)"
        className="whitespace-nowrap"
      />
    ),
    cell: ({ row }) => {
      return (
        <div className="whitespace-nowrap">{row.original.parent.duration}</div>
      );
    },
    enableSorting: true,
    enableHiding: true,
  },
  {
    accessorKey: 'Metadata',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Metadata" />
    ),
    cell: ({ row }) => {
      if (!row.original.parent.additionalMetadata) {
        return <div></div>;
      }

      return (
        <AdditionalMetadata
          metadata={row.original.parent.additionalMetadata}
          onClick={onAdditionalMetadataClick}
        />
      );
    },
    enableSorting: false,
  },
  // {
  //   id: "actions",
  //   cell: ({ row }) => <DataTableRowActions row={row} labels={[]} />,
  // },
];
