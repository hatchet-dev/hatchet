import { Icons } from "@/components/ui/icons";
import { Separator } from "@/components/ui/separator";
import { queries } from "@/lib/api";
import { currTenantAtom } from "@/lib/atoms";
import { useQuery } from "@tanstack/react-query";
import { useAtom } from "jotai";
import { Link, useParams } from "react-router-dom";
import invariant from "tiny-invariant";
import { Badge } from "@/components/ui/badge";
import { relativeDate } from "@/lib/utils";
import {
  AdjustmentsHorizontalIcon,
  BoltIcon,
  Square3Stack3DIcon,
} from "@heroicons/react/24/outline";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/molecules/data-table/data-table";
import { JobRunColumns, columns } from "./components/job-runs-columns";
import { TableCell, TableRow } from "@/components/ui/table";
import { RunStatus } from "../components/run-statuses";

export default function ExpandedWorkflowRun() {
  const [tenant] = useAtom(currTenantAtom);
  invariant(tenant);

  const params = useParams();
  invariant(params.run);

  const runQuery = useQuery({
    ...queries.workflowRuns.get(tenant.metadata.id, params.run),
  });

  if (runQuery.isLoading || !runQuery.data) {
    return (
      <div className="flex flex-row flex-1 w-full h-full">
        <Icons.spinner className="mr-2 h-4 w-4 animate-spin" />
      </div>
    );
  }

  const run = runQuery.data;

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <div className="flex flex-row gap-4 items-center">
            <AdjustmentsHorizontalIcon className="h-6 w-6 text-foreground mt-1" />
            <h2 className="text-2xl font-bold leading-tight tracking-tight text-foreground">
              {run?.metadata.id}
            </h2>
            <Badge className="text-sm mt-1" variant={"secondary"}>
              {/* {workflow.versions && workflow.versions[0].version} */}
              {run.status}
            </Badge>
          </div>
        </div>
        <div className="flex flex-row justify-start items-center mt-4 gap-2">
          {run?.workflowVersion?.workflow && (
            <Link
              to={`/workflows/${run?.workflowVersion?.workflow?.metadata.id}`}
            >
              <Button
                variant="ghost"
                className="flex flex-row items-center gap-2 text-sm text-foreground hover:bg-muted"
              >
                <Square3Stack3DIcon className="h-4 w-4" />
                {run?.workflowVersion?.workflow?.name}
              </Button>
            </Link>
          )}
          <div className="text-sm text-muted-foreground">
            Created {relativeDate(run?.metadata.createdAt)}
          </div>
          {run?.startedAt && (
            <div className="text-sm text-muted-foreground">
              Started {relativeDate(run?.startedAt)}
            </div>
          )}
          {run?.finishedAt && (
            <div className="text-sm text-muted-foreground">
              Finished {relativeDate(run?.startedAt)}
            </div>
          )}
        </div>
        <Separator className="my-4" />
        <h3 className="text-xl font-bold leading-tight tracking-tight text-foreground mb-4">
          Job Runs
        </h3>
        <DataTable
          columns={columns}
          data={
            run.jobRuns
              ?.map((jobRun): JobRunColumns[] => {
                return [
                  {
                    kind: "job",
                    isGroupingRow: true,
                    getGroupingRow: () => {
                      return (
                        <TableRow key={jobRun.metadata.id} className="bg-muted">
                          <TableCell colSpan={1}>
                            <div className="flex flex-row gap-2 items-center justify-start">
                              <BoltIcon className="h-4 w-4" />
                              {jobRun.job?.name}
                            </div>
                          </TableCell>
                          <TableCell colSpan={1}>
                            <RunStatus status={jobRun.status} />
                          </TableCell>
                          <TableCell colSpan={1}>
                            <div>{relativeDate(jobRun.startedAt)}</div>
                          </TableCell>
                          <TableCell colSpan={1}>
                            <div>{relativeDate(jobRun.finishedAt)}</div>
                          </TableCell>
                          <TableCell colSpan={columns.length - 4} />
                        </TableRow>
                      );
                    },
                    ...jobRun,
                  },
                  ...(jobRun.stepRuns?.map((stepRun): JobRunColumns => {
                    return {
                      kind: "step",
                      ...stepRun,
                    };
                  }) || []),
                ];
              })
              .flat() || []
          }
          filters={[]}
        />
      </div>
    </div>
  );
}
