import { useMemo } from "react";
import { RunDataCard } from "@/next/components/runs/run-output-card";
import { useRunDetail } from "@/next/hooks/use-run-detail";
import { V1WorkflowType } from "@/lib/api/generated/data-contracts";
import { DocsButton } from "@/next/components/ui/docs-button";
import docs from "@/next/lib/docs";

export type RunDetailPayloadContentProps = {
  selectedTaskId?: string;
}

export const RunDetailPayloadContent = ({ selectedTaskId }: RunDetailPayloadContentProps) => {
    const { data, isLoading } = useRunDetail();
    const workflow = useMemo(() => data?.run, [data]);
    const tasks = useMemo(() => data?.tasks, [data]);
  
    const selectedTask = useMemo(() => {
      if (selectedTaskId) {
        return tasks?.find((t) => t.taskExternalId === selectedTaskId);
      }
      return tasks?.[0];
    }, [tasks, selectedTaskId]);
  
    if (isLoading || !workflow) {
      return (
        <>
          <RunDataCard
            title="Input"
            output={(workflow?.input as { input: object })?.input ?? {}}
            status={workflow?.status}
            variant="input"
          />
          <RunDataCard
            title="Metadata"
            output={{
              workflowRunId: workflow?.metadata.id,
              additional: workflow?.additionalMetadata,
            }}
            status={workflow?.status}
            variant="metadata"
            collapsed
            actions={
              <div className="flex items-center gap-2">
                <DocsButton doc={docs.home.additional_metadata} size="icon" />
              </div>
            }
          />
        </>
      );
    }
  
    if (!selectedTask) {
      return null;
    }
  
    return (
      <>
        <RunDataCard
          title="Input"
          output={selectedTask}
          variant="input"
        />
        <RunDataCard
          title="Input"
          output={(selectedTask.input as any).input ?? {}}
          variant="input"
        />
        {selectedTask.type === V1WorkflowType.DAG && (
          <RunDataCard
            title="Parent Data"
            output={(selectedTask.input as any).parents ?? {}}
            variant="input"
            collapsed
          />
        )}
        <RunDataCard
          title="Output"
          output={selectedTask.output}
          error={selectedTask.errorMessage}
          status={selectedTask.status}
          variant="output"
        />
        <RunDataCard
          title="Metadata"
          output={{
            taskRunId: selectedTask.metadata.id,
            workflowRunId: workflow?.metadata.id,
            additional: selectedTask.additionalMetadata,
          }}
          status={selectedTask.status}
          variant="metadata"
          collapsed
          actions={
            <div className="flex items-center gap-2">
              <DocsButton doc={docs.home.additional_metadata} size="icon" />
            </div>
          }
        />
      </>
    );
  };