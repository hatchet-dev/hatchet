import { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import { JobRun, StepRun } from '@/lib/api';
import { relativeDate } from '@/lib/utils';
import { RunStatus } from '../../components/run-statuses';
import RelativeDate from '@/components/v1/molecules/relative-date';

type JobRunRow = {
  kind: 'job';
} & JobRun;

type StepRunRow = {
  kind: 'step';
  onClick?: () => void;
} & StepRun;

export type JobRunColumns = (JobRunRow | StepRunRow) & {
  isExpandable?: boolean;
  getRow?: () => JSX.Element;
};

export const columns: ColumnDef<JobRunColumns>[] = [
  {
    accessorKey: 'id',
    header: () => <></>,
    cell: ({ row }) => {
      if (row.original.kind === 'job') {
        return <></>;
      }
      return (
        <div className="min-w-fit whitespace-nowrap ml-6">
          {row.original.step?.readableId}
        </div>
      );
    },
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: 'status',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Status" />
    ),
    cell: ({ row }) => {
      let reason;

      if (row.original.kind == 'step') {
        reason = row.original.cancelledReason;
      }

      return <RunStatus status={row.original.status} reason={reason} />;
    },
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: 'Started at',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Started at" />
    ),
    cell: ({ row }) => {
      return (
        <div>
          {row.original.startedAt && (
            <RelativeDate date={row.original.startedAt} />
          )}
        </div>
      );
    },
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: 'Finished at',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Finished at" />
    ),
    cell: ({ row }) => {
      const finishedAt = row.original.finishedAt
        ? relativeDate(row.original.finishedAt)
        : 'N/A';

      return <div>{finishedAt}</div>;
    },
    enableSorting: false,
    enableHiding: false,
  },
  // {
  //   id: "actions",
  //   cell: ({ row }) => <DataTableRowActions row={row} labels={[]} />,
  // },
];
