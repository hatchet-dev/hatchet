import { useMemo } from "react";
import { useRunDetail } from "@/next/hooks/use-run-detail";
import { Skeleton } from "@/next/components/ui/skeleton";
import WorkflowRunVisualizer from "@/next/components/runs/run-dag/dag-run-visualizer";
import { useSideSheet } from "@/next/hooks/use-side-sheet";
import { RunId } from "@/next/components/runs/run-id";
import { RunsBadge } from "@/next/components/runs/runs-badge";
import { Badge } from "@/next/components/ui/badge";
import { Duration } from "@/next/components/ui/duration";
export interface RunDetailSummaryProps {
    selectedTaskId?: string;
    detailsLink?: string;
}

export const TaskRunOverview = ({
    selectedTaskId,
  }: RunDetailSummaryProps) => {
    const { data, isLoading } = useRunDetail();
    const workflow = useMemo(() => data?.run, [data]);
  
    const task = useMemo(() => data?.tasks.find((t) => t.taskExternalId === selectedTaskId), [data, selectedTaskId]);

    const isDAG = data?.shape.length && data?.shape.length > 1;
    const { open: openSheet } = useSideSheet();

    if (isLoading || !workflow) {
      return (
        <div className="size-full flex flex-col my-2 gap-y-4">
          <Skeleton className="h-28 w-full" />
        </div>
      );
    }
  
    return <div className="w-full overflow-x-auto bg-slate-100 dark:bg-slate-900 ">
      <div className="flex flex-row  px-8 py-4 gap-2 items-center">
      <RunsBadge status={workflow.status} variant="xs" />
      <RunId wfRun={workflow} onClick={()=>{
        openSheet({
          type: 'task-detail',
          props: {  
            selectedWorkflowRunId: workflow.metadata.id,
          },
        });
      }}/>{isDAG ? selectedTaskId && <>/<RunId wfRun={task} onClick={()=>{}}/> <Badge variant="outline" className="ml-auto">DAG</Badge></> : <><Badge variant="outline" className="ml-auto">Standalone</Badge></>}
      </div>
      {<WorkflowRunVisualizer
        workflowRunId={workflow.metadata.id}
          onTaskSelect={(taskId) => {
            openSheet({
              type: 'task-detail',
              props: {  
                selectedWorkflowRunId: workflow.metadata.id,
                selectedTaskId: taskId,
              },
            });
          }}
          selectedTaskId={isDAG && selectedTaskId === workflow.metadata.id ? undefined : selectedTaskId}
        />}
      </div> 

  };
  