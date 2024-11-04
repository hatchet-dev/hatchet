import { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '../../../../components/molecules/data-table/data-table-column-header';
import { CronWorkflows } from '@/lib/api';
import CronPrettifier from 'cronstrue';
import RelativeDate from '@/components/molecules/relative-date';
import { Link } from 'react-router-dom';
import { DataTableRowActions } from '@/components/molecules/data-table/data-table-row-actions';

export const columns = ({
  onDeleteClick,
}: {
  onDeleteClick: (row: CronWorkflows) => void;
}): ColumnDef<CronWorkflows>[] => {
  return [
    {
      accessorKey: 'crons',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Cron" />
      ),
      cell: ({ row }) => (
        <div className="flex flex-row items-center gap-4 pl-4">
          {row.original.cron}
        </div>
      ),
      enableSorting: false,
    },
    {
      accessorKey: 'name',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Name" />
      ),
      cell: ({ row }) => <div>{row.original.name }</div>,
    },
    {
      accessorKey: 'readable',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Readable" />
      ),
      cell: ({ row }) => (
        <div className="flex flex-row items-center gap-4 pl-4">
          (runs {CronPrettifier.toString(row.original.cron).toLowerCase()} UTC)
        </div>
      ),
      enableSorting: false,
    },
    {
      accessorKey: 'Workflow',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Workflow" />
      ),
      cell: ({ row }) => (
        <div className="flex flex-row items-center gap-4 pl-4">
          <div className="cursor-pointer hover:underline min-w-fit whitespace-nowrap">
            <Link to={`/workflows/${row.original.workflowId}`}>
              {row.original.workflowName}
            </Link>
          </div>
        </div>
      ),
      enableSorting: false,
      enableHiding: true,
    },
    // {
    //   accessorKey: 'Metadata',
    //   header: ({ column }) => (
    //     <DataTableColumnHeader column={column} title="Metadata" />
    //   ),
    //   cell: ({ row }) => {
    //     if (!row.original.additionalMetadata) {
    //       return <div></div>;
    //     }

    //     return <AdditionalMetadata metadata={row.original.additionalMetadata} />;
    //   },
    //   enableSorting: false,
    // },
    {
      accessorKey: 'createdAt',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Created At" />
      ),
      cell: ({ row }) => (
        <div className="flex flex-row items-center gap-4 pl-4">
          <RelativeDate date={row.original.metadata.createdAt} />
        </div>
      ),
      enableSorting: true,
      enableHiding: true,
    },
    {
      accessorKey: 'actions',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Actions" />
      ),
      cell: ({ row }) => (
        <div className="flex flex-row justify-center">
          <DataTableRowActions
            row={row}
            actions={[
              {
                label: 'Delete',
                onClick: () => onDeleteClick(row.original),
              },
            ]}
          />
        </div>
      ),
      enableHiding: true,
      enableSorting: false,
    },
  ];
};
