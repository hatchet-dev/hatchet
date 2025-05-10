import { RunDataCard } from "@/next/components/runs/run-output-card";
import { V1TaskSummary } from "@/lib/api/generated/data-contracts";

export type RunDetailRawContentProps = {
  selectedTask?: V1TaskSummary | null;
}

export const RunDetailRawContent = ({ selectedTask }: RunDetailRawContentProps) => {

    if (!selectedTask) {
      return null;
    }

    return (
      <>
        <RunDataCard
          title="Raw"
          output={selectedTask}
          variant="input"
        />
      </>
    );
  };
