import { useMemo } from "react";
import { useRunDetail } from "@/next/hooks/use-run-detail";
import { Skeleton } from "@/next/components/ui/skeleton";
import { RunId } from "@/next/components/runs/run-id";
import { TaskRunSummaryTable } from "./run-detail-table";

export interface RunDetailSummaryProps {
    selectedTaskId?: string;
    detailsLink?: string;
}

export const TaskRunOverview = ({
    selectedTaskId,
    detailsLink,
  }: RunDetailSummaryProps) => {
    const { data, isLoading } = useRunDetail();
    const workflow = useMemo(() => data?.run, [data]);
    const tasks = useMemo(() => data?.tasks, [data]);
  
    const selectedTask = useMemo(() => {
      if (selectedTaskId) {
        return tasks?.find((t) => t.taskExternalId === selectedTaskId);
      }
      // TODO: Add payload content for dag 
      return tasks?.[0];
    }, [tasks, selectedTaskId]);
  
    if (isLoading || !workflow) {
      return (
        <div className="size-full flex flex-col my-2 gap-y-4">
          <Skeleton className="h-28 w-full" />
          <div className="h-4" />
          <Skeleton className="h-36 w-full" />
          <Skeleton className="h-36 w-full" />
          <Skeleton className="h-36 w-full" />
        </div>
      );
    }
  
    if (!selectedTask) {
      return (
        <TaskRunSummaryTable
          status={workflow.status}
          detailsLink={detailsLink}
          runIdElement={<RunId wfRun={workflow} />}
        />
      );
    }
  
    return (
      <TaskRunSummaryTable
        status={selectedTask.status}
        detailsLink={detailsLink}
        runIdElement={<RunId taskRun={selectedTask} />}
      />
    );
  };
  