import { RunDataCard } from '@/next/components/runs/run-output-card';
import { useRunDetail } from '@/next/hooks/use-run-detail';
import { V1WorkflowRun } from '@/lib/api/generated/data-contracts';
import { DocsButton } from '@/next/components/ui/docs-button';
import docs from '@/next/lib/docs';

export type WorkflowRunDetailPayloadContentProps = {
  workflowRun?: V1WorkflowRun | null;
};

export const WorkflowRunDetailPayloadContent = ({
  workflowRun,
}: WorkflowRunDetailPayloadContentProps) => {
  const { isLoading } = useRunDetail();

  if (isLoading) {
    return (
      <>
        <RunDataCard
          title="Input"
          output={workflowRun?.input}
          variant="input"
        />
        <RunDataCard
          title="Metadata"
          output={{
            workflowRunId: workflowRun?.metadata.id,
            additional: workflowRun?.additionalMetadata,
          }}
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

  if (!workflowRun) {
    return null;
  }

  return (
    <>
      <RunDataCard title="Input" output={workflowRun?.input} variant="input" />
      <RunDataCard
        title="Output"
        output={workflowRun?.output}
        error={workflowRun?.errorMessage}
        status={workflowRun?.status}
        variant="output"
      />
      <RunDataCard
        title="Metadata"
        output={{
          workflowRunId: workflowRun?.metadata.id,
          additional: workflowRun?.additionalMetadata,
        }}
        status={workflowRun?.status}
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
