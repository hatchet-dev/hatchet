import { TableCell, TableRow, TableBody, Table } from "@/next/components/ui/table";
import { RunsBadge } from "@/next/components/runs/runs-badge";
import { ExternalLink } from "lucide-react";
import { Link } from "react-router-dom";
import { V1TaskStatus } from "@/lib/api/generated/data-contracts";

export type TaskRunSummaryTableProps = {
    status: V1TaskStatus;
    detailsLink?: string;
    runIdElement: JSX.Element;
  };
  
  
export const TaskRunSummaryTable = ({
    status,
    detailsLink,
    runIdElement,
}: TaskRunSummaryTableProps) => {
    return (
      <div className="flex flex-col gap-2">
        <div className="flex flex-col gap-2">
          <Table className="text-sm">
            <TableBody>
              <TableRow className="border-none">
                <TableCell className="pr-4 text-muted-foreground">
                  Status
                </TableCell>
                <TableCell className="flex flex-row items-center justify-center">
                  <RunsBadge status={status} variant="default" />
                </TableCell>
              </TableRow>
              <TableRow className="border-none">
                <TableCell className="pr-4 text-muted-foreground">
                  Task ID
                </TableCell>
                <TableCell className="flex flex-row items-center justify-center">
                  {runIdElement}
                </TableCell>
              </TableRow>
              {detailsLink && (
                <TableRow className="border-none hover:cursor-pointer">
                  <TableCell className="pr-4 text-muted-foreground">
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