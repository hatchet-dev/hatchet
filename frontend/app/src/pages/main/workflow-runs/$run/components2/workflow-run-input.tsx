import { CodeEditor } from '@/components/ui/code-editor';
import { Loading } from '@/components/ui/loading';
import { WorkflowRun, queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';

export function WorkflowRunInputDialog({ run }: { run: WorkflowRun }) {
  const getInputQuery = useQuery({
    ...queries.workflowRuns.getInput(run.tenantId, run.metadata.id),
  });

  if (getInputQuery.isLoading) {
    return <Loading />;
  }

  if (!getInputQuery.data) {
    return null;
  }

  const input = getInputQuery.data;

  return (
    <>
      <CodeEditor
        language="json"
        className="my-4"
        height="400px"
        code={JSON.stringify(input, null, 2)}
      />
    </>
  );
}
