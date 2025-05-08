import { TableCell, TableRow, TableBody, Table } from "@/next/components/ui/table";
import { RunsBadge } from "@/next/components/runs/runs-badge";
import { ExternalLink } from "lucide-react";
import { Link } from "react-router-dom";
import { V1TaskStatus } from "@/lib/api/generated/data-contracts";

export type TaskRunSummaryTableProps = {
    status: V1TaskStatus;
    detailsLink?: string;
    workflowRunId?: JSX.Element;
    taskRunId: JSX.Element;
  };
  
  
export const TaskRunSummaryTable = ({
    status,
    detailsLink,
    workflowRunId,
    taskRunId,
}: TaskRunSummaryTableProps) => {

    return (
      <div className="flex flex-col gap-2">
        <div className="flex flex-col gap-2">
          <Table className="text-sm table-fixed w-full">
            <TableBody>
              <TableRow className="border-none">
                <TableCell className="pr-4 text-muted-foreground w-[140px]">
                  Status
                </TableCell>
                <TableCell>
                  <RunsBadge status={status} variant="default" />
                </TableCell>
              </TableRow>
              {workflowRunId && (
                <TableRow className="border-none">
                  <TableCell className="pr-4 text-muted-foreground w-[140px]">
                    Workflow Run ID
                  </TableCell>
                <TableCell>
                  {workflowRunId}
                  </TableCell>
                </TableRow>
              )}
              <TableRow className="border-none">
                <TableCell className="pr-4 text-muted-foreground w-[140px]">
                  Task Run ID
                </TableCell>
                <TableCell>
                  {taskRunId}
                </TableCell>
              </TableRow>
              {detailsLink && (
                <TableRow className="border-none hover:cursor-pointer">
                  <TableCell className="pr-4 text-muted-foreground w-[140px]">
                    <Link
                      to={detailsLink}
                      className="text-sm text-muted-foreground hover:text-foreground transition-colors flex items-center gap-1"
                    >
                      View Run Details
                      <ExternalLink className="h-3 w-3" />
                    </Link>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      </div>
    );
  };