import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/molecules/data-table/data-table-column-header";
import { DataTableRowActions } from "@/components/molecules/data-table/data-table-row-actions";
import { StepRun } from "@/lib/api";
import { relativeDate } from "@/lib/utils";
import { Link } from "react-router-dom";
import { RunStatus } from "@/pages/main/workflow-runs/components/run-statuses";

export const columns: ColumnDef<StepRun>[] = [
  {
    accessorKey: "id",
    header: () => <></>,
    cell: ({ row }) => {
      return (
        <Link to={"/workflow-runs/" + row.original.metadata.id}>
          <div className="cursor-pointer hover:underline min-w-fit whitespace-nowrap ml-6">
            {row.original.step?.readableId}
          </div>
        </Link>
      );
    },
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: "status",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Status" />
    ),
    cell: ({ row }) => <RunStatus status={row.original.status} />,
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: "Started at",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Started at" />
    ),
    cell: ({ row }) => {
      return <div>{relativeDate(row.original.startedAt)}</div>;
    },
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: "Finished at",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Finished at" />
    ),
    cell: ({ row }) => {
      const finishedAt = row.original.finishedAt
        ? relativeDate(row.original.finishedAt)
        : "N/A";

      return <div>{finishedAt}</div>;
    },
    enableSorting: false,
    enableHiding: false,
  },
  {
    id: "actions",
    cell: ({ row }) => <DataTableRowActions row={row} labels={[]} />,
  },
];
